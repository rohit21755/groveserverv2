package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rohit21755/groveserverv2/internal/auth"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// CreateAdminRequest represents the request to create an admin
type CreateAdminRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateAdminResponse represents the response after creating an admin
type CreateAdminResponse struct {
	Admin *store.Admin `json:"admin"`
	Message string    `json:"message"`
}

// handleCreateAdmin handles creating a new admin user
// @Summary      Create admin
// @Description  Create a new admin user. This should be protected and only accessible by super admins.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        admin  body      CreateAdminRequest  true  "Admin information"
// @Success      201    {object}  CreateAdminResponse  "Admin created successfully"
// @Failure      400    {string}  string  "Bad request - invalid input or username already exists"
// @Failure      401    {string}  string  "Unauthorized"
// @Failure      500    {string}  string  "Internal server error"
// @Router       /admin/create [post]
func handleCreateAdmin(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request body
		var req CreateAdminRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding create admin request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" || req.Username == "" || req.Password == "" {
			http.Error(w, "name, username, and password are required", http.StatusBadRequest)
			return
		}

		// Validate password strength (minimum 8 characters)
		if len(req.Password) < 8 {
			http.Error(w, "password must be at least 8 characters long", http.StatusBadRequest)
			return
		}

		// Create admin store
		adminStore := store.NewAdminStore(postgres)

		// Create admin
		admin, err := adminStore.CreateAdmin(ctx, store.CreateAdminRequest{
			Name:     req.Name,
			Username: req.Username,
			Password: req.Password,
		})
		if err != nil {
			log.Printf("Error creating admin: %v", err)
			if err.Error() == "username already exists" {
				http.Error(w, "Username already exists", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to create admin: %v", err), http.StatusInternalServerError)
			return
		}

		// Return response
		response := CreateAdminResponse{
			Admin:   admin,
			Message: "Admin created successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding create admin response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// AdminLoginRequest represents the request to login as admin
type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminLoginResponse represents the response after admin login
type AdminLoginResponse struct {
	Token string       `json:"token"`
	Admin *store.Admin `json:"admin"`
}

// handleAdminLogin handles admin login
// @Summary      Admin login
// @Description  Login as admin and get JWT token
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        credentials  body      AdminLoginRequest  true  "Admin credentials"
// @Success      200          {object}  AdminLoginResponse  "Login successful"
// @Failure      400          {string}  string  "Bad request - invalid credentials"
// @Failure      401          {string}  string  "Unauthorized - invalid username or password"
// @Failure      500          {string}  string  "Internal server error"
// @Router       /admin/login [post]
func handleAdminLogin(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request body
		var req AdminLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding admin login request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Username == "" || req.Password == "" {
			http.Error(w, "username and password are required", http.StatusBadRequest)
			return
		}

		// Create admin store
		adminStore := store.NewAdminStore(postgres)

		// Get admin by username
		admin, err := adminStore.GetAdminByUsername(ctx, req.Username)
		if err != nil {
			log.Printf("Error getting admin: %v", err)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Verify password
		valid, err := adminStore.VerifyAdminPassword(ctx, req.Username, req.Password)
		if err != nil {
			log.Printf("Error verifying admin password: %v", err)
			http.Error(w, "Failed to verify credentials", http.StatusInternalServerError)
			return
		}

		if !valid {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Generate JWT token
		expiryDuration, err := auth.ParseExpiryDuration(cfg.JWTExpiry)
		if err != nil {
			log.Printf("Error parsing JWT expiry: %v", err)
			expiryDuration = 24 * time.Hour // Default to 24 hours
		}

		token, err := auth.GenerateToken(admin.ID, admin.Username, "admin", cfg.JWTSecret, expiryDuration)
		if err != nil {
			log.Printf("Error generating JWT token: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		response := AdminLoginResponse{
			Token: token,
			Admin: admin,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding admin login response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
