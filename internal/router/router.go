package router

import (
	// "net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/router/api"
	"github.com/rohit21755/groveserverv2/internal/router/graphql"
	"github.com/rohit21755/groveserverv2/internal/router/ws"
	_ "github.com/rohit21755/groveserverv2/docs" // swagger docs
)

func SetupRoutes(r *chi.Mux, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	// Health check
	r.Get("/health", api.HandleHealth(postgres, redisClient))

	// API routes
	r.Route("/api", func(r chi.Router) {
		api.SetupAPIRoutes(r, postgres, redisClient, cfg)
	})

	// WebSocket routes
	r.Route("/ws", func(r chi.Router) {
		ws.SetupWSRoutes(r, postgres, redisClient, cfg)
	})

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		api.SetupAdminRoutes(r, postgres, redisClient, cfg)
	})

	// GraphQL routes
	graphql.SetupGraphQLRoutes(r, postgres, redisClient, cfg)
}
