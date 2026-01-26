package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rohit21755/groveserverv2/internal/db"
)

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	XP          int        `json:"xp"`
	Type        string     `json:"type"`
	ProofType   string     `json:"proof_type"`
	Priority    string     `json:"priority"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	IsFlash     bool       `json:"is_flash"`
	IsWeekly    bool       `json:"is_weekly"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type TaskStore struct {
	postgres *db.Postgres
}

func NewTaskStore(postgres *db.Postgres) *TaskStore {
	return &TaskStore{
		postgres: postgres,
	}
}

// CreateTaskRequest represents the request to create a task
type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	XP          int        `json:"xp"`
	Type        string     `json:"type"`
	ProofType   string     `json:"proof_type"`
	Priority    string     `json:"priority"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	IsFlash     bool       `json:"is_flash"`
	IsWeekly    bool       `json:"is_weekly"`
	CreatedBy   string     `json:"created_by"`
}

// AssignmentType represents how the task should be assigned
type AssignmentType string

const (
	AssignmentAll     AssignmentType = "all"     // All users
	AssignmentState   AssignmentType = "state"   // Users from a specific state
	AssignmentCollege AssignmentType = "college"  // Users from a specific college
	AssignmentUser    AssignmentType = "user"   // Single user
)

// CreateTask creates a new task and assigns it to users based on assignment type
func (s *TaskStore) CreateTask(ctx context.Context, req CreateTaskRequest, assignmentType AssignmentType, assignmentID string) (*Task, []string, error) {
	// Start transaction
	tx, err := s.postgres.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create task
	taskID := uuid.New().String()
	query := `
		INSERT INTO tasks (id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
	`

	var task Task
	var startAt, endAt sql.NullTime

	err = tx.QueryRowContext(ctx, query,
		taskID, req.Title, req.Description, req.XP, req.Type, req.ProofType, req.Priority,
		req.StartAt, req.EndAt, req.IsFlash, req.IsWeekly, req.CreatedBy,
	).Scan(
		&task.ID, &task.Title, &task.Description, &task.XP, &task.Type, &task.ProofType, &task.Priority,
		&startAt, &endAt, &task.IsFlash, &task.IsWeekly, &task.CreatedBy, &task.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create task: %w", err)
	}

	if startAt.Valid {
		task.StartAt = &startAt.Time
	}
	if endAt.Valid {
		task.EndAt = &endAt.Time
	}

	// Get user IDs based on assignment type
	userIDs, err := s.getUserIDsForAssignment(ctx, tx, assignmentType, assignmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user IDs for assignment: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &task, userIDs, nil
}

// getUserIDsForAssignment gets user IDs based on assignment type
func (s *TaskStore) getUserIDsForAssignment(ctx context.Context, tx *sql.Tx, assignmentType AssignmentType, assignmentID string) ([]string, error) {
	var query string
	var args []interface{}

	switch assignmentType {
	case AssignmentAll:
		// Get all users
		query = `SELECT id FROM users WHERE role = 'student'`
		args = []interface{}{}

	case AssignmentState:
		// Get users from a specific state
		query = `SELECT id FROM users WHERE state_id = $1 AND role = 'student'`
		args = []interface{}{assignmentID}

	case AssignmentCollege:
		// Get users from a specific college
		query = `SELECT id FROM users WHERE college_id = $1 AND role = 'student'`
		args = []interface{}{assignmentID}

	case AssignmentUser:
		// Get single user
		query = `SELECT id FROM users WHERE id = $1 AND role = 'student'`
		args = []interface{}{assignmentID}

	default:
		return nil, fmt.Errorf("invalid assignment type: %s", assignmentType)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return userIDs, nil
}

// GetTaskByID retrieves a task by ID
func (s *TaskStore) GetTaskByID(ctx context.Context, taskID string) (*Task, error) {
	query := `
		SELECT id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
		FROM tasks WHERE id = $1
	`

	var task Task
	var startAt, endAt sql.NullTime

	err := s.postgres.DB.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID, &task.Title, &task.Description, &task.XP, &task.Type, &task.ProofType, &task.Priority,
		&startAt, &endAt, &task.IsFlash, &task.IsWeekly, &task.CreatedBy, &task.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if startAt.Valid {
		task.StartAt = &startAt.Time
	}
	if endAt.Valid {
		task.EndAt = &endAt.Time
	}

	return &task, nil
}

// GetTasksForUser retrieves all tasks assigned to a user
// Tasks are assigned based on: all users, user's state, user's college, or specific user assignment
func (s *TaskStore) GetTasksForUser(ctx context.Context, userID string) ([]Task, error) {
	// First, get user's state_id and college_id
	var stateID, collegeID sql.NullString
	userQuery := `SELECT state_id, college_id FROM users WHERE id = $1`
	err := s.postgres.DB.QueryRowContext(ctx, userQuery, userID).Scan(&stateID, &collegeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Query tasks that are assigned to:
	// 1. All users (we need to check this by querying all tasks and filtering)
	// 2. User's state (if state_id matches)
	// 3. User's college (if college_id matches)
	// 4. Specific user (if user_id matches)
	
	// For now, we'll get all active tasks and filter in application logic
	// In a production system, you might want a task_assignments table
	// For simplicity, we'll return all tasks that are:
	// - Not expired (end_at is null or in the future)
	// - Started (start_at is null or in the past)
	query := `
		SELECT id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by, created_at
		FROM tasks
		WHERE (start_at IS NULL OR start_at <= NOW())
		AND (end_at IS NULL OR end_at >= NOW())
		ORDER BY created_at DESC
	`

	rows, err := s.postgres.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var startAt, endAt sql.NullTime

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.XP, &task.Type, &task.ProofType, &task.Priority,
			&startAt, &endAt, &task.IsFlash, &task.IsWeekly, &task.CreatedBy, &task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if startAt.Valid {
			task.StartAt = &startAt.Time
		}
		if endAt.Valid {
			task.EndAt = &endAt.Time
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task rows: %w", err)
	}

	return tasks, nil
}

// CheckSubmissionExists checks if user has already submitted a task
func (s *TaskStore) CheckSubmissionExists(ctx context.Context, taskID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM submissions WHERE task_id = $1 AND user_id = $2)`
	var exists bool
	err := s.postgres.DB.QueryRowContext(ctx, query, taskID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check submission: %w", err)
	}
	return exists, nil
}
