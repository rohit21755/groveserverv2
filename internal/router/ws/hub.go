package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rohit21755/groveserverv2/internal/db"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeNotification MessageType = "notification"
	MessageTypeChat         MessageType = "chat"
	MessageTypeLeaderboard  MessageType = "leaderboard"
	MessageTypeTask         MessageType = "task"
	MessageTypeSystem       MessageType = "system"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeTaskAssigned NotificationType = "task_assigned"
	NotificationTypeTaskApproved NotificationType = "task_approved"
	NotificationTypeTaskRejected NotificationType = "task_rejected"
	NotificationTypeNewFollower  NotificationType = "new_follower"
	NotificationTypeNewComment   NotificationType = "new_comment"
	NotificationTypeNewReaction   NotificationType = "new_reaction"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    MessageType       `json:"type"`
	Payload json.RawMessage   `json:"payload"`
	Data    interface{}       `json:"data,omitempty"` // For backward compatibility
}

// NotificationPayload represents a notification message
type NotificationPayload struct {
	ID        string           `json:"id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Data      interface{}      `json:"data,omitempty"` // Additional data (task_id, user_id, etc.)
	CreatedAt string           `json:"created_at"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string          // User ID
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	UserID   string
	UserRole string
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients mapped by user ID
	clients map[string]*Client

	// Broadcast channel for all clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread safety
	mu sync.RWMutex

	// Redis client for pub/sub
	redisClient *db.Redis

	// Postgres for database operations
	postgres *db.Postgres
}

// NewHub creates a new WebSocket hub
func NewHub(redisClient *db.Redis, postgres *db.Postgres) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		redisClient: redisClient,
		postgres:    postgres,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	// Subscribe to Redis pub/sub for notifications
	go h.subscribeToNotifications()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Remove old connection if exists for this user
			if oldClient, exists := h.clients[client.UserID]; exists {
				close(oldClient.Send)
				delete(h.clients, client.UserID)
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("WebSocket client connected: user_id=%s, role=%s", client.UserID, client.UserRole)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: user_id=%s", client.UserID)

		case message := <-h.broadcast:
			// Broadcast to all connected clients
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.UserID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendNotification sends a notification to a specific user
func (h *Hub) SendNotification(userID string, notification NotificationPayload) error {
	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		// User not connected, store notification in database for later retrieval
		// TODO: Store notification in database
		log.Printf("User %s not connected, notification will be stored in database", userID)
		return nil
	}

	// Create message
	message := WSMessage{
		Type: MessageTypeNotification,
		Data: notification,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case client.Send <- messageBytes:
		log.Printf("Notification sent to user %s: %s", userID, notification.Type)
	default:
		log.Printf("Failed to send notification to user %s: channel full", userID)
	}

	return nil
}

// SendNotificationToMultiple sends a notification to multiple users
func (h *Hub) SendNotificationToMultiple(userIDs []string, notification NotificationPayload) {
	for _, userID := range userIDs {
		h.SendNotification(userID, notification)
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *Hub) BroadcastMessage(messageType MessageType, data interface{}) error {
	message := WSMessage{
		Type: messageType,
		Data: data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.broadcast <- messageBytes
	return nil
}

// subscribeToNotifications subscribes to Redis pub/sub for notifications
func (h *Hub) subscribeToNotifications() {
	ctx := context.Background()
	pubsub := h.redisClient.Client.Subscribe(ctx, "notifications")

	ch := pubsub.Channel()
	for msg := range ch {
		var notification NotificationPayload
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			log.Printf("Error unmarshaling notification: %v", err)
			continue
		}

		// Extract user ID from notification data
		// The notification payload should contain target user ID
		if notificationData, ok := notification.Data.(map[string]interface{}); ok {
			if userID, ok := notificationData["user_id"].(string); ok {
				h.SendNotification(userID, notification)
			}
		}
	}
}
