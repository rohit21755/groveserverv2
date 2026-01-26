package api

import (
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
)

// handleGetTasks handles getting all tasks
func handleGetTasks(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleSubmitTask handles submitting a task
func handleSubmitTask(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}
