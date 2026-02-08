package media

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

type S3Storage struct {
	client *minio.Client
	bucket string

	ensureOnce sync.Once
	ensureErr  error
}

func NewS3Storage(client *minio.Client, bucket string) *S3Storage {
	return &S3Storage{
		client: client,
		bucket: strings.TrimSpace(bucket),
	}
}

func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("s3 client is nil")
	}
	if s.bucket == "" {
		return fmt.Errorf("s3 bucket is empty")
	}

	s.ensureOnce.Do(func() {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			s.ensureErr = err
			return
		}
		if exists {
			return
		}
		s.ensureErr = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	})

	if s.ensureErr != nil {
		return fmt.Errorf("ensure s3 bucket %q: %w", s.bucket, s.ensureErr)
	}

	return nil
}

func (s *S3Storage) PutPhoto(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if s.client == nil {
		return fmt.Errorf("s3 client is nil")
	}
	if key == "" || body == nil || size == 0 {
		return ErrValidation
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object to s3: %w", err)
	}

	return nil
}

func (s *S3Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("s3 client is nil")
	}
	if key == "" {
		return "", ErrValidation
	}
	if ttl <= 0 {
		ttl = signedURLTTL
	}

	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}

	return presigned.String(), nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if s.client == nil || key == "" {
		return nil
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}
