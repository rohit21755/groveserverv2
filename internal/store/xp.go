package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rohit21755/groveserverv2/internal/db"
)

// XPSource represents the source of XP
type XPSource string

const (
	XPSourceTaskApproval XPSource = "task_approval" // XP from task submission approval
	XPSourceReferral     XPSource = "referral"      // XP from referring users
	XPSourceDailyLogin   XPSource = "daily_login"   // XP from daily login
	XPSourceFeedPost     XPSource = "feed_post"     // XP from posting on feed
	XPSourceFeedReaction XPSource = "feed_reaction" // XP from reacting to feed
	XPSourceComment      XPSource = "comment"       // XP from commenting
	XPSourceAdminGrant   XPSource = "admin_grant"   // XP added by admin (manual grant)
	XPSourceUserAdd      XPSource = "user_add"      // XP added by user to own account (e.g. redeem code, claim reward)
	// Add more sources as needed in the future
)

type XPLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Source    string    `json:"source"`
	SourceID  string    `json:"source_id,omitempty"`
	XP        int       `json:"xp"`
	CreatedAt time.Time `json:"created_at"`
}

type XPStore struct {
	postgres *db.Postgres
}

func NewXPStore(postgres *db.Postgres) *XPStore {
	return &XPStore{
		postgres: postgres,
	}
}

// AwardXPRequest represents the request to award XP
type AwardXPRequest struct {
	UserID   string   `json:"user_id"`
	XP       int      `json:"xp"`
	Source   XPSource `json:"source"`
	SourceID string   `json:"source_id,omitempty"` // Optional: ID of the source (e.g., task_id, submission_id)
}

// AwardXP awards XP to a user and logs it
// This is a transactional operation that:
// 1. Updates the user's XP in the users table
// 2. Logs the XP award in the xp_logs table
func (s *XPStore) AwardXP(ctx context.Context, req AwardXPRequest) (*XPLog, error) {
	if req.XP <= 0 {
		return nil, fmt.Errorf("XP amount must be greater than 0")
	}

	// Start transaction
	tx, err := s.postgres.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update user's XP
	updateQuery := `
		UPDATE users
		SET xp = xp + $1
		WHERE id = $2
		RETURNING xp
	`
	var newXP int
	err = tx.QueryRowContext(ctx, updateQuery, req.XP, req.UserID).Scan(&newXP)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to update user XP: %w", err)
	}

	// Log XP award
	logID := uuid.New().String()
	var sourceID sql.NullString
	if req.SourceID != "" {
		sourceID = sql.NullString{String: req.SourceID, Valid: true}
	}

	logQuery := `
		INSERT INTO xp_logs (id, user_id, source, source_id, xp)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, source, source_id, xp, created_at
	`

	var xpLog XPLog
	var logSourceID sql.NullString

	err = tx.QueryRowContext(ctx, logQuery,
		logID, req.UserID, string(req.Source), sourceID, req.XP,
	).Scan(
		&xpLog.ID, &xpLog.UserID, &xpLog.Source, &logSourceID, &xpLog.XP, &xpLog.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to log XP award: %w", err)
	}

	if logSourceID.Valid {
		xpLog.SourceID = logSourceID.String
	}

	// Get user's current level (for badge checking)
	var userLevel int
	levelQuery := `SELECT level FROM users WHERE id = $1`
	err = tx.QueryRowContext(ctx, levelQuery, req.UserID).Scan(&userLevel)
	if err != nil {
		// Log error but don't fail - level check is not critical
		userLevel = 1
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Check and award badges based on new XP and level (after commit)
	// This is done outside the transaction to avoid long-running transactions
	badgeStore := NewBadgeStore(s.postgres)
	err = badgeStore.CheckAndAwardBadges(ctx, req.UserID, newXP, userLevel)
	if err != nil {
		// Log error but don't fail - badge awarding is not critical
		// In production, you might want to use a queue/retry mechanism
	}

	return &xpLog, nil
}

// GetXPLogs retrieves XP logs for a user
func (s *XPStore) GetXPLogs(ctx context.Context, userID string, limit int) ([]XPLog, error) {
	query := `
		SELECT id, user_id, source, source_id, xp, created_at
		FROM xp_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query XP logs: %w", err)
	}
	defer rows.Close()

	var logs []XPLog
	for rows.Next() {
		var log XPLog
		var sourceID sql.NullString

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Source, &sourceID, &log.XP, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan XP log: %w", err)
		}

		if sourceID.Valid {
			log.SourceID = sourceID.String
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating XP log rows: %w", err)
	}

	return logs, nil
}

// GetUserTotalXP retrieves the current total XP for a user
func (s *XPStore) GetUserTotalXP(ctx context.Context, userID string) (int, error) {
	query := `SELECT xp FROM users WHERE id = $1`

	var xp int
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&xp)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found")
		}
		return 0, fmt.Errorf("failed to get user XP: %w", err)
	}

	return xp, nil
}
