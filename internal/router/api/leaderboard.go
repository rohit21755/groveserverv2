package api

import (
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
)

// handleGetPanIndiaLeaderboard handles getting the pan-India leaderboard
func handleGetPanIndiaLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleGetStateLeaderboard handles getting the state leaderboard
func handleGetStateLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleGetCollegeLeaderboard handles getting the college leaderboard
func handleGetCollegeLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}
