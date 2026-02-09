package s3

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Signer struct {
	client *minio.Client
	bucket string
}

func NewSigner(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Signer, error) {
	endpoint = strings.TrimSpace(endpoint)
	bucket = strings.TrimSpace(bucket)
	if endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(strings.TrimSpace(accessKey), strings.TrimSpace(secretKey), ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return &Signer{client: client, bucket: bucket}, nil
}

func (s *Signer) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("s3 signer is not initialized")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return presigned.String(), nil
}
