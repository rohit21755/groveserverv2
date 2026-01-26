package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type College struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	StateID string `json:"state_id"`
	City    string `json:"city,omitempty"`
}

type CollegeStore struct {
	postgres *db.Postgres
}

func NewCollegeStore(postgres *db.Postgres) *CollegeStore {
	return &CollegeStore{
		postgres: postgres,
	}
}

// GetCollegesByStateID retrieves all colleges for a given state ID
func (s *CollegeStore) GetCollegesByStateID(ctx context.Context, stateID string) ([]College, error) {
	query := `SELECT id, name, state_id, city FROM colleges WHERE state_id = $1 ORDER BY name ASC`
	rows, err := s.postgres.DB.QueryContext(ctx, query, stateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query colleges: %w", err)
	}
	defer rows.Close()

	var colleges []College
	for rows.Next() {
		var college College
		var city sql.NullString
		if err := rows.Scan(&college.ID, &college.Name, &college.StateID, &city); err != nil {
			return nil, fmt.Errorf("failed to scan college: %w", err)
		}
		if city.Valid {
			college.City = city.String
		}
		colleges = append(colleges, college)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating colleges: %w", err)
	}

	return colleges, nil
}

// GetCollegeByID retrieves a college by its ID
func (s *CollegeStore) GetCollegeByID(ctx context.Context, collegeID string) (*College, error) {
	query := `SELECT id, name, state_id, city FROM colleges WHERE id = $1`
	var college College
	var city sql.NullString
	err := s.postgres.DB.QueryRowContext(ctx, query, collegeID).Scan(&college.ID, &college.Name, &college.StateID, &city)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("college not found")
		}
		return nil, fmt.Errorf("failed to get college: %w", err)
	}
	if city.Valid {
		college.City = city.String
	}
	return &college, nil
}

// CreateCollegeRequest represents the request to create a new college
type CreateCollegeRequest struct {
	Name    string `json:"name"`
	StateID string `json:"state_id"`
	City    string `json:"city,omitempty"`
}

// CreateCollege creates a new college in the database
func (s *CollegeStore) CreateCollege(ctx context.Context, req CreateCollegeRequest) (*College, error) {
	query := `INSERT INTO colleges (name, state_id, city) VALUES ($1, $2, $3) RETURNING id, name, state_id, city`
	var college College
	var city sql.NullString
	err := s.postgres.DB.QueryRowContext(ctx, query, req.Name, req.StateID, req.City).Scan(&college.ID, &college.Name, &college.StateID, &city)
	if err != nil {
		return nil, fmt.Errorf("failed to create college: %w", err)
	}
	if city.Valid {
		college.City = city.String
	}
	return &college, nil
}
