package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/rohit21755/groveserverv2/internal/db"
)

type Submission struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	UserID      string     `json:"user_id"`
	ProofURL    string     `json:"proof_url"`
	Status      string     `json:"status"`
	AdminComment string    `json:"admin_comment,omitempty"`
	ReviewedBy  string     `json:"reviewed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SubmissionStore struct {
	postgres *db.Postgres
}

func NewSubmissionStore(postgres *db.Postgres) *SubmissionStore {
	return &SubmissionStore{
		postgres: postgres,
	}
}

// CreateSubmissionRequest represents the request to create a submission
type CreateSubmissionRequest struct {
	TaskID   string `json:"task_id"`
	UserID   string `json:"user_id"`
	ProofURL string `json:"proof_url"`
}

// GetSubmissionByTaskAndUser retrieves a submission by task ID and user ID
func (s *SubmissionStore) GetSubmissionByTaskAndUser(ctx context.Context, taskID, userID string) (*Submission, error) {
	query := `
		SELECT id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
		FROM submissions WHERE task_id = $1 AND user_id = $2
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, taskID, userID).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	return &submission, nil
}

// UpdateSubmissionProof updates the proof URL for an existing submission (for resubmission)
func (s *SubmissionStore) UpdateSubmissionProof(ctx context.Context, submissionID, newProofURL string) (*Submission, error) {
	query := `
		UPDATE submissions
		SET proof_url = $1,
		    status = 'pending',
		    admin_comment = NULL,
		    reviewed_by = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, newProofURL, submissionID).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to update submission proof: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	return &submission, nil
}

// CreateSubmission creates a new task submission
// If a submission already exists and is rejected, it will be updated instead of creating a new one
func (s *SubmissionStore) CreateSubmission(ctx context.Context, req CreateSubmissionRequest) (*Submission, error) {
	// Check if submission already exists
	existingSubmission, err := s.GetSubmissionByTaskAndUser(ctx, req.TaskID, req.UserID)
	if err != nil && err.Error() != "submission not found" {
		return nil, fmt.Errorf("failed to check existing submission: %w", err)
	}

	// If submission exists and is rejected, update it (resubmission)
	if existingSubmission != nil {
		if existingSubmission.Status == "rejected" {
			// Allow resubmission by updating the existing rejected submission
			return s.UpdateSubmissionProof(ctx, existingSubmission.ID, req.ProofURL)
		}
		// If submission exists and is not rejected (pending or approved), return error
		return nil, fmt.Errorf("submission already exists for this task with status: %s", existingSubmission.Status)
	}

	// Create submission
	submissionID := uuid.New().String()
	query := `
		INSERT INTO submissions (id, task_id, user_id, proof_url, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err = s.postgres.DB.QueryRowContext(ctx, query,
		submissionID, req.TaskID, req.UserID, req.ProofURL,
	).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	return &submission, nil
}

// GetSubmissionByID retrieves a submission by ID
func (s *SubmissionStore) GetSubmissionByID(ctx context.Context, submissionID string) (*Submission, error) {
	query := `
		SELECT id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
		FROM submissions WHERE id = $1
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, submissionID).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	return &submission, nil
}

// ApproveSubmission approves a submission
func (s *SubmissionStore) ApproveSubmission(ctx context.Context, submissionID, adminUserID string, comment string) (*Submission, error) {
	query := `
		UPDATE submissions
		SET status = 'approved',
		    reviewed_by = $1,
		    admin_comment = CASE WHEN $2 != '' THEN $2 ELSE admin_comment END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, adminUserID, comment, submissionID).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to approve submission: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	return &submission, nil
}

// RejectSubmission rejects a submission
func (s *SubmissionStore) RejectSubmission(ctx context.Context, submissionID, adminUserID, comment string) (*Submission, error) {
	if comment == "" {
		return nil, fmt.Errorf("rejection comment is required")
	}

	// Log rejection details for debugging
	log.Printf("[Submission] Rejecting submission - ID: %s, Admin: %s, Comment: %s", submissionID, adminUserID, comment)

	query := `
		UPDATE submissions
		SET status = 'rejected',
		    reviewed_by = $1,
		    admin_comment = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
	`

	var submission Submission
	var adminComment, reviewedBy sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, adminUserID, comment, submissionID).Scan(
		&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
		&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("failed to reject submission: %w", err)
	}

	if adminComment.Valid {
		submission.AdminComment = adminComment.String
	}
	if reviewedBy.Valid {
		submission.ReviewedBy = reviewedBy.String
	}

	// Verify the rejection was applied correctly
	if submission.Status != "rejected" {
		log.Printf("[Submission] ERROR: Status mismatch after rejection - Expected: rejected, Got: %s", submission.Status)
		return nil, fmt.Errorf("failed to reject submission: status was not set to rejected")
	}
	if submission.AdminComment != comment {
		log.Printf("[Submission] ERROR: Comment mismatch after rejection - Expected: %s, Got: %s", comment, submission.AdminComment)
	}

	log.Printf("[Submission] Successfully rejected submission - ID: %s, Status: %s, Comment: %s", submission.ID, submission.Status, submission.AdminComment)

	return &submission, nil
}

// GetAllSubmissions retrieves all submissions with optional filters
func (s *SubmissionStore) GetAllSubmissions(ctx context.Context, statusFilter string) ([]Submission, error) {
	var query string
	var args []interface{}

	if statusFilter != "" {
		query = `
			SELECT id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
			FROM submissions
			WHERE status = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{statusFilter}
	} else {
		query = `
			SELECT id, task_id, user_id, proof_url, status, admin_comment, reviewed_by, created_at, updated_at
			FROM submissions
			ORDER BY created_at DESC
		`
		args = []interface{}{}
	}

	rows, err := s.postgres.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var submission Submission
		var adminComment, reviewedBy sql.NullString

		err := rows.Scan(
			&submission.ID, &submission.TaskID, &submission.UserID, &submission.ProofURL, &submission.Status,
			&adminComment, &reviewedBy, &submission.CreatedAt, &submission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		if adminComment.Valid {
			submission.AdminComment = adminComment.String
		}
		if reviewedBy.Valid {
			submission.ReviewedBy = reviewedBy.String
		}

		submissions = append(submissions, submission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submission rows: %w", err)
	}

	return submissions, nil
}
