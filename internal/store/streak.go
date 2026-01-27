package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type StreakStore struct {
	postgres *db.Postgres
}

func NewStreakStore(postgres *db.Postgres) *StreakStore {
	return &StreakStore{
		postgres: postgres,
	}
}

// UpdateStreak updates or creates a streak for a user
// This should be called daily when user is active
func (s *StreakStore) UpdateStreak(ctx context.Context, userID string) error {
	// Get current user streak info
	var streakStartedAt sql.NullTime
	var streakDays int
	query := `SELECT streak_started_at, streak_days FROM users WHERE id = $1`
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&streakStartedAt, &streakDays)
	if err != nil {
		return fmt.Errorf("failed to get user streak: %w", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var newStreakDays int
	var newStreakStartedAt time.Time

	if !streakStartedAt.Valid {
		// First streak - start today
		newStreakDays = 1
		newStreakStartedAt = today
	} else {
		lastActive := time.Date(
			streakStartedAt.Time.Year(),
			streakStartedAt.Time.Month(),
			streakStartedAt.Time.Day(),
			0, 0, 0, 0,
			streakStartedAt.Time.Location(),
		)
		daysDiff := int(today.Sub(lastActive).Hours() / 24)

		if daysDiff == 0 {
			// Already updated today, no change
			return nil
		} else if daysDiff == 1 {
			// Consecutive day - increment streak
			newStreakDays = streakDays + 1
			newStreakStartedAt = streakStartedAt.Time
		} else {
			// Streak broken - reset to 1
			newStreakDays = 1
			newStreakStartedAt = today
		}
	}

	// Update user streak
	updateQuery := `
		UPDATE users
		SET streak_started_at = $1, streak_days = $2
		WHERE id = $3
	`
	_, err = s.postgres.DB.ExecContext(ctx, updateQuery, newStreakStartedAt, newStreakDays, userID)
	if err != nil {
		return fmt.Errorf("failed to update streak: %w", err)
	}

	return nil
}

// GetUserStreak retrieves streak information for a user
func (s *StreakStore) GetUserStreak(ctx context.Context, userID string) (int, *time.Time, error) {
	var streakDays int
	var streakStartedAt sql.NullTime
	query := `SELECT streak_days, streak_started_at FROM users WHERE id = $1`
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&streakDays, &streakStartedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil, fmt.Errorf("user not found")
		}
		return 0, nil, fmt.Errorf("failed to get user streak: %w", err)
	}

	var startedAt *time.Time
	if streakStartedAt.Valid {
		startedAt = &streakStartedAt.Time
	}

	return streakDays, startedAt, nil
}

// RedeemStreakReward redeems a streak reward (XP and/or badge)
// This should be called when user wants to redeem their streak
func (s *StreakStore) RedeemStreakReward(ctx context.Context, userID string, streakDays int) (int, []string, error) {
	// Get streak badges that match this streak length
	badgeStore := NewBadgeStore(s.postgres)
	query := `
		SELECT id, xp
		FROM badges
		WHERE is_streak_badge = true
		AND (
			(rule LIKE '%' || $1 || '%' OR rule = '')
		)
		ORDER BY required_level ASC
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query, streakDays)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to query streak badges: %w", err)
	}
	defer rows.Close()

	var xpReward int
	var awardedBadgeIDs []string

	for rows.Next() {
		var badgeID string
		var badgeXP int
		err := rows.Scan(&badgeID, &badgeXP)
		if err != nil {
			continue
		}

		// Check if user already has this badge
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_badges WHERE user_id = $1 AND badge_id = $2)`
		err = s.postgres.DB.QueryRowContext(ctx, checkQuery, userID, badgeID).Scan(&exists)
		if err != nil || exists {
			continue
		}

		// Award badge and XP
		err = badgeStore.AwardBadge(ctx, userID, badgeID)
		if err == nil {
			awardedBadgeIDs = append(awardedBadgeIDs, badgeID)
			xpReward += badgeXP
		}
	}

	// Award XP for streak (base reward: 10 XP per day of streak, capped at 100 XP)
	baseXP := streakDays * 10
	if baseXP > 100 {
		baseXP = 100
	}
	xpReward += baseXP

	// Award XP if there's any reward
	if xpReward > 0 {
		xpStore := NewXPStore(s.postgres)
		_, err = xpStore.AwardXP(ctx, AwardXPRequest{
			UserID:   userID,
			XP:       xpReward,
			Source:   XPSourceDailyLogin, // Using daily login as source for streak rewards
			SourceID: "",
		})
		if err != nil {
			// Log error but don't fail - badges were already awarded
			return xpReward, awardedBadgeIDs, nil
		}
	}

	return xpReward, awardedBadgeIDs, nil
}
