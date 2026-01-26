package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	XP             int        `json:"xp"`
	Type           string     `json:"type"`
	ProofType      string     `json:"proof_type"`
	Priority       string     `json:"priority"`
	StartAt        *time.Time `json:"start_at,omitempty"`
	EndAt          *time.Time `json:"end_at,omitempty"`
	IsFlash        bool       `json:"is_flash"`
	IsWeekly       bool       `json:"is_weekly"`
	// Assignment fields
	AssignmentType store.AssignmentType `json:"assignment_type"` // "all", "state", "college", "user"
	AssignmentID   string               `json:"assignment_id,omitempty"` // State ID, College ID, or User ID (empty for "all")
}

// CreateTaskResponse represents the response after creating a task
type CreateTaskResponse struct {
	Task    *store.Task `json:"task"`
	AssignedTo int      `json:"assigned_to"` // Number of users the task was assigned to
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

		// Get admin user ID from context (set by admin middleware)
		// For now, we'll use a placeholder - TODO: Get from admin middleware
		adminUserID := "admin-user-id" // TODO: Get from admin middleware context
		
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

		// ============================================================================
		// TODO: Send WebSocket notifications to all assigned users
		// ============================================================================
		// Call WebSocket notification function here to notify users about the new task.
		// This should be implemented in internal/router/ws/notifications.go
		//
		// Example implementation:
		//   ws.SendTaskAssignmentNotifications(redisClient, assignedUserIDs, task)
		//
		// Or send individual notifications:
		//   for _, userID := range assignedUserIDs {
		//       notification := map[string]interface{}{
		//           "type": "task_assigned",
		//           "task_id": task.ID,
		//           "task_title": task.Title,
		//           "task_description": task.Description,
		//           "xp": task.XP,
		//           "priority": task.Priority,
		//           "is_flash": task.IsFlash,
		//           "created_at": task.CreatedAt,
		//       }
		//       ws.SendNotificationToUser(redisClient, userID, notification)
		//   }
		//
		// The WebSocket notification should:
		//   1. Check if user is connected via /ws/notifications endpoint
		//   2. Send notification message to connected WebSocket clients
		//   3. Optionally store notification in database (notifications table) for offline users
		// ============================================================================

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

// handleUpdateTask handles updating a task (admin)
func handleUpdateTask(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
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
func handleApproveSubmission(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get submission ID from URL path
		submissionID := chi.URLParam(r, "id")
		if submissionID == "" {
			http.Error(w, "Submission ID is required", http.StatusBadRequest)
			return
		}

		// Get admin user ID from context (set by admin middleware)
		// For now, we'll use JWT middleware to get user ID
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			// Fallback: try to get from admin middleware context
			// TODO: Implement proper admin middleware that sets admin user ID
			adminUserID = "admin-user-id" // Placeholder
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
				log.Printf("Awarded %d XP to user %s for task approval (task_id: %s, xp_log_id: %s)",
					task.XP, submission.UserID, submission.TaskID, xpLog.ID)
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

		// ============================================================================
		// TODO: Send WebSocket notification to user about submission approval
		// ============================================================================
		// Call WebSocket notification function here to notify the user about the approval.
		// This should be implemented in internal/router/ws/notifications.go
		//
		// Example implementation:
		//   notification := map[string]interface{}{
		//       "type": "submission_approved",
		//       "submission_id": submission.ID,
		//       "task_id": submission.TaskID,
		//       "user_id": submission.UserID,
		//       "xp_awarded": task.XP,
		//       "admin_comment": req.Comment,
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

		// Get admin user ID from context (set by admin middleware)
		// For now, we'll use JWT middleware to get user ID
		adminUserID, ok := GetUserIDFromContext(ctx)
		if !ok {
			// Fallback: try to get from admin middleware context
			// TODO: Implement proper admin middleware that sets admin user ID
			adminUserID = "admin-user-id" // Placeholder
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

		// ============================================================================
		// TODO: Send WebSocket notification to user about submission rejection
		// ============================================================================
		// Call WebSocket notification function here to notify the user about the rejection.
		// This should be implemented in internal/router/ws/notifications.go
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
