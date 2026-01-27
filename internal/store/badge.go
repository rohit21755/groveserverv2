package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rohit21755/groveserverv2/internal/db"
)

type Badge struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Icon          string    `json:"icon,omitempty"`
	Rule          string    `json:"rule,omitempty"`
	XP            int       `json:"xp"`
	RequiredLevel int       `json:"required_level"`
	ImageURL      string    `json:"image_url,omitempty"`
	IsStreakBadge bool      `json:"is_streak_badge"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserBadge struct {
	UserID  string    `json:"user_id"`
	BadgeID string    `json:"badge_id"`
	Badge   *Badge    `json:"badge,omitempty"`
	EarnedAt time.Time `json:"earned_at"`
}

type BadgeStore struct {
	postgres *db.Postgres
}

func NewBadgeStore(postgres *db.Postgres) *BadgeStore {
	return &BadgeStore{
		postgres: postgres,
	}
}

// CreateBadgeRequest represents the request to create a badge
type CreateBadgeRequest struct {
	Name          string `json:"name"`
	Icon          string `json:"icon,omitempty"`
	Rule          string `json:"rule,omitempty"`
	XP            int    `json:"xp"`
	RequiredLevel int    `json:"required_level"`
	ImageURL      string `json:"image_url"`
	IsStreakBadge bool   `json:"is_streak_badge"`
}

// CreateBadge creates a new badge
func (s *BadgeStore) CreateBadge(ctx context.Context, req CreateBadgeRequest) (*Badge, error) {
	badgeID := uuid.New().String()
	query := `
		INSERT INTO badges (id, name, icon, rule, xp, required_level, image_url, is_streak_badge)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, icon, rule, xp, required_level, image_url, is_streak_badge, created_at
	`

	var badge Badge
	var icon, rule, imageURL sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query,
		badgeID, req.Name, req.Icon, req.Rule, req.XP, req.RequiredLevel, req.ImageURL, req.IsStreakBadge,
	).Scan(
		&badge.ID, &badge.Name, &icon, &rule, &badge.XP, &badge.RequiredLevel, &imageURL, &badge.IsStreakBadge, &badge.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create badge: %w", err)
	}

	if icon.Valid {
		badge.Icon = icon.String
	}
	if rule.Valid {
		badge.Rule = rule.String
	}
	if imageURL.Valid {
		badge.ImageURL = imageURL.String
	}

	return &badge, nil
}

// GetBadgeByID retrieves a badge by ID
func (s *BadgeStore) GetBadgeByID(ctx context.Context, badgeID string) (*Badge, error) {
	query := `
		SELECT id, name, icon, rule, xp, required_level, image_url, is_streak_badge, created_at
		FROM badges WHERE id = $1
	`

	var badge Badge
	var icon, rule, imageURL sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, badgeID).Scan(
		&badge.ID, &badge.Name, &icon, &rule, &badge.XP, &badge.RequiredLevel, &imageURL, &badge.IsStreakBadge, &badge.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("badge not found")
		}
		return nil, fmt.Errorf("failed to get badge: %w", err)
	}

	if icon.Valid {
		badge.Icon = icon.String
	}
	if rule.Valid {
		badge.Rule = rule.String
	}
	if imageURL.Valid {
		badge.ImageURL = imageURL.String
	}

	return &badge, nil
}

// GetUserBadges retrieves all badges earned by a user
func (s *BadgeStore) GetUserBadges(ctx context.Context, userID string) ([]UserBadge, error) {
	query := `
		SELECT 
			ub.user_id, ub.badge_id, ub.earned_at,
			b.id, b.name, b.icon, b.rule, b.xp, b.required_level, b.image_url, b.is_streak_badge, b.created_at
		FROM user_badges ub
		INNER JOIN badges b ON ub.badge_id = b.id
		WHERE ub.user_id = $1
		ORDER BY ub.earned_at DESC
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user badges: %w", err)
	}
	defer rows.Close()

	var userBadges []UserBadge
	for rows.Next() {
		var userBadge UserBadge
		var badge Badge
		var icon, rule, imageURL sql.NullString

		err := rows.Scan(
			&userBadge.UserID, &userBadge.BadgeID, &userBadge.EarnedAt,
			&badge.ID, &badge.Name, &icon, &rule, &badge.XP, &badge.RequiredLevel, &imageURL, &badge.IsStreakBadge, &badge.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user badge: %w", err)
		}

		if icon.Valid {
			badge.Icon = icon.String
		}
		if rule.Valid {
			badge.Rule = rule.String
		}
		if imageURL.Valid {
			badge.ImageURL = imageURL.String
		}

		userBadge.Badge = &badge
		userBadges = append(userBadges, userBadge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user badge rows: %w", err)
	}

	return userBadges, nil
}

