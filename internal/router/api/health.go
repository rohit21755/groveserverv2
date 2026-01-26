package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// HandleHealth handles the health check endpoint
// @Summary      Health check
// @Description  Check the health status of the API and its dependencies
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func HandleHealth(postgres *db.Postgres, redisClient *db.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		health := HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Services:  make(map[string]string),
		}

		// Check PostgreSQL
		if err := postgres.Ping(ctx); err != nil {
			health.Status = "unhealthy"
			health.Services["postgres"] = "down"
		} else {
			health.Services["postgres"] = "up"
		}

		// Check Redis
		if err := redisClient.Ping(ctx); err != nil {
			health.Status = "unhealthy"
			health.Services["redis"] = "down"
		} else {
			health.Services["redis"] = "up"
		}

		w.Header().Set("Content-Type", "application/json")

		if health.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
			return
		}
	}
}
