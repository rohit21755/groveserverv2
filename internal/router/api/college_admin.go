package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// handleCreateCollege handles creating a new college (admin)
// @Summary      Create a new college
// @Description  Create a new college (admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        college  body      store.CreateCollegeRequest  true  "College information"
// @Success      201      {object}  store.College
// @Failure      400      {string}  string  "Bad request"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /admin/colleges [post]
func handleCreateCollege(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req store.CreateCollegeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" || req.StateID == "" {
			http.Error(w, "Name and state_id are required", http.StatusBadRequest)
			return
		}

		collegeStore := store.NewCollegeStore(postgres)
		college, err := collegeStore.CreateCollege(ctx, req)
		if err != nil {
			log.Printf("Error creating college: %v", err)
			http.Error(w, "Failed to create college", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(college); err != nil {
			log.Printf("Error encoding college: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
