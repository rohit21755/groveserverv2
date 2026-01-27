package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/router/ws"
	"github.com/rohit21755/groveserverv2/internal/storage"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	XP          int        `json:"xp"`
	Type        string     `json:"type"`
	ProofType   string     `json:"proof_type"`
	Priority    string     `json:"priority"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	IsFlash     bool       `json:"is_flash"`
	IsWeekly    bool       `json:"is_weekly"`
	// Assignment fields
	AssignmentType store.AssignmentType `json:"assignment_type"`         // "all", "state", "college", "user"
	AssignmentID   string               `json:"assignment_id,omitempty"` // State ID, College ID, or User ID (empty for "all")
}

// CreateTaskResponse represents the response after creating a task
type CreateTaskResponse struct {
	Task       *store.Task `json:"task"`
	AssignedTo int         `json:"assigned_to"` // Number of users the task was assigned to
}

// handleCreateTask handles creating a new task (admin)
// @Summary      Create task
// @Description  Create a new task and assign it to users. Can be assigned to all users, users from a state, users from a college, or a single user.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        task  body      CreateTaskRequest  true  "Task information and assignment details"
// @Success      201   {object}  CreateTaskResponse  "Task created successfully"
// @Failure      400   {string}  string  "Bad request - invalid input"
// @Failure      401   {string}  string  "Unauthorized"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /admin/tasks [post]
func handleCreateTask(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request body
		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding create task request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Title == "" || req.Description == "" || req.Type == "" || req.ProofType == "" {
			http.Error(w, "Missing required fields: title, description, type, proof_type are required", http.StatusBadRequest)
			return
		}

		// Validate assignment type
		if req.AssignmentType != store.AssignmentAll &&
			req.AssignmentType != store.AssignmentState &&
			req.AssignmentType != store.AssignmentCollege &&
			req.AssignmentType != store.AssignmentUser {
			http.Error(w, "Invalid assignment_type. Must be one of: all, state, college, user", http.StatusBadRequest)
			return
		}

		// Validate assignment ID is provided when needed
		if req.AssignmentType != store.AssignmentAll && req.AssignmentID == "" {
			http.Error(w, "assignment_id is required when assignment_type is not 'all'", http.StatusBadRequest)
			return
		}

		// Get admin user ID from context (set by JWT middleware)
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Admin user ID not found in context. Please ensure you are authenticated.", http.StatusUnauthorized)
			return
		}
		log.Printf("Admin user ID: %s", adminUserID)

		// Verify admin exists in admins table
		adminStore := store.NewAdminStore(postgres)
		_, err := adminStore.GetAdminByID(ctx, adminUserID)
		if err != nil {
			log.Printf("Error verifying admin: %v", err)
			http.Error(w, "Admin not found. Please use a valid admin account.", http.StatusUnauthorized)
			return
		}

		// Create task store
		taskStore := store.NewTaskStore(postgres)

		// Prepare task creation request
		createReq := store.CreateTaskRequest{
			Title:       req.Title,
			Description: req.Description,
			XP:          req.XP,
			Type:        req.Type,
			ProofType:   req.ProofType,
			Priority:    req.Priority,
			StartAt:     req.StartAt,
			EndAt:       req.EndAt,
			IsFlash:     req.IsFlash,
			IsWeekly:    req.IsWeekly,
			CreatedBy:   adminUserID,
		}

		// Set default priority if not provided
		if createReq.Priority == "" {
			createReq.Priority = "normal"
		}

		// Create task and get assigned user IDs
		task, assignedUserIDs, err := taskStore.CreateTask(ctx, createReq, req.AssignmentType, req.AssignmentID)
		if err != nil {
			log.Printf("Error creating task: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create task: %v", err), http.StatusInternalServerError)
			return
		}

		// Send WebSocket notifications to all assigned users
		wsHub := ws.GetHub()
		if wsHub != nil && len(assignedUserIDs) > 0 {
			err = ws.SendTaskAssignmentNotification(wsHub, assignedUserIDs, task.ID, task.Title, task.Description)
			if err != nil {
				log.Printf("Error sending task assignment notifications: %v", err)
				// Don't fail the request if notification fails
			} else {
				log.Printf("Sent task assignment notifications to %d users", len(assignedUserIDs))
			}
		}

		// Return response
		response := CreateTaskResponse{
			Task:       task,
			AssignedTo: len(assignedUserIDs),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding create task response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	XP          *int       `json:"xp,omitempty"`
	Type        *string    `json:"type,omitempty"`
	ProofType   *string    `json:"proof_type,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	IsFlash     *bool      `json:"is_flash,omitempty"`
	IsWeekly    *bool      `json:"is_weekly,omitempty"`
}

