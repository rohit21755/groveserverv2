package ws

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

func SetupWSRoutes(r chi.Router, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// WebSocket endpoints
	r.Get("/chat", handleChatWS(postgres, redisClient))
	r.Get("/leaderboard", handleLeaderboardWS(postgres, redisClient))
	r.Get("/notifications", handleNotificationsWS(postgres, redisClient))
}

// Placeholder WebSocket handlers - to be implemented
func handleChatWS(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("WebSocket not implemented"))
	}
}

func handleLeaderboardWS(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("WebSocket not implemented"))
	}
}

func handleNotificationsWS(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("WebSocket not implemented"))
	}
}
