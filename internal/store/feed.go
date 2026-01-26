package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rohit21755/groveserverv2/internal/db"
)

type FeedItem struct {
	ID           string    `json:"id"`
	SubmissionID string    `json:"submission_id"`
	UserID       string    `json:"user_id"`
	TaskID       string    `json:"task_id"`
	UserName     string    `json:"user_name"`
	UserAvatar   string    `json:"user_avatar,omitempty"`
	TaskTitle    string    `json:"task_title"`
	TaskXP       int       `json:"task_xp"`
	ProofURL     string    `json:"proof_url"`
	ReactionCount int      `json:"reaction_count"`
	CommentCount  int      `json:"comment_count"`
	UserReacted   bool     `json:"user_reacted,omitempty"` // Whether current user reacted
	CreatedAt     time.Time `json:"created_at"`
}

type FeedReaction struct {
	FeedID    string    `json:"feed_id"`
	UserID    string    `json:"user_id"`
	Reaction  string    `json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
}

type FeedComment struct {
	ID        string    `json:"id"`
	FeedID    string    `json:"feed_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserAvatar string   `json:"user_avatar,omitempty"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

type FeedStore struct {
	postgres *db.Postgres
}

func NewFeedStore(postgres *db.Postgres) *FeedStore {
	return &FeedStore{
		postgres: postgres,
	}
}

// FeedType represents the type of feed
type FeedType string

const (
	FeedTypePanIndia FeedType = "pan-india" // All approved submissions
	FeedTypeState    FeedType = "state"     // Submissions from users in same state
	FeedTypeCollege  FeedType = "college"   // Submissions from users in same college
)

// GetFeedOptions represents options for getting feed
type GetFeedOptions struct {
	FeedType FeedType // pan-india, state, college
	UserID   string   // Current user ID (for filtering by state/college and checking reactions)
	Page     int      // Page number (1-based)
	PageSize int      // Items per page
}

// GetFeed retrieves feed items with pagination
func (s *FeedStore) GetFeed(ctx context.Context, opts GetFeedOptions) ([]FeedItem, int, error) {
	offset := (opts.Page - 1) * opts.PageSize
	if offset < 0 {
		offset = 0
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 20 // Default page size
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100 // Max page size
	}

	var query string
	var args []interface{}
	argIndex := 1

	// Base query - only approved submissions
	baseQuery := `
		FROM completed_task_feed ctf
		INNER JOIN submissions s ON ctf.submission_id = s.id
		INNER JOIN tasks t ON ctf.task_id = t.id
		INNER JOIN users u ON ctf.user_id = u.id
		WHERE s.status = 'approved'
	`

	// Add filtering based on feed type
	switch opts.FeedType {
	case FeedTypeState:
		// Get user's state_id
		var stateID sql.NullString
		userQuery := `SELECT state_id FROM users WHERE id = $1`
		err := s.postgres.DB.QueryRowContext(ctx, userQuery, opts.UserID).Scan(&stateID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user state: %w", err)
		}
		if stateID.Valid {
			baseQuery += fmt.Sprintf(" AND u.state_id = $%d", argIndex)
			args = append(args, stateID.String)
			argIndex++
		}
	case FeedTypeCollege:
		// Get user's college_id
		var collegeID sql.NullString
		userQuery := `SELECT college_id FROM users WHERE id = $1`
		err := s.postgres.DB.QueryRowContext(ctx, userQuery, opts.UserID).Scan(&collegeID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user college: %w", err)
		}
		if collegeID.Valid {
			baseQuery += fmt.Sprintf(" AND u.college_id = $%d", argIndex)
			args = append(args, collegeID.String)
			argIndex++
		}
	// FeedTypePanIndia - no additional filtering needed
	}

	// Count total items
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	err := s.postgres.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count feed items: %w", err)
	}

	// Get feed items with reactions and comments count
	selectQuery := `
		SELECT 
			ctf.id,
			ctf.submission_id,
			ctf.user_id,
			ctf.task_id,
			u.name as user_name,
			u.avatar_url as user_avatar,
			t.title as task_title,
			t.xp as task_xp,
			s.proof_url,
			COALESCE(reaction_counts.count, 0) as reaction_count,
			COALESCE(comment_counts.count, 0) as comment_count,
			ctf.created_at
		` + baseQuery + `
		LEFT JOIN (
			SELECT feed_id, COUNT(*) as count
			FROM task_feed_reactions
			GROUP BY feed_id
		) reaction_counts ON ctf.id = reaction_counts.feed_id
		LEFT JOIN (
			SELECT feed_id, COUNT(*) as count
			FROM task_feed_comments
			GROUP BY feed_id
		) comment_counts ON ctf.id = comment_counts.feed_id
		ORDER BY ctf.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	
	args = append(args, opts.PageSize, offset)

	rows, err := s.postgres.DB.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query feed: %w", err)
	}
	defer rows.Close()

	var feedItems []FeedItem
	for rows.Next() {
		var item FeedItem
		var userAvatar sql.NullString

		err := rows.Scan(
			&item.ID, &item.SubmissionID, &item.UserID, &item.TaskID,
			&item.UserName, &userAvatar, &item.TaskTitle, &item.TaskXP,
			&item.ProofURL, &item.ReactionCount, &item.CommentCount, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan feed item: %w", err)
		}

		if userAvatar.Valid {
			item.UserAvatar = userAvatar.String
		}

		// Check if current user reacted (if userID provided)
		if opts.UserID != "" {
			var reacted bool
			reactionQuery := `SELECT EXISTS(SELECT 1 FROM task_feed_reactions WHERE feed_id = $1 AND user_id = $2)`
			err := s.postgres.DB.QueryRowContext(ctx, reactionQuery, item.ID, opts.UserID).Scan(&reacted)
			if err == nil {
				item.UserReacted = reacted
			}
		}

		feedItems = append(feedItems, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating feed rows: %w", err)
	}

	return feedItems, total, nil
}

// GetUserFeed retrieves feed items for a specific user
func (s *FeedStore) GetUserFeed(ctx context.Context, userID string, page, pageSize int) ([]FeedItem, int, error) {
	opts := GetFeedOptions{
		FeedType: FeedTypePanIndia, // Not used for user feed
		UserID:   "",                // Not needed for user feed
		Page:     page,
		PageSize: pageSize,
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Count total items
	countQuery := `
		SELECT COUNT(*)
		FROM completed_task_feed ctf
		INNER JOIN submissions s ON ctf.submission_id = s.id
		WHERE ctf.user_id = $1 AND s.status = 'approved'
	`
	var total int
	err := s.postgres.DB.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user feed items: %w", err)
	}

	// Get feed items
	query := `
		SELECT 
			ctf.id,
			ctf.submission_id,
			ctf.user_id,
			ctf.task_id,
			u.name as user_name,
			u.avatar_url as user_avatar,
			t.title as task_title,
			t.xp as task_xp,
			s.proof_url,
			COALESCE(reaction_counts.count, 0) as reaction_count,
			COALESCE(comment_counts.count, 0) as comment_count,
			ctf.created_at
		FROM completed_task_feed ctf
		INNER JOIN submissions s ON ctf.submission_id = s.id
		INNER JOIN tasks t ON ctf.task_id = t.id
		INNER JOIN users u ON ctf.user_id = u.id
		LEFT JOIN (
			SELECT feed_id, COUNT(*) as count
			FROM task_feed_reactions
			GROUP BY feed_id
		) reaction_counts ON ctf.id = reaction_counts.feed_id
		LEFT JOIN (
			SELECT feed_id, COUNT(*) as count
			FROM task_feed_comments
			GROUP BY feed_id
		) comment_counts ON ctf.id = comment_counts.feed_id
		WHERE ctf.user_id = $1 AND s.status = 'approved'
		ORDER BY ctf.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user feed: %w", err)
	}
	defer rows.Close()

	var feedItems []FeedItem
	for rows.Next() {
		var item FeedItem
		var userAvatar sql.NullString

		err := rows.Scan(
			&item.ID, &item.SubmissionID, &item.UserID, &item.TaskID,
			&item.UserName, &userAvatar, &item.TaskTitle, &item.TaskXP,
			&item.ProofURL, &item.ReactionCount, &item.CommentCount, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan feed item: %w", err)
		}

		if userAvatar.Valid {
			item.UserAvatar = userAvatar.String
		}

		feedItems = append(feedItems, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating feed rows: %w", err)
	}

	return feedItems, total, nil
}

