package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// LeaderboardResponse represents the leaderboard response
type LeaderboardResponse struct {
	Entries   []store.LeaderboardEntry `json:"entries"`
	Type      string                   `json:"type"`       // "pan-india", "state", "college"
	ScopeID   string                   `json:"scope_id,omitempty"` // state_id or college_id
	Page      int                      `json:"page"`
	PageSize  int                      `json:"page_size"`
	Total     int                      `json:"total,omitempty"` // Optional: total count
}

// handleGetPanIndiaLeaderboard handles getting the pan-India leaderboard
// @Summary      Get pan-India leaderboard
// @Description  Get the pan-India leaderboard with pagination. Shows top users by XP across all states and colleges.
// @Tags         leaderboard
// @Accept       json
// @Produce      json
// @Param        page      query     int     false  "Page number (default: 1)"
// @Param        page_size query     int     false  "Items per page (default: 100, max: 1000)"
// @Success      200       {object}  LeaderboardResponse  "Leaderboard entries"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/leaderboard/pan-india [get]
func handleGetPanIndiaLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get pagination parameters
		page := 1
		pageSize := 100

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
		}

		offset := (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}

		// Create leaderboard store
		leaderboardStore := store.NewLeaderboardStore(postgres)

		// Get leaderboard entries
		entries, err := leaderboardStore.GetPanIndiaLeaderboard(ctx, pageSize, offset)
		if err != nil {
			log.Printf("Error getting pan-india leaderboard: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
			return
		}

		// Adjust ranks based on offset
		for i := range entries {
			entries[i].Rank = offset + i + 1
		}

		// Return response
		response := LeaderboardResponse{
			Entries:  entries,
			Type:     "pan-india",
			Page:     page,
			PageSize: pageSize,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding leaderboard response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetStateLeaderboard handles getting the state leaderboard
// @Summary      Get state leaderboard
// @Description  Get the state leaderboard with pagination. Shows top users by XP within a specific state.
// @Tags         leaderboard
// @Accept       json
// @Produce      json
// @Param        state_id  query     string  true   "State ID"
// @Param        page      query     int     false  "Page number (default: 1)"
// @Param        page_size query     int     false  "Items per page (default: 100, max: 1000)"
// @Success      200       {object}  LeaderboardResponse  "Leaderboard entries"
// @Failure      400       {string}  string  "Bad request - state_id required"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/leaderboard/state [get]
func handleGetStateLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get state ID from query parameter
		stateID := r.URL.Query().Get("state_id")
		if stateID == "" {
			http.Error(w, "state_id query parameter is required", http.StatusBadRequest)
			return
		}

		// Get pagination parameters
		page := 1
		pageSize := 100

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
		}

		offset := (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}

		// Create leaderboard store
		leaderboardStore := store.NewLeaderboardStore(postgres)

		// Get leaderboard entries
		entries, err := leaderboardStore.GetStateLeaderboard(ctx, stateID, pageSize, offset)
		if err != nil {
			log.Printf("Error getting state leaderboard: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
			return
		}

		// Adjust ranks based on offset
		for i := range entries {
			entries[i].Rank = offset + i + 1
		}

		// Return response
		response := LeaderboardResponse{
			Entries:  entries,
			Type:     "state",
			ScopeID:  stateID,
			Page:     page,
			PageSize: pageSize,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding leaderboard response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetCollegeLeaderboard handles getting the college leaderboard
// @Summary      Get college leaderboard
// @Description  Get the college leaderboard with pagination. Shows top users by XP within a specific college.
// @Tags         leaderboard
// @Accept       json
// @Produce      json
// @Param        college_id query     string  true   "College ID"
// @Param        page       query     int     false  "Page number (default: 1)"
// @Param        page_size  query     int     false  "Items per page (default: 100, max: 1000)"
// @Success      200        {object}  LeaderboardResponse  "Leaderboard entries"
// @Failure      400        {string}  string  "Bad request - college_id required"
// @Failure      500        {string}  string  "Internal server error"
// @Router       /api/leaderboard/college [get]
func handleGetCollegeLeaderboard(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get college ID from query parameter
		collegeID := r.URL.Query().Get("college_id")
		if collegeID == "" {
			http.Error(w, "college_id query parameter is required", http.StatusBadRequest)
			return
		}

		// Get pagination parameters
		page := 1
		pageSize := 100

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
		}

		offset := (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}

		// Create leaderboard store
		leaderboardStore := store.NewLeaderboardStore(postgres)

		// Get leaderboard entries
		entries, err := leaderboardStore.GetCollegeLeaderboard(ctx, collegeID, pageSize, offset)
		if err != nil {
			log.Printf("Error getting college leaderboard: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
			return
		}

		// Adjust ranks based on offset
		for i := range entries {
			entries[i].Rank = offset + i + 1
		}

		// Return response
		response := LeaderboardResponse{
			Entries:  entries,
			Type:     "college",
			ScopeID:  collegeID,
			Page:     page,
			PageSize: pageSize,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding leaderboard response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
