package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/rohit21755/groveserverv2/internal/db"
)

type User struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone,omitempty"`
	StateID         string    `json:"state_id"`
	CollegeID       string    `json:"college_id"`
	Role            string    `json:"role"`
	XP              int       `json:"xp"`
	Level           int       `json:"level"`
	Coins           int       `json:"coins"`
	Bio             string    `json:"bio,omitempty"`
	AvatarURL       string    `json:"avatar_url,omitempty"`
	ResumeURL       string    `json:"resume_url,omitempty"`
	ResumeVisibility string   `json:"resume_visibility"`
	ReferralCode    string    `json:"referral_code"`
	ReferredByID    string    `json:"referred_by_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type UserStore struct {
	postgres *db.Postgres
}

func NewUserStore(postgres *db.Postgres) *UserStore {
	return &UserStore{
		postgres: postgres,
	}
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	StateID      string `json:"state_id"`
	CollegeID    string `json:"college_id"`
	ReferralCode string `json:"referral_code,omitempty"` // Optional: code of the user who referred them
}

// Register creates a new user account
func (s *UserStore) Register(ctx context.Context, req RegisterRequest, resumeURL, profilePicURL string) (*User, error) {
	// Start transaction
	tx, err := s.postgres.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate unique referral code for the new user
	// Keep generating until we get a unique one
	referralCode, err := s.generateUniqueReferralCode(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique referral code: %w", err)
	}

	// Check if referral code is provided and valid
	var referrerID sql.NullString
	if req.ReferralCode != "" {
		referrerIDStr, err := s.getUserIDByReferralCode(ctx, tx, req.ReferralCode)
		if err != nil {
			return nil, fmt.Errorf("invalid referral code: %w", err)
		}
		if referrerIDStr != "" {
			referrerID = sql.NullString{String: referrerIDStr, Valid: true}
		}
	}

	// Insert user
	userID := uuid.New().String()
	query := `
		INSERT INTO users (
			id, name, email, password_hash, state_id, college_id,
			avatar_url, resume_url, referral_code, referred_by_id, role
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, name, email, phone, state_id, college_id, role, xp, level, coins,
		          bio, avatar_url, resume_url, resume_visibility, referral_code, 
		          referred_by_id, created_at
	`

	var user User
	var phone, bio sql.NullString
	var referredByID sql.NullString

	err = tx.QueryRowContext(ctx, query,
		userID, req.Name, req.Email, hashedPassword, req.StateID, req.CollegeID,
		profilePicURL, resumeURL, referralCode, referrerID, "student",
	).Scan(
		&user.ID, &user.Name, &user.Email, &phone, &user.StateID, &user.CollegeID,
		&user.Role, &user.XP, &user.Level, &user.Coins,
		&bio, &user.AvatarURL, &user.ResumeURL, &user.ResumeVisibility, &user.ReferralCode,
		&referredByID, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if phone.Valid {
		user.Phone = phone.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if referredByID.Valid {
		user.ReferredByID = referredByID.String
	}

	// If user was referred, create referral record
	if referrerID.Valid {
		referralQuery := `
			INSERT INTO user_referrals (referrer_id, referred_id, referral_code)
			VALUES ($1, $2, $3)
		`
		_, err = tx.ExecContext(ctx, referralQuery, referrerID.String, userID, req.ReferralCode)
		if err != nil {
			return nil, fmt.Errorf("failed to create referral record: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, nil
}

// getUserIDByReferralCode gets user ID by referral code
func (s *UserStore) getUserIDByReferralCode(ctx context.Context, tx *sql.Tx, referralCode string) (string, error) {
	var userID string
	query := `SELECT id FROM users WHERE referral_code = $1`
	err := tx.QueryRowContext(ctx, query, referralCode).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("referral code not found")
		}
		return "", fmt.Errorf("failed to get user by referral code: %w", err)
	}
	return userID, nil
}

// GetUserByEmail retrieves a user by email (without password hash)
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, name, email, phone, state_id, college_id, role, xp, level, coins,
		       bio, avatar_url, resume_url, resume_visibility, referral_code,
		       referred_by_id, created_at
		FROM users WHERE email = $1
	`
	var user User
	var phone, bio sql.NullString
	var referredByID sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &phone, &user.StateID, &user.CollegeID,
		&user.Role, &user.XP, &user.Level, &user.Coins,
		&bio, &user.AvatarURL, &user.ResumeURL, &user.ResumeVisibility, &user.ReferralCode,
		&referredByID, &user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if phone.Valid {
		user.Phone = phone.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if referredByID.Valid {
		user.ReferredByID = referredByID.String
	}

	return &user, nil
}

// GetUserPasswordHash retrieves password hash for a user (for login verification)
func (s *UserStore) GetUserPasswordHash(ctx context.Context, email string) (string, error) {
	query := `SELECT password_hash FROM users WHERE email = $1`
	var passwordHash string
	err := s.postgres.DB.QueryRowContext(ctx, query, email).Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		return "", fmt.Errorf("failed to get password hash: %w", err)
	}
	return passwordHash, nil
}

