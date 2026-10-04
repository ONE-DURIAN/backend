package s3client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"community-backend/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	Client *minio.Client
	Bucket string
}

func ConnectS3(cfg config.Config) (*S3Client, error) {
	endpoint := cfg.S3Endpoint
	useSSL := false
	if strings.HasPrefix(endpoint, "https://") {
		useSSL = true
		endpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: useSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init s3 client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure bucket exists
	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check s3 bucket '%s': %w", cfg.S3Bucket, err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create s3 bucket '%s': %w", cfg.S3Bucket, err)
		}
	}

	return &S3Client{
		Client: client,
		Bucket: cfg.S3Bucket,
	}, nil
}

func (s *S3Client) HealthCheck(ctx context.Context) (float64, bool, error) {
	start := time.Now()
	exists, err := s.Client.BucketExists(ctx, s.Bucket)
	if err != nil {
		return 0, false, err
	}
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	return latency, exists, nil
}