// handleUpdateTask handles updating a task (admin)
// @Summary      Update task
// @Description  Update an existing task. Admin only. Sends notifications to assigned users.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string            true   "Task ID"
// @Param        request  body      UpdateTaskRequest  true   "Task update fields"
// @Success      200      {object}  store.Task  "Task updated successfully"
// @Failure      400      {string}  string  "Bad request"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      404      {string}  string  "Task not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /admin/tasks/{id} [put]
func handleUpdateTask(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get task ID from URL path
		taskID := chi.URLParam(r, "id")
		if taskID == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		// Get admin user ID from context
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Admin user ID not found in context", http.StatusUnauthorized)
			return
		}

		// Verify admin exists
		adminStore := store.NewAdminStore(postgres)
		_, err := adminStore.GetAdminByID(ctx, adminUserID)
		if err != nil {
			log.Printf("Error verifying admin: %v", err)
			http.Error(w, "Admin not found", http.StatusUnauthorized)
			return
		}

		// Parse request body
		var req UpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding update task request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify task exists
		taskStore := store.NewTaskStore(postgres)
		_, err = taskStore.GetTaskByID(ctx, taskID)
		if err != nil {
			log.Printf("Error getting task: %v", err)
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		// Update task (we'll need to add UpdateTask method to TaskStore)
		// For now, we'll use a simple SQL update
		updateFields := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.Title != nil {
			updateFields = append(updateFields, fmt.Sprintf("title = $%d", argIndex))
			args = append(args, *req.Title)
			argIndex++
		}
		if req.Description != nil {
			updateFields = append(updateFields, fmt.Sprintf("description = $%d", argIndex))
			args = append(args, *req.Description)
			argIndex++
		}
		if req.XP != nil {
			updateFields = append(updateFields, fmt.Sprintf("xp = $%d", argIndex))
			args = append(args, *req.XP)
			argIndex++
		}
		if req.Type != nil {
			updateFields = append(updateFields, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, *req.Type)
			argIndex++
		}
		if req.ProofType != nil {
			updateFields = append(updateFields, fmt.Sprintf("proof_type = $%d", argIndex))
			args = append(args, *req.ProofType)
			argIndex++
		}
		if req.Priority != nil {
			updateFields = append(updateFields, fmt.Sprintf("priority = $%d", argIndex))
			args = append(args, *req.Priority)
			argIndex++
		}
		if req.StartAt != nil {
			updateFields = append(updateFields, fmt.Sprintf("start_at = $%d", argIndex))
			args = append(args, *req.StartAt)
			argIndex++
		}
		if req.EndAt != nil {
			updateFields = append(updateFields, fmt.Sprintf("end_at = $%d", argIndex))
			args = append(args, *req.EndAt)
			argIndex++
		}
		if req.IsFlash != nil {
			updateFields = append(updateFields, fmt.Sprintf("is_flash = $%d", argIndex))
			args = append(args, *req.IsFlash)
			argIndex++
		}
		if req.IsWeekly != nil {
			updateFields = append(updateFields, fmt.Sprintf("is_weekly = $%d", argIndex))
			args = append(args, *req.IsWeekly)
			argIndex++
		}

		if len(updateFields) == 0 {
			http.Error(w, "No fields to update", http.StatusBadRequest)
			return
		}

		// Add task ID to args
		args = append(args, taskID)
		query := fmt.Sprintf(`
			UPDATE tasks
			SET %s
			WHERE id = $%d
			RETURNING id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
		`, fmt.Sprintf("%s", updateFields[0]), argIndex)
		if len(updateFields) > 1 {
			query = fmt.Sprintf(`
				UPDATE tasks
				SET %s
				WHERE id = $%d
				RETURNING id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
			`, fmt.Sprintf("%s", updateFields[0]), argIndex)
		}

		// Use a simpler approach - build query properly
		setClause := ""
		for i, field := range updateFields {
			if i > 0 {
				setClause += ", "
			}
			setClause += field
		}

		query = fmt.Sprintf(`
			UPDATE tasks
			SET %s
			WHERE id = $%d
			RETURNING id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
		`, setClause, argIndex)

		var updatedTask store.Task
		var startAt, endAt sql.NullTime
		err = postgres.DB.QueryRowContext(ctx, query, args...).Scan(
			&updatedTask.ID, &updatedTask.Title, &updatedTask.Description, &updatedTask.XP, &updatedTask.Type,
			&updatedTask.ProofType, &updatedTask.Priority, &startAt, &endAt, &updatedTask.IsFlash,
			&updatedTask.IsWeekly, &updatedTask.CreatedBy, &updatedTask.CreatedAt,
		)
		if err != nil {
			log.Printf("Error updating task: %v", err)
			http.Error(w, fmt.Sprintf("Failed to update task: %v", err), http.StatusInternalServerError)
			return
		}

		if startAt.Valid {
			updatedTask.StartAt = &startAt.Time
		}
		if endAt.Valid {
			updatedTask.EndAt = &endAt.Time
		}

		// Get users assigned to this task (simplified - get all users who can see this task)
		// In a real system, you'd have a task_assignments table
		// For now, we'll get users who have submissions or can access the task
		submissionStore := store.NewSubmissionStore(postgres)
		submissions, err := submissionStore.GetAllSubmissions(ctx, "")
		if err == nil {
			userIDs := make(map[string]bool)
			for _, sub := range submissions {
				if sub.TaskID == taskID {
					userIDs[sub.UserID] = true
				}
			}

			// Send notifications to all users assigned to this task
			wsHub := ws.GetHub()
			if wsHub != nil && len(userIDs) > 0 {
				userIDList := make([]string, 0, len(userIDs))
				for uid := range userIDs {
					userIDList = append(userIDList, uid)
				}
				err = ws.SendTaskUpdateNotification(wsHub, userIDList, taskID, updatedTask.Title)
				if err != nil {
					log.Printf("Error sending task update notifications: %v", err)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedTask); err != nil {
			log.Printf("Error encoding update task response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetSubmissions handles getting all submissions (admin)
// @Summary      Get all submissions
// @Description  Get all task submissions with optional status filter. Admin only.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        status  query     string  false  "Filter by status (pending, approved, rejected)"
// @Success      200     {array}   store.Submission  "List of submissions"
// @Failure      401     {string}  string  "Unauthorized"
// @Failure      500     {string}  string  "Internal server error"
// @Router       /admin/submissions [get]
func handleGetSubmissions(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get status filter from query parameter
		statusFilter := r.URL.Query().Get("status")

		// Create submission store
		submissionStore := store.NewSubmissionStore(postgres)

		// Get all submissions
		submissions, err := submissionStore.GetAllSubmissions(ctx, statusFilter)
		if err != nil {
			log.Printf("Error getting submissions: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get submissions: %v", err), http.StatusInternalServerError)
			return
		}

		// Return submissions
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(submissions); err != nil {
			log.Printf("Error encoding submissions response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// ApproveSubmissionRequest represents the request body for approving a submission
type ApproveSubmissionRequest struct {
	Comment string `json:"comment,omitempty"` // Optional admin comment
}

// handleApproveSubmission handles approving a submission (admin)
// @Summary      Approve submission
// @Description  Approve a task submission. Admin only.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                  true   "Submission ID"
// @Param        request  body      ApproveSubmissionRequest  false  "Optional approval comment"
// @Success      200      {object}  store.Submission  "Submission approved successfully"
// @Failure      400      {string}  string  "Bad request"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      404      {string}  string  "Submission not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /admin/submissions/{id}/approve [post]
func handleApproveSubmission(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get submission ID from URL path
		submissionID := chi.URLParam(r, "id")
		if submissionID == "" {
			http.Error(w, "Submission ID is required", http.StatusBadRequest)
			return
		}

		// Get admin user ID from context (set by JWT middleware)
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Admin user ID not found in context. Please ensure you are authenticated.", http.StatusUnauthorized)
			return
		}

		// Verify admin exists
		adminStore := store.NewAdminStore(postgres)
		_, err := adminStore.GetAdminByID(ctx, adminUserID)
		if err != nil {
			log.Printf("Error verifying admin: %v", err)
			http.Error(w, "Admin not found. Please use a valid admin account.", http.StatusUnauthorized)
			return
		}

		// Parse request body (optional comment)
		var req ApproveSubmissionRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("Error decoding approve submission request: %v", err)
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
		}

		// Create submission store
		submissionStore := store.NewSubmissionStore(postgres)

		// Get submission to retrieve task ID and user ID
		existingSubmission, err := submissionStore.GetSubmissionByID(ctx, submissionID)
		if err != nil {
			log.Printf("Error getting submission: %v", err)
			if err.Error() == "submission not found" {
				http.Error(w, "Submission not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to get submission: %v", err), http.StatusInternalServerError)
			return
		}

		// Check if submission is already approved (to avoid duplicate XP awards)
		if existingSubmission.Status == "approved" {
			http.Error(w, "Submission already approved", http.StatusBadRequest)
			return
		}

		// Get task to retrieve XP amount
		taskStore := store.NewTaskStore(postgres)
		task, err := taskStore.GetTaskByID(ctx, existingSubmission.TaskID)
		if err != nil {
			log.Printf("Error getting task: %v", err)
			http.Error(w, "Failed to get task", http.StatusInternalServerError)
			return
		}

		// Approve submission
		submission, err := submissionStore.ApproveSubmission(ctx, submissionID, adminUserID, req.Comment)
		if err != nil {
			log.Printf("Error approving submission: %v", err)
			if err.Error() == "submission not found" {
				http.Error(w, "Submission not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to approve submission: %v", err), http.StatusInternalServerError)
			return
		}

		// Award XP to user for task approval
		xpAwarded := 0
		if task.XP > 0 {
			xpStore := store.NewXPStore(postgres)
			xpLog, err := xpStore.AwardXP(ctx, store.AwardXPRequest{
				UserID:   submission.UserID,
				XP:       task.XP,
				Source:   store.XPSourceTaskApproval,
				SourceID: submission.TaskID, // Link XP to the task
			})
			if err != nil {
				log.Printf("Error awarding XP: %v", err)
				// Log error but don't fail the approval - XP can be awarded manually later if needed
				// In production, you might want to use a queue/retry mechanism for XP awards
			} else {
				xpAwarded = task.XP
				log.Printf("Awarded %d XP to user %s for task approval (task_id: %s, xp_log_id: %s)",
					task.XP, submission.UserID, submission.TaskID, xpLog.ID)

				// Broadcast leaderboard updates via Redis
				// Get user info to determine which leaderboards to update
				userStore := store.NewUserStore(postgres)
				user, err := userStore.GetUserByID(ctx, submission.UserID)
				if err == nil {
					// Broadcast pan-india update
					ws.BroadcastLeaderboardUpdate(redisClient, "pan-india", "")

					// Broadcast state update if user has state
					if user.StateID != "" {
						ws.BroadcastLeaderboardUpdate(redisClient, "state", user.StateID)
					}

					// Broadcast college update if user has college
					if user.CollegeID != "" {
						ws.BroadcastLeaderboardUpdate(redisClient, "college", user.CollegeID)
					}
				}
			}
		}

		// Send WebSocket notification to user about task approval (always send, even if XP is 0)
		wsHub := ws.GetHub()
		if wsHub != nil {
			err = ws.SendTaskApprovalNotification(wsHub, submission.UserID, task.ID, task.Title, xpAwarded)
			if err != nil {
				log.Printf("Error sending task approval notification: %v", err)
				// Don't fail the request if notification fails
			} else {
				log.Printf("Sent task approval notification to user %s for task %s", submission.UserID, task.ID)
			}
		}

		// Create feed entry for approved submission
		feedStore := store.NewFeedStore(postgres)
		err = feedStore.CreateFeedEntry(ctx, submission.ID, submission.UserID, submission.TaskID)
		if err != nil {
			log.Printf("Error creating feed entry: %v", err)
			// Log error but don't fail the approval - feed entry can be created manually later if needed
		} else {
			log.Printf("Created feed entry for approved submission (submission_id: %s, user_id: %s, task_id: %s)",
				submission.ID, submission.UserID, submission.TaskID)
		}
		//       "timestamp": time.Now(),
		//   }
		//   ws.SendNotificationToUser(redisClient, submission.UserID, notification)
		// ============================================================================

		// Return approved submission
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(submission); err != nil {
			log.Printf("Error encoding approve submission response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// RejectSubmissionRequest represents the request body for rejecting a submission
type RejectSubmissionRequest struct {
	Comment string `json:"comment"` // Required: reason for rejection
}

// handleRejectSubmission handles rejecting a submission (admin)
// @Summary      Reject submission
// @Description  Reject a task submission with a comment. Admin only. User can resubmit if task deadline hasn't passed.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                true   "Submission ID"
// @Param        request  body      RejectSubmissionRequest  true  "Rejection comment (required)"
// @Success      200      {object}  store.Submission  "Submission rejected successfully"
// @Failure      400      {string}  string  "Bad request - missing comment"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      404      {string}  string  "Submission not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /admin/submissions/{id}/reject [post]
func handleRejectSubmission(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get submission ID from URL path
		submissionID := chi.URLParam(r, "id")
		if submissionID == "" {
			http.Error(w, "Submission ID is required", http.StatusBadRequest)
			return
		}

		// Get admin user ID from context (set by JWT middleware)
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Admin user ID not found in context. Please ensure you are authenticated.", http.StatusUnauthorized)
			return
		}

		// Verify admin exists
		adminStore := store.NewAdminStore(postgres)
		_, err := adminStore.GetAdminByID(ctx, adminUserID)
		if err != nil {
			log.Printf("Error verifying admin: %v", err)
			http.Error(w, "Admin not found. Please use a valid admin account.", http.StatusUnauthorized)
			return
		}

		// Parse request body (required comment)
		var req RejectSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding reject submission request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate comment is provided
		if req.Comment == "" {
			http.Error(w, "Rejection comment is required", http.StatusBadRequest)
			return
		}

		// Create submission store
		submissionStore := store.NewSubmissionStore(postgres)

		// Get submission to retrieve task ID and user ID
		existingSubmission, err := submissionStore.GetSubmissionByID(ctx, submissionID)
		if err != nil {
			log.Printf("Error getting submission: %v", err)
			if err.Error() == "submission not found" {
				http.Error(w, "Submission not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to get submission: %v", err), http.StatusInternalServerError)
			return
		}

		// Reject submission
		rejectedSubmission, err := submissionStore.RejectSubmission(ctx, submissionID, adminUserID, req.Comment)
		if err != nil {
			log.Printf("Error rejecting submission: %v", err)
			if err.Error() == "submission not found" {
				http.Error(w, "Submission not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to reject submission: %v", err), http.StatusInternalServerError)
			return
		}

		// Get task details for notification
		taskStore := store.NewTaskStore(postgres)
		task, err := taskStore.GetTaskByID(ctx, existingSubmission.TaskID)
		taskTitle := "Task"
		if err != nil {
			log.Printf("Error getting task for notification: %v", err)
			// Use task ID as fallback title if task lookup fails
			taskTitle = existingSubmission.TaskID
		} else {
			taskTitle = task.Title
		}

		// Send WebSocket notification to user about task rejection (always send, even if task lookup failed)
		wsHub := ws.GetHub()
		if wsHub != nil {
			err = ws.SendTaskRejectionNotification(wsHub, existingSubmission.UserID, existingSubmission.TaskID, taskTitle, req.Comment)
			if err != nil {
				log.Printf("Error sending task rejection notification: %v", err)
				// Don't fail the request if notification fails
			} else {
				log.Printf("Sent task rejection notification to user %s for task %s", existingSubmission.UserID, existingSubmission.TaskID)
			}
		}
		//
		// Note: To check if user can resubmit, get the task and check if deadline has passed:
		//   taskStore := store.NewTaskStore(postgres)
		//   task, _ := taskStore.GetTaskByID(ctx, rejectedSubmission.TaskID)
		//   canResubmit := task != nil && (task.EndAt == nil || task.EndAt.After(time.Now()))
		//
		// Example implementation:
		//   notification := map[string]interface{}{
		//       "type": "submission_rejected",
		//       "submission_id": rejectedSubmission.ID,
		//       "task_id": rejectedSubmission.TaskID,
		//       "user_id": rejectedSubmission.UserID,
		//       "admin_comment": req.Comment,
		//       "can_resubmit": canResubmit,
		//       "timestamp": time.Now(),
		//   }
		//   ws.SendNotificationToUser(redisClient, rejectedSubmission.UserID, notification)
		// ============================================================================

		// Return rejected submission
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(rejectedSubmission); err != nil {
			log.Printf("Error encoding reject submission response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// CreateBadgeRequest represents the request body for creating a badge
type CreateBadgeRequest struct {
	Name          string `json:"name"`
	Icon          string `json:"icon,omitempty"`
	Rule          string `json:"rule,omitempty"`
	XP            int    `json:"xp"`
	RequiredLevel int    `json:"required_level"`
	IsStreakBadge bool   `json:"is_streak_badge"`
}

// handleCreateBadge handles creating a new badge (admin)
// @Summary      Create badge
// @Description  Create a new badge with image upload. Admin only.
// @Tags         admin
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        name            formData  string  true   "Badge name"
// @Param        xp              formData  int     true   "XP reward for badge"
// @Param        required_level   formData  int     true   "Required level to earn badge"
// @Param        icon            formData  string  false  "Badge icon"
// @Param        rule            formData  string  false  "Badge rule description"
// @Param        is_streak_badge formData  bool    false  "Is this a streak badge"
// @Param        image           formData  file    false  "Badge image"
// @Success      201   {object}  store.Badge  "Badge created successfully"
// @Failure      400   {string}  string  "Bad request"
// @Failure      401   {string}  string  "Unauthorized"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /admin/badges [post]
func handleCreateBadge(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get admin user ID from context
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Admin user ID not found in context", http.StatusUnauthorized)
			return
		}

		// Verify admin exists
		adminStore := store.NewAdminStore(postgres)
		_, err := adminStore.GetAdminByID(ctx, adminUserID)
		if err != nil {
			log.Printf("Error verifying admin: %v", err)
			http.Error(w, "Admin not found", http.StatusUnauthorized)
			return
		}

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get form values
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "Badge name is required", http.StatusBadRequest)
			return
		}

		xpStr := r.FormValue("xp")
		if xpStr == "" {
			http.Error(w, "XP is required", http.StatusBadRequest)
			return
		}
		var xp int
		if _, err := fmt.Sscanf(xpStr, "%d", &xp); err != nil {
			http.Error(w, "Invalid XP value", http.StatusBadRequest)
			return
		}

		requiredLevelStr := r.FormValue("required_level")
		if requiredLevelStr == "" {
			http.Error(w, "Required level is required", http.StatusBadRequest)
			return
		}
		var requiredLevel int
		if _, err := fmt.Sscanf(requiredLevelStr, "%d", &requiredLevel); err != nil {
			http.Error(w, "Invalid required level value", http.StatusBadRequest)
			return
		}

		icon := r.FormValue("icon")
		rule := r.FormValue("rule")
		isStreakBadge := r.FormValue("is_streak_badge") == "true"

		// Initialize S3 storage
		badgeBucket := cfg.AWSBadgeBucket
		if badgeBucket == "" {
			badgeBucket = cfg.AWSProfileBucket // Fallback to profile bucket
		}

		s3Storage, err := storage.NewS3Storage(storage.S3Config{
			Region:             cfg.AWSRegion,
			ProfileBucket:      cfg.AWSProfileBucket,
			ResumeBucket:        cfg.AWSResumeBucket,
			TaskProofBucket:     cfg.AWSTaskProofBucket,
			BadgeBucket:         badgeBucket,
			AccessKeyID:         cfg.AWSAccessKeyID,
			SecretAccessKey:     cfg.AWSSecretAccessKey,
			ProfilePublicURL:    cfg.AWSProfilePublicURL,
			ResumePublicURL:      cfg.AWSResumePublicURL,
			TaskProofPublicURL:   cfg.AWSTaskProofPublicURL,
			BadgePublicURL:       cfg.AWSBadgePublicURL,
		})
		if err != nil {
			log.Printf("Error initializing S3 storage: %v", err)
			http.Error(w, "Failed to initialize file storage", http.StatusInternalServerError)
			return
		}

		var imageURL string
		// Handle image upload if provided
		imageFile, imageHeader, err := r.FormFile("image")
		if err == nil {
			defer imageFile.Close()

			// Validate file type
			filename := imageHeader.Filename
			ext := strings.ToLower(filepath.Ext(filename))
			allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

			isValid := false
			for _, allowedExt := range allowedExts {
				if ext == allowedExt {
					isValid = true
					break
				}
			}

			if !isValid {
				http.Error(w, "Invalid file type. Only images (JPG, PNG, GIF, WEBP) are allowed", http.StatusBadRequest)
				return
			}

			// Create badge first to get ID
			badgeStore := store.NewBadgeStore(postgres)
			tempBadge, err := badgeStore.CreateBadge(ctx, store.CreateBadgeRequest{
				Name:          name,
				Icon:          icon,
				Rule:          rule,
				XP:            xp,
				RequiredLevel: requiredLevel,
				ImageURL:       "", // Will update after upload
				IsStreakBadge: isStreakBadge,
			})
			if err != nil {
				log.Printf("Error creating badge: %v", err)
				http.Error(w, fmt.Sprintf("Failed to create badge: %v", err), http.StatusInternalServerError)
				return
			}

			// Upload image
			imageURL, err = s3Storage.UploadBadgeImage(ctx, imageFile, tempBadge.ID, filename)
			if err != nil {
				log.Printf("Error uploading badge image: %v", err)
				// Delete badge if image upload fails
				// Note: In production, you might want to keep the badge and allow image upload later
				http.Error(w, "Failed to upload badge image", http.StatusInternalServerError)
				return
			}

			// Update badge with image URL
			updateQuery := `UPDATE badges SET image_url = $1 WHERE id = $2`
			_, err = postgres.DB.ExecContext(ctx, updateQuery, imageURL, tempBadge.ID)
			if err != nil {
				log.Printf("Error updating badge image URL: %v", err)
				// Badge created but image URL not updated - not critical
			}

			tempBadge.ImageURL = imageURL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(tempBadge); err != nil {
				log.Printf("Error encoding create badge response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
			return
		}

		// Create badge without image
		badgeStore := store.NewBadgeStore(postgres)
		badge, err := badgeStore.CreateBadge(ctx, store.CreateBadgeRequest{
			Name:          name,
			Icon:          icon,
			Rule:          rule,
			XP:            xp,
			RequiredLevel: requiredLevel,
			ImageURL:      imageURL,
			IsStreakBadge: isStreakBadge,
		})
		if err != nil {
			log.Printf("Error creating badge: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create badge: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(badge); err != nil {
			log.Printf("Error encoding create badge response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// adminAuthMiddleware handles admin authentication
func adminAuthMiddleware(cfg *env.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement admin authentication
			log.Printf("Admin middleware: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
