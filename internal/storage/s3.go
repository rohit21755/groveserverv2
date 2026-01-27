package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/google/uuid"
)

type S3Storage struct {
	client             *s3.Client
	uploader           *manager.Uploader
	profileBucket      string
	resumeBucket       string
	taskProofBucket    string
	region             string
	profilePublicURL   string
	resumePublicURL    string
	taskProofPublicURL string
}

type S3Config struct {
	Region             string
	ProfileBucket      string
	ResumeBucket       string
	TaskProofBucket    string
	AccessKeyID        string
	SecretAccessKey    string
	ProfilePublicURL   string // Optional: CDN URL or S3 public URL for profile bucket
	ResumePublicURL    string // Optional: CDN URL or S3 public URL for resume bucket
	TaskProofPublicURL string // Optional: CDN URL or S3 public URL for task proof bucket
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	log.Printf("[S3] Initializing S3 storage - Region: %s, Profile Bucket: %s, Resume Bucket: %s, Task Proof Bucket: %s", cfg.Region, cfg.ProfileBucket, cfg.ResumeBucket, cfg.TaskProofBucket)

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		log.Printf("[S3] ERROR: Failed to load AWS config: %v", err)
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	uploader := manager.NewUploader(client)

	// Set default public URLs if not provided
	profilePublicURL := cfg.ProfilePublicURL
	if profilePublicURL == "" {
		profilePublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.ProfileBucket, cfg.Region)
		log.Printf("[S3] Using default profile public URL: %s", profilePublicURL)
	} else {
		log.Printf("[S3] Using custom profile public URL: %s", profilePublicURL)
	}

	resumePublicURL := cfg.ResumePublicURL
	if resumePublicURL == "" {
		resumePublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.ResumeBucket, cfg.Region)
		log.Printf("[S3] Using default resume public URL: %s", resumePublicURL)
	} else {
		log.Printf("[S3] Using custom resume public URL: %s", resumePublicURL)
	}

	taskProofPublicURL := cfg.TaskProofPublicURL
	if taskProofPublicURL == "" {
		taskProofPublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.TaskProofBucket, cfg.Region)
		log.Printf("[S3] Using default task proof public URL: %s", taskProofPublicURL)
	} else {
		log.Printf("[S3] Using custom task proof public URL: %s", taskProofPublicURL)
	}

	log.Printf("[S3] S3 storage initialized successfully")
	return &S3Storage{
		client:             client,
		uploader:           uploader,
		profileBucket:      cfg.ProfileBucket,
		resumeBucket:       cfg.ResumeBucket,
		taskProofBucket:    cfg.TaskProofBucket,
		region:             cfg.Region,
		profilePublicURL:   profilePublicURL,
		resumePublicURL:    resumePublicURL,
		taskProofPublicURL: taskProofPublicURL,
	}, nil
}

// GetProfileBucket returns the profile bucket name
func (s *S3Storage) GetProfileBucket() string {
	return s.profileBucket
}

// GetResumeBucket returns the resume bucket name
func (s *S3Storage) GetResumeBucket() string {
	return s.resumeBucket
}

// GetTaskProofBucket returns the task proof bucket name
func (s *S3Storage) GetTaskProofBucket() string {
	return s.taskProofBucket
}

// GetTaskProofPublicURL returns the task proof public URL
func (s *S3Storage) GetTaskProofPublicURL() string {
	return s.taskProofPublicURL
}

// UploadFile uploads a file to S3 and returns the public URL
func (s *S3Storage) UploadFile(
	ctx context.Context,
	file io.Reader,
	bucket string,
	key string,
	contentType string,
	publicURL string,
	forceDownload bool,
) (string, error) {

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	if forceDownload {
		filename := filepath.Base(key)
		input.ContentDisposition = aws.String(
			fmt.Sprintf("attachment; filename=\"%s\"", filename),
		)
	}

	start := time.Now()
	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Ensure publicURL is not empty - construct default if needed
	if publicURL == "" {
		// Construct default S3 public URL
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, s.region)
		log.Printf("[S3] Warning: publicURL was empty, using default: %s", publicURL)
	}

	// Construct full URL
	url := fmt.Sprintf("%s/%s", publicURL, key)
	
	// Ensure URL doesn't have double slashes
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.Replace(url, "https:/", "https://", 1)
	url = strings.Replace(url, "http:/", "http://", 1)

	log.Printf(
		"[S3] Upload successful - Bucket=%s Key=%s ETag=%s Duration=%v",
		bucket,
		key,
		aws.ToString(result.ETag),
		time.Since(start),
	)

	return url, nil
}

