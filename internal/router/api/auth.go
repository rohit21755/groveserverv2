package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/rohit21755/groveserverv2/internal/auth"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/storage"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string       `json:"token"`
	User  *store.User  `json:"user"`
}

// handleLogin handles user login
// @Summary      User login
// @Description  Authenticate user with email and password, return JWT token and user data
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "Login credentials"
// @Success      200         {object}  LoginResponse  "Login successful"
// @Failure      400         {string}  string  "Bad request - invalid input"
// @Failure      401         {string}  string  "Invalid credentials"
// @Failure      500         {string}  string  "Internal server error"
// @Router       /api/auth/login [post]
func handleLogin(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request body
		var loginReq LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			log.Printf("Error decoding login request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if loginReq.Email == "" || loginReq.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		// Create user store
		userStore := store.NewUserStore(postgres)

		// Get password hash
		passwordHash, err := userStore.GetUserPasswordHash(ctx, loginReq.Email)
		if err != nil {
			log.Printf("Error getting password hash: %v", err)
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Verify password
		if !userStore.VerifyPassword(passwordHash, loginReq.Password) {
			log.Printf("Invalid password for email: %s", loginReq.Email)
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Get user details
		user, err := userStore.GetUserByEmail(ctx, loginReq.Email)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
			return
		}

		// Parse JWT expiry duration
		expiryDuration, err := auth.ParseExpiryDuration(cfg.JWTExpiry)
		if err != nil {
			log.Printf("Error parsing JWT expiry, using default 24h: %v", err)
			expiryDuration = 24 * time.Hour
		}

		// Generate JWT token
		token, err := auth.GenerateToken(user.ID, user.Email, user.Role, cfg.JWTSecret, expiryDuration)
		if err != nil {
			log.Printf("Error generating token: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		response := LoginResponse{
			Token: token,
			User:  user,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding login response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	Token string       `json:"token"`
	User  *store.User  `json:"user"`
}

// handleRegister handles user registration
// @Summary      User registration
// @Description  Register a new user account. Each user gets a unique referral code automatically. Referral code (input), resume, and profile picture are optional. Returns JWT token for automatic login.
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
// @Success      201           {object}  RegisterResponse  "User created with auto-generated referral_code and JWT token"
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

		// Parse JWT expiry duration
		expiryDuration, err := auth.ParseExpiryDuration(cfg.JWTExpiry)
		if err != nil {
			log.Printf("Error parsing JWT expiry, using default 24h: %v", err)
			expiryDuration = 24 * time.Hour
		}

		// Generate JWT token for automatic login after registration
		token, err := auth.GenerateToken(user.ID, user.Email, user.Role, cfg.JWTSecret, expiryDuration)
		if err != nil {
			log.Printf("Error generating token after registration: %v", err)
			// Still return user data even if token generation fails
			// But log the error for debugging
		}

		// Return response with token and user
		response := RegisterResponse{
			Token: token,
			User:  user,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
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
