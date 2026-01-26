package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type State struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type StateStore struct {
	postgres *db.Postgres
}

func NewStateStore(postgres *db.Postgres) *StateStore {
	return &StateStore{
		postgres: postgres,
	}
}

// GetAllStates retrieves all states from the database
func (s *StateStore) GetAllStates(ctx context.Context) ([]State, error) {
	query := `SELECT id, name, code FROM states ORDER BY name ASC`
	rows, err := s.postgres.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	defer rows.Close()

	var states []State
	for rows.Next() {
		var state State
		if err := rows.Scan(&state.ID, &state.Name, &state.Code); err != nil {
			return nil, fmt.Errorf("failed to scan state: %w", err)
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating states: %w", err)
	}

	return states, nil
}

// GetStateByID retrieves a state by its ID
func (s *StateStore) GetStateByID(ctx context.Context, stateID string) (*State, error) {
	query := `SELECT id, name, code FROM states WHERE id = $1`
	var state State
	err := s.postgres.DB.QueryRowContext(ctx, query, stateID).Scan(&state.ID, &state.Name, &state.Code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state not found")
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	return &state, nil
}

// CreateStateRequest represents the request to create a new state
type CreateStateRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// CreateState creates a new state in the database
func (s *StateStore) CreateState(ctx context.Context, req CreateStateRequest) (*State, error) {
	query := `INSERT INTO states (name, code) VALUES ($1, $2) RETURNING id, name, code`
	var state State
	err := s.postgres.DB.QueryRowContext(ctx, query, req.Name, req.Code).Scan(&state.ID, &state.Name, &state.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}
	return &state, nil
}
