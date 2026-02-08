package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

var (
	ErrValidation        = errors.New("validation error")
	ErrPhotoLimitReached = errors.New("photo limit reached")
)

const (
	photoKind       = "photo"
	photoStatus     = "active"
	signedURLTTL    = 5 * time.Minute
	maxActivePhotos = 3
)

type Store interface {
	CreatePhoto(ctx context.Context, userID int64, objectKey string) (PhotoRecord, error)
	ListActivePhotos(ctx context.Context, userID int64) ([]PhotoRecord, error)
}

type ObjectStorage interface {
	EnsureBucket(ctx context.Context) error
	PutPhoto(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type Service struct {
	store   Store
	storage ObjectStorage
	now     func() time.Time
}

type PhotoRecord struct {
	ID        int64
	Position  int
	ObjectKey string
	CreatedAt time.Time
}

type Photo struct {
	ID        int64
	Position  int
	URL       string
	CreatedAt time.Time
}

func NewService(store Store, storage ObjectStorage) *Service {
	return &Service{
		store:   store,
		storage: storage,
		now:     time.Now,
	}
}

func (s *Service) UploadPhoto(ctx context.Context, userID int64, fileName, contentType string, body io.Reader, size int64) (Photo, error) {
	if userID <= 0 || body == nil || size <= 0 {
		return Photo{}, ErrValidation
	}
	if s.store == nil || s.storage == nil {
		return Photo{}, fmt.Errorf("media dependencies are not configured")
	}

	if err := s.storage.EnsureBucket(ctx); err != nil {
		return Photo{}, fmt.Errorf("ensure bucket: %w", err)
	}

	objectKey, err := buildPhotoObjectKey(userID, fileName)
	if err != nil {
		return Photo{}, fmt.Errorf("build object key: %w", err)
	}

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	if err := s.storage.PutPhoto(ctx, objectKey, body, size, contentType); err != nil {
		return Photo{}, fmt.Errorf("put object: %w", err)
	}

	record, err := s.store.CreatePhoto(ctx, userID, objectKey)
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		if errors.Is(err, ErrPhotoLimitReached) {
			return Photo{}, ErrPhotoLimitReached
		}
		return Photo{}, fmt.Errorf("create photo record: %w", err)
	}

	url, err := s.storage.PresignGet(ctx, record.ObjectKey, signedURLTTL)
	if err != nil {
		return Photo{}, fmt.Errorf("presign photo url: %w", err)
	}

	return Photo{
		ID:        record.ID,
		Position:  record.Position,
		URL:       url,
		CreatedAt: record.CreatedAt,
	}, nil
}

func (s *Service) ListPhotos(ctx context.Context, userID int64) ([]Photo, error) {
	if userID <= 0 {
		return nil, ErrValidation
	}
	if s.store == nil || s.storage == nil {
		return nil, fmt.Errorf("media dependencies are not configured")
	}

	records, err := s.store.ListActivePhotos(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list media records: %w", err)
	}

	photos := make([]Photo, 0, len(records))
	for _, rec := range records {
		url, err := s.storage.PresignGet(ctx, rec.ObjectKey, signedURLTTL)
		if err != nil {
			return nil, fmt.Errorf("presign photo url: %w", err)
		}
		photos = append(photos, Photo{
			ID:        rec.ID,
			Position:  rec.Position,
			URL:       url,
			CreatedAt: rec.CreatedAt,
		})
	}

	return photos, nil
}

func buildPhotoObjectKey(userID int64, fileName string) (string, error) {
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return "", err
	}

	ext := strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
	if ext == "" {
		ext = ".bin"
	}

	stamp := time.Now().UTC().Format("20060102T150405")
	return fmt.Sprintf("users/%d/photos/%s_%s%s", userID, stamp, hex.EncodeToString(rnd), ext), nil
}

func MaxActivePhotos() int {
	return maxActivePhotos
}
