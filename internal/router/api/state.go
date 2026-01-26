package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleGetStates handles getting all states
// @Summary      Get all states
// @Description  Retrieve a list of all states
// @Tags         states
// @Accept       json
// @Produce      json
// @Success      200  {array}   store.State
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/states [get]
func handleGetStates(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		stateStore := store.NewStateStore(postgres)
		states, err := stateStore.GetAllStates(ctx)
		if err != nil {
			log.Printf("Error fetching states: %v", err)
			http.Error(w, "Failed to fetch states", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(states); err != nil {
			log.Printf("Error encoding states: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
