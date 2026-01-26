package api

import (
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
)

// handleGetNotifications handles getting user notifications
func handleGetNotifications(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}
