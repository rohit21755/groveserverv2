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
	StateName       string    `json:"state_name,omitempty"`
	CollegeID       string    `json:"college_id"`
	CollegeName     string    `json:"college_name,omitempty"`
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

	// Fetch user with state and college names
	userWithNames, err := s.GetUserByID(ctx, userID)
	if err != nil {
		// If fetch fails, return the user without names (fallback)
		return &user, nil
	}

	return userWithNames, nil
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

// GetUserByEmail retrieves a user by email (without password hash) with state and college names
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT 
			u.id, u.name, u.email, u.phone, u.state_id, u.college_id, u.role, u.xp, u.level, u.coins,
			u.bio, u.avatar_url, u.resume_url, u.resume_visibility, u.referral_code,
			u.referred_by_id, u.created_at,
			COALESCE(s.name, '') as state_name,
			COALESCE(c.name, '') as college_name
		FROM users u
		LEFT JOIN states s ON u.state_id = s.id
		LEFT JOIN colleges c ON u.college_id = c.id
		WHERE u.email = $1
	`
	var user User
	var phone, bio sql.NullString
	var referredByID sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &phone, &user.StateID, &user.CollegeID,
		&user.Role, &user.XP, &user.Level, &user.Coins,
		&bio, &user.AvatarURL, &user.ResumeURL, &user.ResumeVisibility, &user.ReferralCode,
		&referredByID, &user.CreatedAt,
		&user.StateName, &user.CollegeName,
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

// UpdateResumeURL updates the resume URL for a user
func (s *UserStore) UpdateResumeURL(ctx context.Context, userID, resumeURL string) error {
	query := `UPDATE users SET resume_url = $1 WHERE id = $2`
	_, err := s.postgres.DB.ExecContext(ctx, query, resumeURL, userID)
	if err != nil {
		return fmt.Errorf("failed to update resume URL: %w", err)
	}
	return nil
}

// UpdateProfilePicURL updates the profile picture URL for a user
func (s *UserStore) UpdateProfilePicURL(ctx context.Context, userID, profilePicURL string) error {
	query := `UPDATE users SET avatar_url = $1 WHERE id = $2`
	_, err := s.postgres.DB.ExecContext(ctx, query, profilePicURL, userID)
	if err != nil {
		return fmt.Errorf("failed to update profile picture URL: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID with state and college names
func (s *UserStore) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT 
			u.id, u.name, u.email, u.phone, u.state_id, u.college_id, u.role, u.xp, u.level, u.coins,
			u.bio, u.avatar_url, u.resume_url, u.resume_visibility, u.referral_code,
			u.referred_by_id, u.created_at,
			COALESCE(s.name, '') as state_name,
			COALESCE(c.name, '') as college_name
		FROM users u
		LEFT JOIN states s ON u.state_id = s.id
		LEFT JOIN colleges c ON u.college_id = c.id
		WHERE u.id = $1
	`
	var user User
	var phone, bio sql.NullString
	var referredByID sql.NullString

	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Name, &user.Email, &phone, &user.StateID, &user.CollegeID,
		&user.Role, &user.XP, &user.Level, &user.Coins,
		&bio, &user.AvatarURL, &user.ResumeURL, &user.ResumeVisibility, &user.ReferralCode,
		&referredByID, &user.CreatedAt,
		&user.StateName, &user.CollegeName,
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

// FollowUser creates a follow relationship between two users
func (s *UserStore) FollowUser(ctx context.Context, followerID, followingID string) error {
	// Check if trying to follow self
	if followerID == followingID {
		return fmt.Errorf("cannot follow yourself")
	}

	// Check if user exists
	_, err := s.GetUserByID(ctx, followingID)
	if err != nil {
		return fmt.Errorf("user to follow not found: %w", err)
	}

	// Check if already following
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $1 AND following_id = $2)`
	err = s.postgres.DB.QueryRowContext(ctx, checkQuery, followerID, followingID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check follow relationship: %w", err)
	}

	if exists {
		return fmt.Errorf("already following this user")
	}

	// Create follow relationship
	query := `INSERT INTO user_follows (follower_id, following_id) VALUES ($1, $2)`
	_, err = s.postgres.DB.ExecContext(ctx, query, followerID, followingID)
	if err != nil {
		return fmt.Errorf("failed to create follow relationship: %w", err)
	}

	return nil
}

// UnfollowUser removes a follow relationship between two users
func (s *UserStore) UnfollowUser(ctx context.Context, followerID, followingID string) error {
	// Check if trying to unfollow self
	if followerID == followingID {
		return fmt.Errorf("cannot unfollow yourself")
	}

	// Remove follow relationship
	query := `DELETE FROM user_follows WHERE follower_id = $1 AND following_id = $2`
	result, err := s.postgres.DB.ExecContext(ctx, query, followerID, followingID)
	if err != nil {
		return fmt.Errorf("failed to remove follow relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("not following this user")
	}

	return nil
}

// GetFollowingCount returns the number of users that the given user is following
func (s *UserStore) GetFollowingCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM user_follows WHERE follower_id = $1`
	var count int
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get following count: %w", err)
	}
	return count, nil
}

// GetFollowersCount returns the number of users following the given user
func (s *UserStore) GetFollowersCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM user_follows WHERE following_id = $1`
	var count int
	err := s.postgres.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get followers count: %w", err)
	}
	return count, nil
}
