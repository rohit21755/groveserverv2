package env

import (
	"os"
)

type Config struct {
	// Server
	Env     string
	APIHost string
	APIPort string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret string
	JWTExpiry string

	// CORS
	CORSAllowedOrigins []string

	// AWS S3
	AWSRegion              string
	AWSProfileBucket       string
	AWSResumeBucket        string
	AWSTaskProofBucket     string // Bucket for task submission proof files
	AWSBadgeBucket         string // Bucket for badge images
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
	AWSProfilePublicURL    string // Optional: CDN URL for profile bucket
	AWSResumePublicURL     string // Optional: CDN URL for resume bucket
	AWSTaskProofPublicURL  string // Optional: CDN URL for task proof bucket
	AWSBadgePublicURL      string // Optional: CDN URL for badge bucket
}

func Load() *Config {
	return &Config{
		Env:     getEnv("ENV", "development"),
		APIHost: getEnv("API_HOST", "0.0.0.0"),
		APIPort: getEnv("API_PORT", "8080"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gamified_ambassador?sslmode=disable"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiry: getEnv("JWT_EXPIRY", "24h"),

		CORSAllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),

		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		AWSProfileBucket:       getEnv("AWS_PROFILE_BUCKET", ""),
		AWSResumeBucket:        getEnv("AWS_RESUME_BUCKET", ""),
		AWSTaskProofBucket:     getEnv("AWS_TASK_PROOF_BUCKET", "users-submissions"),
		AWSBadgeBucket:         getEnv("AWS_BADGE_BUCKET", ""), // Defaults to profile bucket if not set
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSProfilePublicURL:    getEnv("AWS_PROFILE_PUBLIC_URL", ""),
		AWSResumePublicURL:     getEnv("AWS_RESUME_PUBLIC_URL", ""),
		AWSTaskProofPublicURL:  getEnv("AWS_TASK_PROOF_PUBLIC_URL", ""),
		AWSBadgePublicURL:      getEnv("AWS_BADGE_PUBLIC_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		result := []string{}
		// Split by comma and trim spaces
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	current := ""
	for _, char := range s {
		if string(char) == sep {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else if char != ' ' {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
