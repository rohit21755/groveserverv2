package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/storage"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleLogin handles user login
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "Login credentials"
// @Success      200         {object}  LoginResponse
// @Failure      401         {string}  string  "Invalid credentials"
// @Router       /api/auth/login [post]
func handleLogin(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleRegister handles user registration
// @Summary      User registration
// @Description  Register a new user account. Each user gets a unique referral code automatically. Referral code (input), resume, and profile picture are optional.
// @Tags         auth
// @Accept       multipart/form-data
// @Produce      json
// @Param        name          formData  string  true   "User's full name"
// @Param        email         formData  string  true   "User's email address (must be unique)"
// @Param        password      formData  string  true   "User's password"
// @Param        state_id      formData  string  true   "State ID (UUID)"
// @Param        college_id    formData  string  true   "College ID (UUID)"
// @Param        referral_code formData  string  false  "Optional: Referral code of the user who referred them"
// @Param        resume        formData  file    false  "Optional: Resume file (PDF recommended)"
// @Param        profile_pic   formData  file    false  "Optional: Profile picture (JPG/PNG)"
// @Success      201           {object}  store.User  "User created with auto-generated referral_code"
// @Failure      400           {string}  string  "Bad request - missing required fields or invalid data"
// @Failure      500           {string}  string  "Internal server error"
// @Router       /api/auth/register [post]
func handleRegister(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Initialize S3 storage
		s3Storage, err := storage.NewS3Storage(storage.S3Config{
			Region:              cfg.AWSRegion,
			ProfileBucket:       cfg.AWSProfileBucket,
			ResumeBucket:        cfg.AWSResumeBucket,
			AccessKeyID:         cfg.AWSAccessKeyID,
			SecretAccessKey:     cfg.AWSSecretAccessKey,
			ProfilePublicURL:    cfg.AWSProfilePublicURL,
			ResumePublicURL:     cfg.AWSResumePublicURL,
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

		// Get form values
		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")
		stateID := r.FormValue("state_id")
		collegeID := r.FormValue("college_id")
		referralCode := r.FormValue("referral_code") // Optional

		// Validate required fields
		if name == "" || email == "" || password == "" || stateID == "" || collegeID == "" {
			http.Error(w, "Missing required fields: name, email, password, state_id, college_id are required", http.StatusBadRequest)
			return
		}

		// Handle resume upload (optional)
		var resumeURL string
		resumeFile, resumeHeader, err := r.FormFile("resume")
		if err == nil && resumeFile != nil {
			defer resumeFile.Close()
			
			// Use email as temporary identifier (will be updated after user creation if needed)
			tempUserID := email
			
			resumeURL, err = s3Storage.UploadResume(ctx, resumeFile, tempUserID, resumeHeader.Filename)
			if err != nil {
				log.Printf("Error uploading resume: %v", err)
				// Continue without resume if upload fails
				resumeURL = ""
			}
		}

		// Handle profile picture upload (optional)
		var profilePicURL string
		profilePicFile, profilePicHeader, err := r.FormFile("profile_pic")
		if err == nil && profilePicFile != nil {
			defer profilePicFile.Close()
			
			tempUserID := email
			
			profilePicURL, err = s3Storage.UploadProfilePic(ctx, profilePicFile, tempUserID, profilePicHeader.Filename)
			if err != nil {
				log.Printf("Error uploading profile picture: %v", err)
				// Continue without profile pic if upload fails
				profilePicURL = ""
			}
		}

		// Create user store
		userStore := store.NewUserStore(postgres)

		// Register user
		registerReq := store.RegisterRequest{
			Name:         name,
			Email:        email,
			Password:     password,
			StateID:      stateID,
			CollegeID:    collegeID,
			ReferralCode: referralCode,
		}

		user, err := userStore.Register(ctx, registerReq, resumeURL, profilePicURL)
		if err != nil {
			log.Printf("Error registering user: %v", err)
			
			// If user creation failed, try to clean up uploaded files
			if resumeURL != "" {
				// Extract key from URL and delete
				key := extractS3KeyFromURL(resumeURL)
				_ = s3Storage.DeleteResume(ctx, key)
			}
			if profilePicURL != "" {
				key := extractS3KeyFromURL(profilePicURL)
				_ = s3Storage.DeleteProfilePic(ctx, key)
			}
			
			http.Error(w, fmt.Sprintf("Failed to register user: %v", err), http.StatusInternalServerError)
			return
		}

		// If files were uploaded with temp IDs, we might want to rename them
		// For now, we'll keep the temp IDs in the filename - this is acceptable

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(user); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// extractS3KeyFromURL extracts the S3 key from a full URL
func extractS3KeyFromURL(url string) string {
	// URL format: https://bucket.s3.region.amazonaws.com/folder/filename
	// We need to extract: folder/filename
	// Simple approach: find the last part after the domain
	parts := url
	if len(url) > 0 {
		// Find the last "/" and take everything after it
		for i := len(url) - 1; i >= 0; i-- {
			if url[i] == '/' {
				parts = url[i+1:]
				break
			}
		}
	}
	return parts
}

// Helper function to read file content
func readFileContent(file io.Reader) ([]byte, error) {
	return io.ReadAll(file)
}
