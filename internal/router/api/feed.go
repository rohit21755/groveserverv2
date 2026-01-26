package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
	"github.com/rohit21755/groveserverv2/internal/store"
)

// FeedResponse represents the paginated feed response
type FeedResponse struct {
	Items      []store.FeedItem `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
	FeedType   string           `json:"feed_type"` // "pan-india", "state", "college"
}

// handleGetFeed handles getting the task feed with pagination
// @Summary      Get feed
// @Description  Get feed items (pan-india, state, or college) with pagination. Only shows approved task submissions.
// @Tags         feed
// @Accept       json
// @Produce      json
// @Param        type      query     string  false  "Feed type: pan-india, state, college (default: pan-india)"
// @Param        page      query     int     false  "Page number (default: 1)"
// @Param        page_size query     int     false  "Items per page (default: 20, max: 100)"
// @Success      200       {object}  FeedResponse  "Feed items"
// @Failure      400       {string}  string  "Bad request"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/feed [get]
func handleGetFeed(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get feed type from query parameter (default: pan-india)
		feedTypeStr := r.URL.Query().Get("type")
		if feedTypeStr == "" {
			feedTypeStr = "pan-india"
		}

		// Validate feed type
		var feedType store.FeedType
		switch feedTypeStr {
		case "pan-india":
			feedType = store.FeedTypePanIndia
		case "state":
			feedType = store.FeedTypeState
		case "college":
			feedType = store.FeedTypeCollege
		default:
			http.Error(w, "Invalid feed type. Must be one of: pan-india, state, college", http.StatusBadRequest)
			return
		}

		// Get pagination parameters
		page := 1
		pageSize := 20

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

		// Get current user ID (optional - for state/college filtering and reaction checking)
		userID := ""
		if userIDFromCtx, ok := GetUserIDFromContext(ctx); ok {
			userID = userIDFromCtx
		}

		// For state/college feeds, userID is required
		if (feedType == store.FeedTypeState || feedType == store.FeedTypeCollege) && userID == "" {
			http.Error(w, "Authentication required for state/college feeds", http.StatusUnauthorized)
			return
		}

		// Create feed store
		feedStore := store.NewFeedStore(postgres)

		// Get feed items
		items, total, err := feedStore.GetFeed(ctx, store.GetFeedOptions{
			FeedType: feedType,
			UserID:   userID,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			log.Printf("Error getting feed: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get feed: %v", err), http.StatusInternalServerError)
			return
		}

		// Calculate total pages
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}

		// Return response
		response := FeedResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			FeedType:   feedTypeStr,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding feed response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetUserFeed handles getting a user's task feed
// @Summary      Get user feed
// @Description  Get feed items for a specific user (their completed tasks) with pagination.
// @Tags         feed
// @Accept       json
// @Produce      json
// @Param        userId    path      string  true   "User ID"
// @Param        page      query     int     false  "Page number (default: 1)"
// @Param        page_size query     int     false  "Items per page (default: 20, max: 100)"
// @Success      200       {object}  FeedResponse  "User feed items"
// @Failure      400       {string}  string  "Bad request"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/feed/user/{userId} [get]
func handleGetUserFeed(postgres *db.Postgres) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from URL path
		userID := chi.URLParam(r, "userId")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Get pagination parameters
		page := 1
		pageSize := 20

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

		// Create feed store
		feedStore := store.NewFeedStore(postgres)

		// Get user feed items
		items, total, err := feedStore.GetUserFeed(ctx, userID, page, pageSize)
		if err != nil {
			log.Printf("Error getting user feed: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get user feed: %v", err), http.StatusInternalServerError)
			return
		}

		// Calculate total pages
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}

		// Return response
		response := FeedResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			FeedType:   "user",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding user feed response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// ReactToFeedRequest represents the request to react to a feed item
type ReactToFeedRequest struct {
	Reaction string `json:"reaction"` // e.g., "like", "love", "fire", etc.
}

// handleReactToFeed handles reacting to a feed item
// @Summary      React to feed
// @Description  Add or update a reaction to a feed item. Protected route.
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        feedId    path      string            true   "Feed ID"
// @Param        request   body      ReactToFeedRequest  true  "Reaction details"
// @Success      200       {object}  map[string]string  "Reaction added successfully"
// @Failure      400       {string}  string  "Bad request"
// @Failure      401       {string}  string  "Unauthorized"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/feed/{feedId}/react [post]
func handleReactToFeed(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get feed ID from URL path
		feedID := chi.URLParam(r, "feedId")
		if feedID == "" {
			http.Error(w, "Feed ID is required", http.StatusBadRequest)
			return
		}

		// Parse request body
		var req ReactToFeedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding react request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Reaction == "" {
			http.Error(w, "Reaction is required", http.StatusBadRequest)
			return
		}

		// Create feed store
		feedStore := store.NewFeedStore(postgres)

		// Add reaction
		err := feedStore.AddReaction(ctx, feedID, userID, req.Reaction)
		if err != nil {
			log.Printf("Error adding reaction: %v", err)
			http.Error(w, fmt.Sprintf("Failed to add reaction: %v", err), http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Reaction added successfully",
			"feed_id": feedID,
			"reaction": req.Reaction,
		})
	}
}

// CommentOnFeedRequest represents the request to comment on a feed item
type CommentOnFeedRequest struct {
	Comment string `json:"comment"` // Comment text
}

// CommentResponse represents the response after adding a comment
type CommentResponse struct {
	Comment *store.FeedComment `json:"comment"`
	Message string             `json:"message"`
}

// handleCommentOnFeed handles commenting on a feed item
// @Summary      Comment on feed
// @Description  Add a comment to a feed item. Protected route.
// @Tags         feed
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        feedId    path      string              true   "Feed ID"
// @Param        request   body      CommentOnFeedRequest true  "Comment details"
// @Success      201       {object}  CommentResponse      "Comment added successfully"
// @Failure      400       {string}  string  "Bad request"
// @Failure      401       {string}  string  "Unauthorized"
// @Failure      500       {string}  string  "Internal server error"
// @Router       /api/feed/{feedId}/comment [post]
func handleCommentOnFeed(postgres *db.Postgres, cfg *env.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get user ID from context (set by JWT middleware)
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get feed ID from URL path
		feedID := chi.URLParam(r, "feedId")
		if feedID == "" {
			http.Error(w, "Feed ID is required", http.StatusBadRequest)
			return
		}

		// Parse request body
		var req CommentOnFeedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding comment request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Comment == "" {
			http.Error(w, "Comment is required", http.StatusBadRequest)
			return
		}

		// Create feed store
		feedStore := store.NewFeedStore(postgres)

		// Add comment
		comment, err := feedStore.AddComment(ctx, feedID, userID, req.Comment)
		if err != nil {
			log.Printf("Error adding comment: %v", err)
			http.Error(w, fmt.Sprintf("Failed to add comment: %v", err), http.StatusInternalServerError)
			return
		}

		// Return response
		response := CommentResponse{
			Comment: comment,
			Message: "Comment added successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding comment response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
