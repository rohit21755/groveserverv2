package api

import (
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
)

// handleGetChatRooms handles getting all chat rooms
func handleGetChatRooms(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}

// handleGetChatRoom handles getting a specific chat room
func handleGetChatRoom(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}
}
