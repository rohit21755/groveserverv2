package ws

import (
	"github.com/go-chi/chi/v5"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

var globalHub *Hub

// SetupWSRoutes sets up WebSocket routes
func SetupWSRoutes(r chi.Router, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// Create global hub if not exists
	if globalHub == nil {
		globalHub = NewHub(redisClient, postgres)
		go globalHub.Run()
	}

	// Unified WebSocket connection endpoint (requires JWT token)
	// Connect via: ws://localhost:8080/ws/connect?token=JWT_TOKEN
	// Or: ws://localhost:8080/ws/connect with Authorization: Bearer JWT_TOKEN header
	r.Get("/connect", handleWSConnection(globalHub, cfg))

	// Legacy endpoints (kept for backward compatibility)
	r.Get("/leaderboard", handleLeaderboardWS(postgres, redisClient))
}

// GetHub returns the global WebSocket hub
func GetHub() *Hub {
	return globalHub
}