// CreateFeedEntry creates a feed entry when submission is approved
func (s *FeedStore) CreateFeedEntry(ctx context.Context, submissionID, userID, taskID string) error {
	// Check if feed entry already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM completed_task_feed WHERE submission_id = $1)`
	err := s.postgres.DB.QueryRowContext(ctx, checkQuery, submissionID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing feed entry: %w", err)
	}

	if exists {
		return nil // Already exists, no need to create
	}

	// Create feed entry
	feedID := uuid.New().String()
	query := `
		INSERT INTO completed_task_feed (id, submission_id, user_id, task_id, visibility)
		VALUES ($1, $2, $3, $4, 'public')
	`
	_, err = s.postgres.DB.ExecContext(ctx, query, feedID, submissionID, userID, taskID)
	if err != nil {
		return fmt.Errorf("failed to create feed entry: %w", err)
	}

	return nil
}

// AddReaction adds a reaction to a feed item
func (s *FeedStore) AddReaction(ctx context.Context, feedID, userID, reaction string) error {
	// Check if user already reacted
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM task_feed_reactions WHERE feed_id = $1 AND user_id = $2)`
	err := s.postgres.DB.QueryRowContext(ctx, checkQuery, feedID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing reaction: %w", err)
	}

	if exists {
		// Update existing reaction
		query := `UPDATE task_feed_reactions SET reaction = $1 WHERE feed_id = $2 AND user_id = $3`
		_, err = s.postgres.DB.ExecContext(ctx, query, reaction, feedID, userID)
		if err != nil {
			return fmt.Errorf("failed to update reaction: %w", err)
		}
		return nil
	}

	// Create new reaction
	query := `INSERT INTO task_feed_reactions (feed_id, user_id, reaction) VALUES ($1, $2, $3)`
	_, err = s.postgres.DB.ExecContext(ctx, query, feedID, userID, reaction)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	return nil
}

// RemoveReaction removes a reaction from a feed item
func (s *FeedStore) RemoveReaction(ctx context.Context, feedID, userID string) error {
	query := `DELETE FROM task_feed_reactions WHERE feed_id = $1 AND user_id = $2`
	_, err := s.postgres.DB.ExecContext(ctx, query, feedID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}
	return nil
}

// AddComment adds a comment to a feed item
func (s *FeedStore) AddComment(ctx context.Context, feedID, userID, comment string) (*FeedComment, error) {
	commentID := uuid.New().String()
	query := `
		INSERT INTO task_feed_comments (id, feed_id, user_id, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, feed_id, user_id, comment, created_at
	`

	var feedComment FeedComment
	err := s.postgres.DB.QueryRowContext(ctx, query, commentID, feedID, userID, comment).Scan(
		&feedComment.ID, &feedComment.FeedID, &feedComment.UserID, &feedComment.Comment, &feedComment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	// Get user info for comment
	userQuery := `SELECT name, avatar_url FROM users WHERE id = $1`
	var userName string
	var userAvatar sql.NullString
	err = s.postgres.DB.QueryRowContext(ctx, userQuery, userID).Scan(&userName, &userAvatar)
	if err == nil {
		feedComment.UserName = userName
		if userAvatar.Valid {
			feedComment.UserAvatar = userAvatar.String
		}
	}

	return &feedComment, nil
}

// GetComments retrieves comments for a feed item
func (s *FeedStore) GetComments(ctx context.Context, feedID string, limit int) ([]FeedComment, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `
		SELECT 
			tfc.id,
			tfc.feed_id,
			tfc.user_id,
			u.name as user_name,
			u.avatar_url as user_avatar,
			tfc.comment,
			tfc.created_at
		FROM task_feed_comments tfc
		INNER JOIN users u ON tfc.user_id = u.id
		WHERE tfc.feed_id = $1
		ORDER BY tfc.created_at ASC
		LIMIT $2
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, feedID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []FeedComment
	for rows.Next() {
		var comment FeedComment
		var userAvatar sql.NullString

		err := rows.Scan(
			&comment.ID, &comment.FeedID, &comment.UserID,
			&comment.UserName, &userAvatar, &comment.Comment, &comment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		if userAvatar.Valid {
			comment.UserAvatar = userAvatar.String
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comment rows: %w", err)
	}

	return comments, nil
}
