package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type Admin struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminStore struct {
	postgres *db.Postgres
}

func NewAdminStore(postgres *db.Postgres) *AdminStore {
	return &AdminStore{
		postgres: postgres,
	}
}

// CreateAdminRequest represents the request to create an admin
type CreateAdminRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateAdmin creates a new admin user
func (s *AdminStore) CreateAdmin(ctx context.Context, req CreateAdminRequest) (*Admin, error) {
	// Validate required fields
	if req.Name == "" || req.Username == "" || req.Password == "" {
		return nil, fmt.Errorf("name, username, and password are required")
	}

	// Check if username already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM admins WHERE username = $1)`
	err := s.postgres.DB.QueryRowContext(ctx, checkQuery, req.Username).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing admin: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin
	adminID := uuid.New().String()
	query := `
		INSERT INTO admins (id, name, username, password_hash, role)
		VALUES ($1, $2, $3, $4, 'admin')
		RETURNING id, name, username, role, created_at, updated_at
	`

	var admin Admin
	err = s.postgres.DB.QueryRowContext(ctx, query,
		adminID, req.Name, req.Username, string(hashedPassword),
	).Scan(
		&admin.ID, &admin.Name, &admin.Username, &admin.Role, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	return &admin, nil
}

// GetAdminByID retrieves an admin by ID
func (s *AdminStore) GetAdminByID(ctx context.Context, adminID string) (*Admin, error) {
	query := `
		SELECT id, name, username, role, created_at, updated_at
		FROM admins WHERE id = $1
	`

	var admin Admin
	err := s.postgres.DB.QueryRowContext(ctx, query, adminID).Scan(
		&admin.ID, &admin.Name, &admin.Username, &admin.Role, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

// GetAdminByUsername retrieves an admin by username
func (s *AdminStore) GetAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	query := `
		SELECT id, name, username, role, created_at, updated_at
		FROM admins WHERE username = $1
	`

	var admin Admin
	err := s.postgres.DB.QueryRowContext(ctx, query, username).Scan(
		&admin.ID, &admin.Name, &admin.Username, &admin.Role, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

// VerifyAdminPassword verifies an admin's password
func (s *AdminStore) VerifyAdminPassword(ctx context.Context, username, password string) (bool, error) {
	query := `SELECT password_hash FROM admins WHERE username = $1`
	var passwordHash string
	err := s.postgres.DB.QueryRowContext(ctx, query, username).Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to get admin password: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	return err == nil, nil
}
