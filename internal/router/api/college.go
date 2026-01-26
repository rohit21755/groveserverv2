package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleGetCollegesByState handles getting colleges by state ID
// @Summary      Get colleges by state
// @Description  Retrieve all colleges for a specific state
// @Tags         colleges
// @Accept       json
// @Produce      json
// @Param        stateId  path      string  true  "State ID"
// @Success      200      {array}   store.College
// @Failure      400      {string}  string  "Bad request"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/states/{stateId}/colleges [get]
func handleGetCollegesByState(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		stateID := chi.URLParam(r, "stateId")

		if stateID == "" {
			http.Error(w, "State ID is required", http.StatusBadRequest)
			return
		}

		collegeStore := store.NewCollegeStore(postgres)
		colleges, err := collegeStore.GetCollegesByStateID(ctx, stateID)
		if err != nil {
			log.Printf("Error fetching colleges: %v", err)
			http.Error(w, "Failed to fetch colleges", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(colleges); err != nil {
			log.Printf("Error encoding colleges: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
