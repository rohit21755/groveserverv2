package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/router/ws"
	"github.com/rohit21755/groveserverv2/internal/storage"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleGetMe handles getting the current user
// @Summary      Get current user
// @Description  Get the authenticated user's profile with state and college names
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  store.User  "Current user profile"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/me [get]
func handleGetMe(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Create user store
		userStore := store.NewUserStore(postgres)

		// Get user details with state and college names
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Return user
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(user); err != nil {
			log.Printf("Error encoding user response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// UserProfile represents a complete user profile
type UserProfile struct {
	User           *store.User      `json:"user"`
	CompletedTasks []store.FeedItem `json:"completed_tasks"`
	FollowingCount int              `json:"following_count"`
	FollowersCount int              `json:"followers_count"`
	StateName      string           `json:"state_name,omitempty"`
	CollegeName    string           `json:"college_name,omitempty"`
}

// handleGetUser handles getting a user profile by ID with completed tasks, following/followers
// @Summary      Get user profile
// @Description  Get a user's complete profile including completed tasks, resume, profile picture, following/followers count, college, and state.
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "User ID"
// @Success      200  {object}  UserProfile  "User profile"
// @Failure      404  {string}  string  "User not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/{id} [get]
func handleGetUser(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from URL path
		userID := chi.URLParam(r, "id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Create stores
		userStore := store.NewUserStore(postgres)
		feedStore := store.NewFeedStore(postgres)

		// Get user details
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get following and followers count
		followingCount, err := userStore.GetFollowingCount(ctx, userID)
		if err != nil {
			log.Printf("Error getting following count: %v", err)
			followingCount = 0
		}

		followersCount, err := userStore.GetFollowersCount(ctx, userID)
		if err != nil {
			log.Printf("Error getting followers count: %v", err)
			followersCount = 0
		}

		// Get completed tasks (feed items) for this user
		completedTasks, _, err := feedStore.GetUserFeed(ctx, userID, 1, 50) // Get first 50 completed tasks
		if err != nil {
			log.Printf("Error getting user feed: %v", err)
			completedTasks = []store.FeedItem{}
		}

		// Get state and college names
		stateName := ""
		collegeName := ""
		if user.StateID != "" {
			stateQuery := `SELECT name FROM states WHERE id = $1`
			err := postgres.DB.QueryRowContext(ctx, stateQuery, user.StateID).Scan(&stateName)
			if err != nil {
				log.Printf("Error getting state name: %v", err)
			}
		}
		if user.CollegeID != "" {
			collegeQuery := `SELECT name FROM colleges WHERE id = $1`
			err := postgres.DB.QueryRowContext(ctx, collegeQuery, user.CollegeID).Scan(&collegeName)
			if err != nil {
				log.Printf("Error getting college name: %v", err)
			}
		}

		// Build profile response
		profile := UserProfile{
			User:           user,
			CompletedTasks: completedTasks,
			FollowingCount: followingCount,
			FollowersCount: followersCount,
			StateName:      stateName,
			CollegeName:    collegeName,
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			log.Printf("Error encoding user profile response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleFollow handles following a user
// @Summary      Follow user
// @Description  Follow another user. The authenticated user will follow the user specified in the URL path.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID to follow"
// @Success      200  {object}  map[string]interface{}  "Successfully followed user"
// @Failure      400  {string}  string  "Bad request - invalid user ID or already following"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "User not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/{id}/follow [post]
func handleFollow(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get follower ID from context (set by JWT middleware)
		followerID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get following ID from URL path
		followingID := chi.URLParam(r, "id")
		if followingID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Create user store
		userStore := store.NewUserStore(postgres)

		// Follow user
		err := userStore.FollowUser(ctx, followerID, followingID)
		if err != nil {
			log.Printf("Error following user: %v", err)

			// Check for specific errors
			if err.Error() == "cannot follow yourself" {
				http.Error(w, "Cannot follow yourself", http.StatusBadRequest)
				return
			}
			if err.Error() == "already following this user" {
				http.Error(w, "Already following this user", http.StatusBadRequest)
				return
			}
			if err.Error() == "user to follow not found" {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			http.Error(w, fmt.Sprintf("Failed to follow user: %v", err), http.StatusInternalServerError)
			return
		}

		// ============================================================================
		// TODO: Send WebSocket notification to the user being followed
		// ============================================================================
		// Call WebSocket notification function here to notify the user that they have a new follower.
		// This should be implemented in internal/router/ws/notifications.go
		//
		// Example implementation:
		//   notification := map[string]interface{}{
		//       "type": "new_follower",
		//       "follower_id": followerID,
		//       "follower_name": followerName, // Get from user store if needed
		//       "timestamp": time.Now(),
		//   }
		//   ws.SendNotificationToUser(redisClient, followingID, notification)
		// ============================================================================

		// Return success response
		response := map[string]interface{}{
			"message":      "Successfully followed user",
			"following_id": followingID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding follow response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleUnfollow handles unfollowing a user
// @Summary      Unfollow user
// @Description  Unfollow a user. The authenticated user will unfollow the user specified in the URL path.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID to unfollow"
// @Success      200  {object}  map[string]interface{}  "Successfully unfollowed user"
// @Failure      400  {string}  string  "Bad request - invalid user ID or not following"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "User not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/{id}/unfollow [post]
func handleUnfollow(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get follower ID from context (set by JWT middleware)
		followerID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get following ID from URL path
		followingID := chi.URLParam(r, "id")
		if followingID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Create user store
		userStore := store.NewUserStore(postgres)

		// Unfollow user
		err := userStore.UnfollowUser(ctx, followerID, followingID)
		if err != nil {
			log.Printf("Error unfollowing user: %v", err)

			// Check for specific errors
			if err.Error() == "cannot unfollow yourself" {
				http.Error(w, "Cannot unfollow yourself", http.StatusBadRequest)
				return
			}
			if err.Error() == "not following this user" {
				http.Error(w, "Not following this user", http.StatusBadRequest)
				return
			}

			http.Error(w, fmt.Sprintf("Failed to unfollow user: %v", err), http.StatusInternalServerError)
			return
		}

		// Return success response
		response := map[string]interface{}{
			"message":      "Successfully unfollowed user",
			"following_id": followingID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding unfollow response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleUploadResume handles uploading a user's resume (for users who didn't upload during registration)
// @Summary      Upload resume
// @Description  Upload a resume file for the authenticated user. Only works if user hasn't uploaded a resume during registration.
// @Tags         user
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        resume  formData  file  true  "Resume file (PDF recommended)"
// @Success      200     {object}  store.User  "Resume uploaded successfully"
// @Failure      400     {string}  string  "Bad request - user already has a resume or invalid file"
// @Failure      401     {string}  string  "Unauthorized"
// @Failure      500     {string}  string  "Internal server error"
// @Router       /api/user/resume [post]
func handleUploadResume(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user to check if resume already exists
		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check if user already has a resume
		if user.ResumeURL != "" {
			http.Error(w, "Resume already exists. Use PUT /api/user/resume to update it.", http.StatusBadRequest)
			return
		}

		// Initialize S3 storage
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

		// Parse multipart form (max 10MB)
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get resume file
		resumeFile, resumeHeader, err := r.FormFile("resume")
		if err != nil {
			http.Error(w, "Resume file is required", http.StatusBadRequest)
			return
		}
		defer resumeFile.Close()

		// Upload resume to S3
		resumeURL, err := s3Storage.UploadResume(ctx, resumeFile, userID, resumeHeader.Filename)
		if err != nil {
			log.Printf("Error uploading resume: %v", err)
			http.Error(w, "Failed to upload resume", http.StatusInternalServerError)
			return
		}

		// Update user's resume URL in database
		err = userStore.UpdateResumeURL(ctx, userID, resumeURL)
		if err != nil {
			log.Printf("Error updating resume URL: %v", err)
			// Try to delete uploaded file
			key := extractS3KeyFromURL(resumeURL)
			_ = s3Storage.DeleteResume(ctx, key)
			http.Error(w, "Failed to update resume URL", http.StatusInternalServerError)
			return
		}

		// Get updated user
		updatedUser, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting updated user: %v", err)
			http.Error(w, "Failed to retrieve updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleUpdateResume handles updating a user's existing resume
// @Summary      Update resume
// @Description  Update the resume file for the authenticated user. Replaces existing resume.
// @Tags         user
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        resume  formData  file  true  "Resume file (PDF recommended)"
// @Success      200     {object}  store.User  "Resume updated successfully"
// @Failure      400     {string}  string  "Bad request - invalid file"
// @Failure      401     {string}  string  "Unauthorized"
// @Failure      500     {string}  string  "Internal server error"
// @Router       /api/user/resume [put]
func handleUpdateResume(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user to get existing resume URL
		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Initialize S3 storage
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

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get resume file
		resumeFile, resumeHeader, err := r.FormFile("resume")
		if err != nil {
			http.Error(w, "Resume file is required", http.StatusBadRequest)
			return
		}
		defer resumeFile.Close()

		// Upload new resume to S3
		newResumeURL, err := s3Storage.UploadResume(ctx, resumeFile, userID, resumeHeader.Filename)
		if err != nil {
			log.Printf("Error uploading resume: %v", err)
			http.Error(w, "Failed to upload resume", http.StatusInternalServerError)
			return
		}

		// Update user's resume URL in database
		err = userStore.UpdateResumeURL(ctx, userID, newResumeURL)
		if err != nil {
			log.Printf("Error updating resume URL: %v", err)
			// Try to delete uploaded file
			key := extractS3KeyFromURL(newResumeURL)
			_ = s3Storage.DeleteResume(ctx, key)
			http.Error(w, "Failed to update resume URL", http.StatusInternalServerError)
			return
		}

		// Delete old resume from S3 if it exists
		if user.ResumeURL != "" {
			oldKey := extractS3KeyFromURL(user.ResumeURL)
			_ = s3Storage.DeleteResume(ctx, oldKey)
		}

		// Get updated user
		updatedUser, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting updated user: %v", err)
			http.Error(w, "Failed to retrieve updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleUploadProfilePic handles uploading a user's profile picture (for users who didn't upload during registration)
// @Summary      Upload profile picture
// @Description  Upload a profile picture for the authenticated user. Only works if user hasn't uploaded a profile picture during registration.
// @Tags         user
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        profile_pic  formData  file  true  "Profile picture (JPG/PNG)"
// @Success      200          {object}  store.User  "Profile picture uploaded successfully"
// @Failure      400          {string}  string  "Bad request - user already has a profile picture or invalid file"
// @Failure      401          {string}  string  "Unauthorized"
// @Failure      500          {string}  string  "Internal server error"
// @Router       /api/user/profile-pic [post]
func handleUploadProfilePic(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user to check if profile pic already exists
		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check if user already has a profile picture
		if user.AvatarURL != "" {
			http.Error(w, "Profile picture already exists. Use PUT /api/user/profile-pic to update it.", http.StatusBadRequest)
			return
		}

		// Initialize S3 storage
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

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get profile picture file
		profilePicFile, profilePicHeader, err := r.FormFile("profile_pic")
		if err != nil {
			http.Error(w, "Profile picture file is required", http.StatusBadRequest)
			return
		}
		defer profilePicFile.Close()

		// Upload profile picture to S3
		profilePicURL, err := s3Storage.UploadProfilePic(ctx, profilePicFile, userID, profilePicHeader.Filename)
		if err != nil {
			log.Printf("Error uploading profile picture: %v", err)
			http.Error(w, "Failed to upload profile picture", http.StatusInternalServerError)
			return
		}

		// Update user's profile picture URL in database
		err = userStore.UpdateProfilePicURL(ctx, userID, profilePicURL)
		if err != nil {
			log.Printf("Error updating profile picture URL: %v", err)
			// Try to delete uploaded file
			key := extractS3KeyFromURL(profilePicURL)
			_ = s3Storage.DeleteProfilePic(ctx, key)
			http.Error(w, "Failed to update profile picture URL", http.StatusInternalServerError)
			return
		}

		// Get updated user
		updatedUser, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting updated user: %v", err)
			http.Error(w, "Failed to retrieve updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleUpdateProfilePic handles updating a user's existing profile picture
// @Summary      Update profile picture
// @Description  Update the profile picture for the authenticated user. Replaces existing profile picture.
// @Tags         user
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        profile_pic  formData  file  true  "Profile picture (JPG/PNG)"
// @Success      200          {object}  store.User  "Profile picture updated successfully"
// @Failure      400          {string}  string  "Bad request - invalid file"
// @Failure      401          {string}  string  "Unauthorized"
// @Failure      500          {string}  string  "Internal server error"
// @Router       /api/user/profile-pic [put]
func handleUpdateProfilePic(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user to get existing profile pic URL
		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Initialize S3 storage
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

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get profile picture file
		profilePicFile, profilePicHeader, err := r.FormFile("profile_pic")
		if err != nil {
			http.Error(w, "Profile picture file is required", http.StatusBadRequest)
			return
		}
		defer profilePicFile.Close()

		// Upload new profile picture to S3
		newProfilePicURL, err := s3Storage.UploadProfilePic(ctx, profilePicFile, userID, profilePicHeader.Filename)
		if err != nil {
			log.Printf("Error uploading profile picture: %v", err)
			http.Error(w, "Failed to upload profile picture", http.StatusInternalServerError)
			return
		}

		// Update user's profile picture URL in database
		err = userStore.UpdateProfilePicURL(ctx, userID, newProfilePicURL)
		if err != nil {
			log.Printf("Error updating profile picture URL: %v", err)
			// Try to delete uploaded file
			key := extractS3KeyFromURL(newProfilePicURL)
			_ = s3Storage.DeleteProfilePic(ctx, key)
			http.Error(w, "Failed to update profile picture URL", http.StatusInternalServerError)
			return
		}

		// Delete old profile picture from S3 if it exists
		if user.AvatarURL != "" {
			oldKey := extractS3KeyFromURL(user.AvatarURL)
			_ = s3Storage.DeleteProfilePic(ctx, oldKey)
		}

		// Get updated user
		updatedUser, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting updated user: %v", err)
			http.Error(w, "Failed to retrieve updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetMyBadges handles getting badges for the authenticated user
// @Summary      Get my badges
// @Description  Get all badges earned by the authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   store.UserBadge  "List of user badges"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/badges [get]
func handleGetMyBadges(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user badges
		badgeStore := store.NewBadgeStore(postgres)
		badges, err := badgeStore.GetUserBadges(ctx, userID)
		if err != nil {
			log.Printf("Error getting user badges: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get badges: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(badges); err != nil {
			log.Printf("Error encoding badges response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetMyTaskHistory handles getting task submission history for the authenticated user
// @Summary      Get my task history
// @Description  Get all task submissions (approved and rejected) for the authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   store.Submission  "List of submissions"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/tasks/history [get]
func handleGetMyTaskHistory(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user submissions
		query := `
			SELECT id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
			FROM submissions
			WHERE user_id = $1
			ORDER BY created_at DESC
		`

		rows, err := postgres.DB.QueryContext(ctx, query, userID)
		if err != nil {
			log.Printf("Error getting user submissions: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get submissions: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var submissions []store.Submission
		for rows.Next() {
			var submission store.Submission
			var adminComment, reviewedBy sql.NullString

			err := rows.Scan(
				&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
				&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
			)
			if err != nil {
				log.Printf("Error scanning submission: %v", err)
				continue
			}

			if adminComment.Valid {
				submission.AdminComment = adminComment.String
			}
			if reviewedBy.Valid {
				submission.ReviewedBy = reviewedBy.String
			}

			submissions = append(submissions, submission)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(submissions); err != nil {
			log.Printf("Error encoding submissions response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// UserAddXPRequest is the body for adding XP to own account (user only, not admin).
type UserAddXPRequest struct {
	XP     int    `json:"xp"`
	Reason string `json:"reason,omitempty"`
}

// handleAddXPForUser adds XP to the authenticated user's own account. User-only route (no admin).
// @Summary      Add XP to my account
// @Description  Add XP to your own account. JWT required. Use for redeeming codes, claiming rewards, etc. Logs in xp_logs (source user_add) and broadcasts leaderboard update.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  UserAddXPRequest  true  "xp (required), optional reason"
// @Success      200   {object}  map[string]interface{}  "xp_awarded, new_total_xp, xp_log_id"
// @Failure      400   {string}  string  "Bad request"
// @Failure      401   {string}  string  "Unauthorized"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /api/user/xp [post]
func handleAddXPForUser(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req UserAddXPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.XP <= 0 {
			http.Error(w, "xp must be greater than 0", http.StatusBadRequest)
			return
		}

		xpStore := store.NewXPStore(postgres)
		xpLog, err := xpStore.AwardXP(ctx, store.AwardXPRequest{
			UserID:   userID,
			XP:       req.XP,
			Source:   store.XPSourceUserAdd,
			SourceID: req.Reason,
		})
		if err != nil {
			log.Printf("Error adding XP for user %s: %v", userID, err)
			http.Error(w, fmt.Sprintf("Failed to add XP: %v", err), http.StatusInternalServerError)
			return
		}

		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("Error getting user after XP award: %v", err)
		} else if redisClient != nil {
			leaderboardStore := store.NewLeaderboardStore(postgres)
			rank, _ := leaderboardStore.GetUserRank(ctx, userID)
			newXP := user.XP
			ws.BroadcastLeaderboardUpdate(redisClient, "pan-india", "", userID, rank, newXP)
			if user.StateID != "" {
				ws.BroadcastLeaderboardUpdate(redisClient, "state", user.StateID, userID, rank, newXP)
			}
			if user.CollegeID != "" {
				ws.BroadcastLeaderboardUpdate(redisClient, "college", user.CollegeID, userID, rank, newXP)
			}
		}

		response := map[string]interface{}{
			"xp_awarded": req.XP,
			"xp_log_id":  xpLog.ID,
		}
		if user != nil {
			response["new_total_xp"] = user.XP
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// handleStreakCheckIn records a daily check-in and updates the user's streak.
// Call when the user opens the app / checks in for the day. Same day repeated calls are idempotent.
// @Summary      Daily streak check-in
// @Description  Record a daily check-in to the app. Counts toward streak (consecutive days). Same-day calls are idempotent. Returns current streak_days and streak_started_at.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "Check-in recorded, current streak"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/streak/check-in [post]
func handleStreakCheckIn(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		streakStore := store.NewStreakStore(postgres)
		err := streakStore.UpdateStreak(ctx, userID)
		if err != nil {
			log.Printf("Error updating streak on check-in: %v", err)
			http.Error(w, fmt.Sprintf("Failed to record check-in: %v", err), http.StatusInternalServerError)
			return
		}

		streakDays, startedAt, err := streakStore.GetUserStreak(ctx, userID)
		if err != nil {
			log.Printf("Error getting user streak: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get streak: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"streak_days": streakDays,
		}
		if startedAt != nil {
			response["streak_started_at"] = startedAt.Format(time.RFC3339)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding streak check-in response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleRedeemStreak handles redeeming streak rewards
// @Summary      Redeem streak reward
// @Description  Redeem XP and badges based on current streak. Updates streak if needed.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "Streak reward redeemed"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/user/streak/redeem [post]
func handleRedeemStreak(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Update streak first (in case user is active today)
		streakStore := store.NewStreakStore(postgres)
		err := streakStore.UpdateStreak(ctx, userID)
		if err != nil {
			log.Printf("Error updating streak: %v", err)
			// Continue anyway
		}

		// Get current streak
		streakDays, _, err := streakStore.GetUserStreak(ctx, userID)
		if err != nil {
			log.Printf("Error getting user streak: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get streak: %v", err), http.StatusInternalServerError)
			return
		}

		// Redeem streak reward
		xpReward, badgeIDs, err := streakStore.RedeemStreakReward(ctx, userID, streakDays)
		if err != nil {
			log.Printf("Error redeeming streak reward: %v", err)
			http.Error(w, fmt.Sprintf("Failed to redeem streak reward: %v", err), http.StatusInternalServerError)
			return
		}

		// Get user to check for badge auto-awarding
		userStore := store.NewUserStore(postgres)
		user, err := userStore.GetUserByID(ctx, userID)
		if err == nil {
			// Check and award badges based on new XP/level
			badgeStore := store.NewBadgeStore(postgres)
			_ = badgeStore.CheckAndAwardBadges(ctx, userID, user.XP, user.Level)
		}

		response := map[string]interface{}{
			"streak_days":    streakDays,
			"xp_reward":      xpReward,
			"badges_awarded": badgeIDs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding streak redeem response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// extractS3KeyFromURL extracts the S3 key from a full URL
// func extractS3KeyFromURL(url string) string {
// 	// URL format: https://bucket.s3.region.amazonaws.com/folder/filename
// 	// We need to extract: folder/filename
// 	parts := url
// 	if len(url) > 0 {
// 		// Find the last "/" and take everything after it
// 		for i := len(url) - 1; i >= 0; i-- {
// 			if url[i] == '/' {
// 				parts = url[i+1:]
// 				break
// 			}
// 		}
// 	}
// 	return parts
// }