// VerifyPassword verifies a password against the stored hash
func (s *UserStore) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// generateUniqueReferralCode generates a unique referral code that doesn't exist in the database
func (s *UserStore) generateUniqueReferralCode(ctx context.Context, tx *sql.Tx) (string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		code := generateReferralCode()
		
		// Check if code already exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE referral_code = $1)`
		err := tx.QueryRowContext(ctx, checkQuery, code).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("failed to check referral code uniqueness: %w", err)
		}
		
		if !exists {
			return code, nil
		}
	}
	
	return "", fmt.Errorf("failed to generate unique referral code after %d attempts", maxAttempts)
}

// generateReferralCode generates a referral code (8 characters, alphanumeric uppercase)
func generateReferralCode() string {
	// Generate 6 random bytes
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to UUID-based code if random generation fails
		return generateUUIDBasedCode()
	}
	
	// Encode to base64 and take first 8 characters, make uppercase
	code := base64.URLEncoding.EncodeToString(bytes)
	if len(code) < 8 {
		return generateUUIDBasedCode()
	}
	code = code[:8]
	
	// Remove any special characters and make uppercase
	result := ""
	for _, char := range code {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			if char >= 'a' && char <= 'z' {
				result += string(char - 32) // Convert to uppercase
			} else {
				result += string(char)
			}
		}
	}
	
	// Ensure we have at least 8 characters
	if len(result) < 8 {
		return generateUUIDBasedCode()
	}
	
	return result[:8]
}

// generateUUIDBasedCode generates a referral code from UUID
func generateUUIDBasedCode() string {
	uuidCode := uuid.New().String()
	result := ""
	for _, char := range uuidCode {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			result += string(char)
			if len(result) >= 8 {
				break
			}
		}
	}
	// If still not enough, pad with numbers
	for len(result) < 8 {
		result += "0"
	}
	return result[:8]
}

// GetUserByID retrieves a user by ID
func (s *UserStore) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT id, name, email, phone, state_id, college_id, role, xp, level, coins,
		       bio, avatar_url, resume_url, resume_visibility, referral_code,
		       referred_by_id, created_at
		FROM users WHERE id = $1
	`
	var user User
	var phone, bio sql.NullString
	var referredByID sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Name, &user.Email, &phone, &user.StateID, &user.CollegeID,
		&user.Role, &user.XP, &user.Level, &user.Coins,
		&bio, &user.AvatarURL, &user.ResumeURL, &user.ResumeVisibility, &user.ReferralCode,
		&referredByID, &user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if phone.Valid {
		user.Phone = phone.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if referredByID.Valid {
		user.ReferredByID = referredByID.String
	}

	return &user, nil
}
