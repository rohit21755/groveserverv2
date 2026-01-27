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

	// User routes (protected with JWT)
	r.Route("/user", func(r chi.Router) {
		r.Use(JWTAuthMiddleware(cfg))
		r.Get("/me", handleGetMe(postgres))
		r.Get("/{id}", handleGetUser(postgres))
		r.Post("/{id}/follow", handleFollow(postgres))
		r.Post("/{id}/unfollow", handleUnfollow(postgres))
		// Resume routes
		r.Post("/resume", handleUploadResume(postgres, cfg))
		r.Put("/resume", handleUpdateResume(postgres, cfg))
		// Profile picture routes
		r.Post("/profile-pic", handleUploadProfilePic(postgres, cfg))
		r.Put("/profile-pic", handleUpdateProfilePic(postgres, cfg))
	})

	// Task routes (protected with JWT)
	r.Route("/tasks", func(r chi.Router) {
		r.Use(JWTAuthMiddleware(cfg))
		r.Get("/", handleGetTasks(postgres))
		r.Post("/{id}/submit", handleSubmitTask(postgres, cfg))
	})

	// Feed routes
	r.Route("/feed", func(r chi.Router) {
		r.Get("/", handleGetFeed(postgres, cfg))             // Public, but can use JWT for state/college filtering
		r.Get("/user/{userId}", handleGetUserFeed(postgres)) // Public
		// Protected routes for reactions and comments
		r.Group(func(r chi.Router) {
			r.Use(JWTAuthMiddleware(cfg))
			r.Post("/{feedId}/react", handleReactToFeed(postgres, cfg))
			r.Post("/{feedId}/comment", handleCommentOnFeed(postgres, cfg))
		})
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
	// Admin authentication routes (public - no auth required)
	r.Post("/login", handleAdminLogin(postgres, cfg))

	// Protected admin routes (require JWT authentication)
	r.Group(func(r chi.Router) {
		// Use JWT middleware for admin routes
		r.Use(JWTAuthMiddleware(cfg))
		// Admin middleware (authorization/role checking will be added)
		r.Use(adminAuthMiddleware(cfg))

		// Admin management
		r.Post("/create", handleCreateAdmin(postgres))

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
			r.Post("/", handleCreateTask(postgres, redisClient))
			r.Put("/{id}", handleUpdateTask(postgres))
		})

		// Submission management
		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", handleGetSubmissions(postgres))
			r.Post("/{id}/approve", handleApproveSubmission(postgres, redisClient))
			r.Post("/{id}/reject", handleRejectSubmission(postgres))
		})
	})
}