// AwardBadge awards a badge to a user
func (s *BadgeStore) AwardBadge(ctx context.Context, userID, badgeID string) error {
	// Check if user already has this badge
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_badges WHERE user_id = $1 AND badge_id = $2)`
	err := s.postgres.DB.QueryRowContext(ctx, checkQuery, userID, badgeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing badge: %w", err)
	}

	if exists {
		return nil // User already has this badge, no error
	}

	// Award badge
	query := `INSERT INTO user_badges (user_id, badge_id) VALUES ($1, $2)`
	_, err = s.postgres.DB.ExecContext(ctx, query, userID, badgeID)
	if err != nil {
		return fmt.Errorf("failed to award badge: %w", err)
	}

	return nil
}

// CheckAndAwardBadges checks if user qualifies for any badges based on XP and level
func (s *BadgeStore) CheckAndAwardBadges(ctx context.Context, userID string, userXP, userLevel int) error {
	// Get all badges that user qualifies for but doesn't have yet
	query := `
		SELECT b.id, b.xp, b.required_level
		FROM badges b
		WHERE b.is_streak_badge = false
		AND (b.xp <= $1 OR b.required_level <= $2)
		AND NOT EXISTS (
			SELECT 1 FROM user_badges ub 
			WHERE ub.user_id = $3 AND ub.badge_id = b.id
		)
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, userXP, userLevel, userID)
	if err != nil {
		return fmt.Errorf("failed to query qualifying badges: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var badgeID string
		var badgeXP, badgeLevel int
		err := rows.Scan(&badgeID, &badgeXP, &badgeLevel)
		if err != nil {
			continue // Skip this badge if scan fails
		}

		// Check if user actually qualifies (both XP and level requirements)
		if userXP >= badgeXP && userLevel >= badgeLevel {
			// Award badge
			err = s.AwardBadge(ctx, userID, badgeID)
			if err != nil {
				// Log error but continue with other badges
				continue
			}
		}
	}

	return rows.Err()
}

// GetAllBadges retrieves all badges (for admin)
func (s *BadgeStore) GetAllBadges(ctx context.Context) ([]Badge, error) {
	query := `
		SELECT id, name, icon, rule, xp, required_level, image_url, is_streak_badge, created_at
		FROM badges
		ORDER BY created_at DESC
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query badges: %w", err)
	}
	defer rows.Close()

	var badges []Badge
	for rows.Next() {
		var badge Badge
		var icon, rule, imageURL sql.NullString

		err := rows.Scan(
			&badge.ID, &badge.Name, &icon, &rule, &badge.XP, &badge.RequiredLevel, &imageURL, &badge.IsStreakBadge, &badge.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan badge: %w", err)
		}

		if icon.Valid {
			badge.Icon = icon.String
		}
		if rule.Valid {
			badge.Rule = rule.String
		}
		if imageURL.Valid {
			badge.ImageURL = imageURL.String
		}

		badges = append(badges, badge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating badge rows: %w", err)
	}

	return badges, nil
}
