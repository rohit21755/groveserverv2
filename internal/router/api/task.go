package api

import (
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
	"github.com/rohit21755/groveserverv2/internal/storage"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleGetTasks handles getting all tasks assigned to the authenticated user
// @Summary      Get tasks
// @Description  Get all tasks assigned to the authenticated user. Returns active tasks that are available for submission.
// @Tags         task
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   store.Task  "List of tasks"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/tasks [get]
func handleGetTasks(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Create task store
		taskStore := store.NewTaskStore(postgres)

		// Get tasks for user
		tasks, err := taskStore.GetTasksForUser(ctx, userID)
		if err != nil {
			log.Printf("Error getting tasks: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
			return
		}

		// Return tasks
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(tasks); err != nil {
			log.Printf("Error encoding tasks response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleSubmitTask handles submitting a task with proof (image or video)
// @Summary      Submit task
// @Description  Submit a task with proof file (image or video). The proof file will be uploaded to S3.
// @Tags         task
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string  true   "Task ID"
// @Param        proof formData  file    true   "Proof file (image or video)"
// @Success      201   {object}  store.Submission  "Submission created successfully"
// @Failure      400   {string}  string  "Bad request - invalid file or task already submitted"
// @Failure      401   {string}  string  "Unauthorized"
// @Failure      404   {string}  string  "Task not found"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /api/tasks/{id}/submit [post]
func handleSubmitTask(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get task ID from URL path
		taskID := chi.URLParam(r, "id")
		if taskID == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		// Verify task exists and get task details
		taskStore := store.NewTaskStore(postgres)
		task, err := taskStore.GetTaskByID(ctx, taskID)
		if err != nil {
			log.Printf("Error getting task: %v", err)
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		// Validate task is still active (not expired)
		now := time.Now()
		if task.EndAt != nil && task.EndAt.Before(now) {
			http.Error(w, "Task has expired", http.StatusBadRequest)
			return
		}
		if task.StartAt != nil && task.StartAt.After(now) {
			http.Error(w, "Task has not started yet", http.StatusBadRequest)
			return
		}

		// Check if user has already submitted this task
		submissionStore := store.NewSubmissionStore(postgres)
		existingSubmission, err := submissionStore.GetSubmissionByTaskAndUser(ctx, taskID, userID)
		if err != nil && err.Error() != "submission not found" {
			log.Printf("Error checking submission: %v", err)
			http.Error(w, "Failed to check submission", http.StatusInternalServerError)
			return
		}

		// If submission exists, check if resubmission is allowed
		if existingSubmission != nil {
			if existingSubmission.Status == "approved" {
				http.Error(w, "Task already approved. Cannot resubmit.", http.StatusBadRequest)
				return
			}
			if existingSubmission.Status == "pending" {
				http.Error(w, "Task submission is pending review. Cannot resubmit.", http.StatusBadRequest)
				return
			}
			// If rejected, allow resubmission only if task hasn't expired
			if existingSubmission.Status == "rejected" {
				if task.EndAt != nil && task.EndAt.Before(now) {
					http.Error(w, "Task has expired. Cannot resubmit rejected submission.", http.StatusBadRequest)
					return
				}
				// Allow resubmission - will be handled by CreateSubmission
			}
		}

		// Initialize S3 storage (using profile bucket for proof files, or create a separate bucket)
		// For now, we'll use the profile bucket for proof files
		s3Storage, err := storage.NewS3Storage(storage.S3Config{
			Region:           cfg.AWSRegion,
			ProfileBucket:    cfg.AWSProfileBucket,
			ResumeBucket:     cfg.AWSResumeBucket,
			AccessKeyID:      cfg.AWSAccessKeyID,
			SecretAccessKey:  cfg.AWSSecretAccessKey,
			ProfilePublicURL: cfg.AWSProfilePublicURL,
			ResumePublicURL:  cfg.AWSResumePublicURL,
		})
		if err != nil {
			log.Printf("Error initializing S3 storage: %v", err)
			http.Error(w, "Failed to initialize file storage", http.StatusInternalServerError)
			return
		}

		// Parse multipart form (max 50MB for videos)
		err = r.ParseMultipartForm(50 << 20) // 50MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get proof file
		proofFile, proofHeader, err := r.FormFile("proof")
		if err != nil {
			http.Error(w, "Proof file is required", http.StatusBadRequest)
			return
		}
		defer proofFile.Close()

		// Validate file type (image or video)
		filename := proofHeader.Filename
		ext := strings.ToLower(filepath.Ext(filename))
		allowedImageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
		allowedVideoExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm"}
		
		isImage := false
		isVideo := false
		for _, allowedExt := range allowedImageExts {
			if ext == allowedExt {
				isImage = true
				break
			}
		}
		if !isImage {
			for _, allowedExt := range allowedVideoExts {
				if ext == allowedExt {
					isVideo = true
					break
				}
			}
		}

		if !isImage && !isVideo {
			http.Error(w, "Invalid file type. Only images (JPG, PNG, GIF, WEBP) and videos (MP4, MOV, AVI, MKV, WEBM) are allowed", http.StatusBadRequest)
			return
		}

		// Upload proof file to S3
		// Use a unique key: task-proofs/{taskID}/{userID}_{filename}
		proofKey := fmt.Sprintf("task-proofs/%s/%s_%s", taskID, userID, filename)
		
		var proofURL string
		if isImage {
			// Upload image using UploadProfilePic (uses profile bucket)
			proofURL, err = s3Storage.UploadProfilePic(ctx, proofFile, userID, filename)
		} else {
			// For videos, upload directly using UploadFile to profile bucket
			// Determine content type
			contentType := "video/mp4"
			switch ext {
			case ".mov":
				contentType = "video/quicktime"
			case ".avi":
				contentType = "video/x-msvideo"
			case ".mkv":
				contentType = "video/x-matroska"
			case ".webm":
				contentType = "video/webm"
			}
			
			// Upload video to profile bucket
			proofURL, err = s3Storage.UploadFile(ctx, proofFile, s3Storage.GetProfileBucket(), proofKey, contentType, cfg.AWSProfilePublicURL, false)
		}

		if err != nil {
			log.Printf("Error uploading proof file: %v", err)
			http.Error(w, "Failed to upload proof file", http.StatusInternalServerError)
			return
		}

		// Create or update submission (if resubmission)
		submission, err := submissionStore.CreateSubmission(ctx, store.CreateSubmissionRequest{
			TaskID:   taskID,
			UserID:   userID,
			ProofURL: proofURL,
		})
		if err != nil {
			log.Printf("Error creating submission: %v", err)
			
			// Try to delete uploaded file if submission creation fails
			key := extractS3KeyFromURL(proofURL)
			if isImage {
				_ = s3Storage.DeleteProfilePic(ctx, key)
			} else {
				_ = s3Storage.DeleteProfilePic(ctx, key) // Using same method for now
			}
			
			if strings.Contains(err.Error(), "already exists") {
				http.Error(w, "Task already submitted", http.StatusBadRequest)
				return
			}
			
			http.Error(w, fmt.Sprintf("Failed to create submission: %v", err), http.StatusInternalServerError)
			return
		}

		// ============================================================================
		// TODO: Send WebSocket notification to admin about new submission
		// ============================================================================
		// Call WebSocket notification function here to notify admins about the new submission.
		// This should be implemented in internal/router/ws/notifications.go
		//
		// Example implementation:
		//   notification := map[string]interface{}{
		//       "type": "new_submission",
		//       "submission_id": submission.ID,
		//       "task_id": taskID,
		//       "task_title": task.Title,
		//       "user_id": userID,
		//       "proof_url": proofURL,
		//       "timestamp": time.Now(),
		//   }
		//   ws.SendNotificationToAdmins(redisClient, notification)
		// ============================================================================

		// Return submission
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(submission); err != nil {
			log.Printf("Error encoding submission response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
