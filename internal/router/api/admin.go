package api

import (
	"log"
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

// handleCreateTask handles creating a new task (admin)
func handleCreateTask(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleUpdateTask handles updating a task (admin)
func handleUpdateTask(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleGetSubmissions handles getting all submissions (admin)
func handleGetSubmissions(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleApproveSubmission handles approving a submission (admin)
func handleApproveSubmission(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleRejectSubmission handles rejecting a submission (admin)
func handleRejectSubmission(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// adminAuthMiddleware handles admin authentication
func adminAuthMiddleware(cfg *env.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement admin authentication
			log.Printf("Admin middleware: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