// UploadResume uploads a resume file to S3 resume bucket
func (s *S3Storage) UploadResume(ctx context.Context, file io.Reader, userID string, filename string) (string, error) {
	log.Printf("[S3] UploadResume - UserID: %s, OriginalFilename: %s", userID, filename)

	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".pdf" // Default to PDF
		log.Printf("[S3] No extension found, defaulting to .pdf")
	}
	newFilename := fmt.Sprintf("%s_resume%s", userID, ext)
	key := fmt.Sprintf("resumes/%s", newFilename)

	// Set content type based on extension
	contentType := "application/pdf"
	if ext == ".doc" || ext == ".docx" {
		contentType = "application/msword"
	} else if ext == ".docx" {
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	log.Printf("[S3] Resume upload - Key: %s, ContentType: %s", key, contentType)

	// Force download for resumes
	url, err := s.UploadFile(ctx, file, s.resumeBucket, key, contentType, s.resumePublicURL, true)
	if err != nil {
		log.Printf("[S3] ERROR: Resume upload failed - UserID: %s, Key: %s, Error: %v", userID, key, err)
		return "", err
	}

	log.Printf("[S3] Resume upload completed - UserID: %s, URL: %s", userID, url)
	return url, nil
}

// UploadProfilePic uploads a profile picture to S3 profile bucket
func (s *S3Storage) UploadProfilePic(ctx context.Context, file io.Reader, userID string, filename string) (string, error) {
	log.Printf("[S3] UploadProfilePic - UserID: %s, OriginalFilename: %s", userID, filename)

	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg" // Default to JPG
		log.Printf("[S3] No extension found, defaulting to .jpg")
	}
	newFilename := fmt.Sprintf("%s_profile%s", userID, ext)
	key := fmt.Sprintf("profile-pics/%s", newFilename)

	// Determine content type based on extension
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	log.Printf("[S3] Profile pic upload - Key: %s, ContentType: %s", key, contentType)

	// Don't force download for profile pictures (display in browser)
	url, err := s.UploadFile(ctx, file, s.profileBucket, key, contentType, s.profilePublicURL, false)
	if err != nil {
		log.Printf("[S3] ERROR: Profile pic upload failed - UserID: %s, Key: %s, Error: %v", userID, key, err)
		return "", err
	}

	log.Printf("[S3] Profile pic upload completed - UserID: %s, URL: %s", userID, url)
	return url, nil
}

// DeleteResume deletes a resume file from S3
func (s *S3Storage) DeleteResume(ctx context.Context, key string) error {
	log.Printf("[S3] Deleting resume - Bucket: %s, Key: %s", s.resumeBucket, key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.resumeBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("[S3] ERROR: Failed to delete resume - Bucket: %s, Key: %s, Error: %v", s.resumeBucket, key, err)
		return err
	}
	log.Printf("[S3] Resume deleted successfully - Bucket: %s, Key: %s", s.resumeBucket, key)
	return nil
}

// DeleteProfilePic deletes a profile picture from S3
func (s *S3Storage) DeleteProfilePic(ctx context.Context, key string) error {
	log.Printf("[S3] Deleting profile pic - Bucket: %s, Key: %s", s.profileBucket, key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.profileBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("[S3] ERROR: Failed to delete profile pic - Bucket: %s, Key: %s, Error: %v", s.profileBucket, key, err)
		return err
	}
	log.Printf("[S3] Profile pic deleted successfully - Bucket: %s, Key: %s", s.profileBucket, key)
	return nil
}

// GeneratePresignedResumeURL generates a presigned URL for resume download
func (s *S3Storage) GeneratePresignedResumeURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	log.Printf("[S3] Generating presigned resume URL - Bucket: %s, Key: %s, Duration: %v", s.resumeBucket, key, duration)
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.resumeBucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String("attachment"), // Force download
	}, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		log.Printf("[S3] ERROR: Failed to generate presigned resume URL - Key: %s, Error: %v", key, err)
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	log.Printf("[S3] Presigned resume URL generated - Key: %s, Expires: %v", key, duration)
	return request.URL, nil
}

// GeneratePresignedProfileURL generates a presigned URL for profile picture
func (s *S3Storage) GeneratePresignedProfileURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	log.Printf("[S3] Generating presigned profile URL - Bucket: %s, Key: %s, Duration: %v", s.profileBucket, key, duration)
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.profileBucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		log.Printf("[S3] ERROR: Failed to generate presigned profile URL - Key: %s, Error: %v", key, err)
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	log.Printf("[S3] Presigned profile URL generated - Key: %s, Expires: %v", key, duration)
	return request.URL, nil
}
