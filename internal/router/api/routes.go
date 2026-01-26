package api

import (
	"github.com/go-chi/chi/v5"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

// SetupAPIRoutes sets up all API routes
func SetupAPIRoutes(r chi.Router, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", handleLogin(postgres, cfg))
		r.Post("/register", handleRegister(postgres, cfg))
	})

	// User routes
	r.Route("/user", func(r chi.Router) {
		r.Get("/me", handleGetMe(postgres))
		r.Get("/{id}", handleGetUser(postgres))
		r.Post("/{id}/follow", handleFollow(postgres))
		r.Post("/{id}/unfollow", handleUnfollow(postgres))
		r.Post("/resume", handleUploadResume(postgres))
	})

	// Task routes
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", handleGetTasks(postgres))
		r.Post("/{id}/submit", handleSubmitTask(postgres))
	})

	// Feed routes
	r.Route("/feed", func(r chi.Router) {
		r.Get("/", handleGetFeed(postgres))
		r.Get("/user/{userId}", handleGetUserFeed(postgres))
		r.Post("/{feedId}/react", handleReactToFeed(postgres))
		r.Post("/{feedId}/comment", handleCommentOnFeed(postgres))
	})

	// Leaderboard routes
	r.Route("/leaderboard", func(r chi.Router) {
		r.Get("/pan-india", handleGetPanIndiaLeaderboard(postgres))
		r.Get("/state", handleGetStateLeaderboard(postgres))
		r.Get("/college", handleGetCollegeLeaderboard(postgres))
	})

	// Chat routes
	r.Route("/chat", func(r chi.Router) {
		r.Get("/rooms", handleGetChatRooms(postgres))
		r.Get("/rooms/{id}", handleGetChatRoom(postgres))
	})

	// Notification routes
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/", handleGetNotifications(postgres))
	})

	// State routes
	r.Route("/states", func(r chi.Router) {
		r.Get("/", handleGetStates(postgres))
		r.Get("/{stateId}/colleges", handleGetCollegesByState(postgres))
	})
}

// SetupAdminRoutes sets up all admin routes
func SetupAdminRoutes(r chi.Router, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// Admin middleware (authentication/authorization will be added)
	r.Use(adminAuthMiddleware(cfg))

	// State management - must be before other routes to avoid conflicts
	r.Route("/states", func(r chi.Router) {
		r.Get("/", handleGetStates(postgres))
		r.Post("/", handleCreateState(postgres))
	})

	// College management
	r.Route("/colleges", func(r chi.Router) {
		r.Post("/", handleCreateCollege(postgres))
	})

	// Task management
	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", handleCreateTask(postgres))
		r.Put("/{id}", handleUpdateTask(postgres))
	})

	// Submission management
	r.Route("/submissions", func(r chi.Router) {
		r.Get("/", handleGetSubmissions(postgres))
		r.Post("/{id}/approve", handleApproveSubmission(postgres))
		r.Post("/{id}/reject", handleRejectSubmission(postgres))
	})
}
