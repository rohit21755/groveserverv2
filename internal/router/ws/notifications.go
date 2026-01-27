package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// SendNotification sends a notification to a specific user via WebSocket
// This is a generic function that can be reused for all notification types
func SendNotification(hub *Hub, userID string, notificationType NotificationType, title, message string, data map[string]interface{}) error {
	if hub == nil {
		return fmt.Errorf("hub is nil")
	}

	notification := NotificationPayload{
		ID:        uuid.New().String(),
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Data:      data,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return hub.SendNotification(userID, notification)
}

// SendNotificationToMultiple sends a notification to multiple users
func SendNotificationToMultiple(hub *Hub, userIDs []string, notificationType NotificationType, title, message string, data map[string]interface{}) error {
	if hub == nil {
		return fmt.Errorf("hub is nil")
	}

	notification := NotificationPayload{
		ID:        uuid.New().String(),
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Data:      data,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	hub.SendNotificationToMultiple(userIDs, notification)
	return nil
}

// SendTaskAssignmentNotification sends a notification when a task is assigned
func SendTaskAssignmentNotification(hub *Hub, userIDs []string, taskID, taskTitle, taskDescription string) error {
	data := map[string]interface{}{
		"task_id":    taskID,
		"task_title": taskTitle,
	}

	title := "New Task Assigned"
	message := fmt.Sprintf("You have been assigned a new task: %s", taskTitle)

	return SendNotificationToMultiple(hub, userIDs, NotificationTypeTaskAssigned, title, message, data)
}

// SendTaskApprovalNotification sends a notification when a task is approved
func SendTaskApprovalNotification(hub *Hub, userID, taskID, taskTitle string, xpAwarded int) error {
	data := map[string]interface{}{
		"task_id":     taskID,
		"task_title":  taskTitle,
		"xp_awarded":  xpAwarded,
	}

	title := "Task Approved"
	message := fmt.Sprintf("Your task '%s' has been approved! You earned %d XP.", taskTitle, xpAwarded)

	return SendNotification(hub, userID, NotificationTypeTaskApproved, title, message, data)
}

// SendTaskRejectionNotification sends a notification when a task is rejected
func SendTaskRejectionNotification(hub *Hub, userID, taskID, taskTitle, rejectionComment string) error {
	data := map[string]interface{}{
		"task_id":           taskID,
		"task_title":        taskTitle,
		"rejection_comment": rejectionComment,
	}

	title := "Task Rejected"
	message := fmt.Sprintf("Your task '%s' has been rejected. Comment: %s", taskTitle, rejectionComment)

	return SendNotification(hub, userID, NotificationTypeTaskRejected, title, message, data)
}

// SendTaskUpdateNotification sends a notification when a task is updated
func SendTaskUpdateNotification(hub *Hub, userIDs []string, taskID, taskTitle string) error {
	data := map[string]interface{}{
		"task_id":    taskID,
		"task_title": taskTitle,
	}

	title := "Task Updated"
	message := fmt.Sprintf("Task '%s' has been updated", taskTitle)

	return SendNotificationToMultiple(hub, userIDs, NotificationTypeTaskAssigned, title, message, data)
}

// PublishNotificationToRedis publishes a notification to Redis for distribution
func PublishNotificationToRedis(hub *Hub, userID string, notification NotificationPayload) error {
	if hub == nil || hub.redisClient == nil {
		return fmt.Errorf("hub or redis client is nil")
	}

	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	ctx := context.Background()
	err = hub.redisClient.Client.Publish(ctx, "notifications", notificationBytes).Err()
	if err != nil {
		return fmt.Errorf("failed to publish notification to Redis: %w", err)
	}

	log.Printf("Published notification to Redis for user %s: %s", userID, notification.Type)
	return nil
}
