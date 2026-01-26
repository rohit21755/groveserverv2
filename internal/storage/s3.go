package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	// "github.com/google/uuid"
)

type S3Storage struct {
	client           *s3.Client
	uploader         *manager.Uploader
	profileBucket    string
	resumeBucket     string
	region           string
	profilePublicURL string
	resumePublicURL  string
}

type S3Config struct {
	Region           string
	ProfileBucket    string
	ResumeBucket     string
	AccessKeyID      string
	SecretAccessKey  string
	ProfilePublicURL string // Optional: CDN URL or S3 public URL for profile bucket
	ResumePublicURL  string // Optional: CDN URL or S3 public URL for resume bucket
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	uploader := manager.NewUploader(client)

	// Set default public URLs if not provided
	profilePublicURL := cfg.ProfilePublicURL
	if profilePublicURL == "" {
		profilePublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.ProfileBucket, cfg.Region)
	}

	resumePublicURL := cfg.ResumePublicURL
	if resumePublicURL == "" {
		resumePublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.ResumeBucket, cfg.Region)
	}

	return &S3Storage{
		client:           client,
		uploader:         uploader,
		profileBucket:    cfg.ProfileBucket,
		resumeBucket:     cfg.ResumeBucket,
		region:           cfg.Region,
		profilePublicURL: profilePublicURL,
		resumePublicURL:  resumePublicURL,
	}, nil
}

// UploadFile uploads a file to S3 and returns the public URL
func (s *S3Storage) UploadFile(ctx context.Context, file io.Reader, bucket string, key string, contentType string, publicURL string, forceDownload bool) (string, error) {
	// Build PutObjectInput
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead, // Make file publicly accessible and downloadable
	}

	// Add Content-Disposition header to force download if needed
	if forceDownload {
		filename := filepath.Base(key)
		input.ContentDisposition = aws.String(fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// Upload file
	_, err := s.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Return public URL (directly downloadable)
	url := fmt.Sprintf("%s/%s", publicURL, key)
	return url, nil
}

// UploadResume uploads a resume file to S3 resume bucket
func (s *S3Storage) UploadResume(ctx context.Context, file io.Reader, userID string, filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".pdf" // Default to PDF
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

	// Force download for resumes
	return s.UploadFile(ctx, file, s.resumeBucket, key, contentType, s.resumePublicURL, true)
}

// UploadProfilePic uploads a profile picture to S3 profile bucket
func (s *S3Storage) UploadProfilePic(ctx context.Context, file io.Reader, userID string, filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg" // Default to JPG
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

	// Don't force download for profile pictures (display in browser)
	return s.UploadFile(ctx, file, s.profileBucket, key, contentType, s.profilePublicURL, false)
}

// DeleteResume deletes a resume file from S3
func (s *S3Storage) DeleteResume(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.resumeBucket),
		Key:    aws.String(key),
	})
	return err
}

// DeleteProfilePic deletes a profile picture from S3
func (s *S3Storage) DeleteProfilePic(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.profileBucket),
		Key:    aws.String(key),
	})
	return err
}

// GeneratePresignedResumeURL generates a presigned URL for resume download
func (s *S3Storage) GeneratePresignedResumeURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.resumeBucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String("attachment"), // Force download
	}, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// GeneratePresignedProfileURL generates a presigned URL for profile picture
func (s *S3Storage) GeneratePresignedProfileURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.profileBucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}
