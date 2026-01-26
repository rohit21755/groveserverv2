package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleCreateState handles creating a new state (admin)
// @Summary      Create a new state
// @Description  Create a new state (admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        state  body      store.CreateStateRequest  true  "State information"
// @Success      201    {object}  store.State
// @Failure      400    {string}  string  "Bad request"
// @Failure      500    {string}  string  "Internal server error"
// @Router       /admin/states [post]
func handleCreateState(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req store.CreateStateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" || req.Code == "" {
			http.Error(w, "Name and code are required", http.StatusBadRequest)
			return
		}

		stateStore := store.NewStateStore(postgres)
		state, err := stateStore.CreateState(ctx, req)
		if err != nil {
			log.Printf("Error creating state: %v", err)
			http.Error(w, "Failed to create state", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(state); err != nil {
			log.Printf("Error encoding state: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
