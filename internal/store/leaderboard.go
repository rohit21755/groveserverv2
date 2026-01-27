package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type LeaderboardEntry struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	UserAvatar  string `json:"user_avatar,omitempty"`
	XP          int    `json:"xp"`
	Level       int    `json:"level"`
	StateName   string `json:"state_name,omitempty"`
	CollegeName string `json:"college_name,omitempty"`
}

type LeaderboardStore struct {
	postgres *db.Postgres
}

func NewLeaderboardStore(postgres *db.Postgres) *LeaderboardStore {
	return &LeaderboardStore{
		postgres: postgres,
	}
}

// GetPanIndiaLeaderboard retrieves the pan-India leaderboard
// period can be "all", "weekly", or "monthly" - defaults to "all"
func (s *LeaderboardStore) GetPanIndiaLeaderboard(ctx context.Context, limit, offset int, period string) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	if period == "" {
		period = "all"
	}

	var query string
	switch period {
	case "weekly":
		// Get XP earned in the last 7 days from xp_logs
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level
			FROM users u
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '7 days'
			WHERE u.role = 'student'
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $1 OFFSET $2
		`
	case "monthly":
		// Get XP earned in the last 30 days from xp_logs
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level
			FROM users u
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '30 days'
			WHERE u.role = 'student'
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $1 OFFSET $2
		`
	default: // "all"
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY u.xp DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				u.xp,
				u.level
			FROM users u
			WHERE u.role = 'student'
			ORDER BY u.xp DESC, u.created_at ASC
			LIMIT $1 OFFSET $2
		`
	}

	rows, err := s.postgres.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query pan-india leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		var userAvatar sql.NullString

		err := rows.Scan(
			&entry.Rank, &entry.UserID, &entry.UserName,
			&userAvatar, &entry.XP, &entry.Level,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}

		if userAvatar.Valid {
			entry.UserAvatar = userAvatar.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return entries, nil
}

// GetStateLeaderboard retrieves the state leaderboard
// period can be "all", "weekly", or "monthly" - defaults to "all"
func (s *LeaderboardStore) GetStateLeaderboard(ctx context.Context, stateID string, limit, offset int, period string) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	if period == "" {
		period = "all"
	}

	var query string
	switch period {
	case "weekly":
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level,
				s.name as state_name
			FROM users u
			INNER JOIN states s ON u.state_id = s.id
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '7 days'
			WHERE u.role = 'student' AND u.state_id = $1
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at, s.name
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	case "monthly":
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level,
				s.name as state_name
			FROM users u
			INNER JOIN states s ON u.state_id = s.id
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '30 days'
			WHERE u.role = 'student' AND u.state_id = $1
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at, s.name
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	default: // "all"
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY u.xp DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				u.xp,
				u.level,
				s.name as state_name
			FROM users u
			INNER JOIN states s ON u.state_id = s.id
			WHERE u.role = 'student' AND u.state_id = $1
			ORDER BY u.xp DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	}

	rows, err := s.postgres.DB.QueryContext(ctx, query, stateID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query state leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		var userAvatar sql.NullString
		var stateName sql.NullString

		err := rows.Scan(
			&entry.Rank, &entry.UserID, &entry.UserName,
			&userAvatar, &entry.XP, &entry.Level, &stateName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}

		if userAvatar.Valid {
			entry.UserAvatar = userAvatar.String
		}
		if stateName.Valid {
			entry.StateName = stateName.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return entries, nil
}

// GetCollegeLeaderboard retrieves the college leaderboard
// period can be "all", "weekly", or "monthly" - defaults to "all"
func (s *LeaderboardStore) GetCollegeLeaderboard(ctx context.Context, collegeID string, limit, offset int, period string) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	if period == "" {
		period = "all"
	}

	var query string
	switch period {
	case "weekly":
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level,
				c.name as college_name
			FROM users u
			INNER JOIN colleges c ON u.college_id = c.id
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '7 days'
			WHERE u.role = 'student' AND u.college_id = $1
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at, c.name
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	case "monthly":
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				COALESCE(SUM(xl.xp), 0) as xp,
				u.level,
				c.name as college_name
			FROM users u
			INNER JOIN colleges c ON u.college_id = c.id
			LEFT JOIN xp_logs xl ON u.id = xl.user_id 
				AND xl.created_at >= NOW() - INTERVAL '30 days'
			WHERE u.role = 'student' AND u.college_id = $1
			GROUP BY u.id, u.name, u.avatar_url, u.level, u.created_at, c.name
			ORDER BY COALESCE(SUM(xl.xp), 0) DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	default: // "all"
		query = `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY u.xp DESC, u.created_at ASC) as rank,
				u.id as user_id,
				u.name as user_name,
				u.avatar_url as user_avatar,
				u.xp,
				u.level,
				c.name as college_name
			FROM users u
			INNER JOIN colleges c ON u.college_id = c.id
			WHERE u.role = 'student' AND u.college_id = $1
			ORDER BY u.xp DESC, u.created_at ASC
			LIMIT $2 OFFSET $3
		`
	}

	rows, err := s.postgres.DB.QueryContext(ctx, query, collegeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query college leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		var userAvatar sql.NullString
		var collegeName sql.NullString

		err := rows.Scan(
			&entry.Rank, &entry.UserID, &entry.UserName,
			&userAvatar, &entry.XP, &entry.Level, &collegeName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}

		if userAvatar.Valid {
			entry.UserAvatar = userAvatar.String
		}
		if collegeName.Valid {
			entry.CollegeName = collegeName.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return entries, nil
}

// GetUserRank retrieves a user's rank in pan-india leaderboard
func (s *LeaderboardStore) GetUserRank(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*) + 1
		FROM users
		WHERE role = 'student'
		AND (xp > (SELECT xp FROM users WHERE id = $1)
		     OR (xp = (SELECT xp FROM users WHERE id = $1) 
		         AND created_at < (SELECT created_at FROM users WHERE id = $1)))
	`

	var rank int
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&rank)
	if err != nil {
		return 0, fmt.Errorf("failed to get user rank: %w", err)
	}

	return rank, nil
}
