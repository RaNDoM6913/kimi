package media

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	records []PhotoRecord
	nextID  int64
}

func (f *fakeStore) CreatePhoto(_ context.Context, _ int64, objectKey string) (PhotoRecord, error) {
	if len(f.records) >= MaxActivePhotos() {
		return PhotoRecord{}, ErrPhotoLimitReached
	}

	f.nextID++
	rec := PhotoRecord{
		ID:        f.nextID,
		Position:  len(f.records) + 1,
		ObjectKey: objectKey,
		CreatedAt: time.Now().UTC(),
	}
	f.records = append(f.records, rec)
	return rec, nil
}

func (f *fakeStore) ListActivePhotos(_ context.Context, _ int64) ([]PhotoRecord, error) {
	out := make([]PhotoRecord, 0, len(f.records))
	out = append(out, f.records...)
	return out, nil
}

type fakeStorage struct {
	deleteCalls int
}

func (f *fakeStorage) EnsureBucket(_ context.Context) error {
	return nil
}

func (f *fakeStorage) PutPhoto(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}

func (f *fakeStorage) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://signed.local/" + key, nil
}

func (f *fakeStorage) Delete(_ context.Context, _ string) error {
	f.deleteCalls++
	return nil
}

func TestUploadPhotoLimitThree(t *testing.T) {
	store := &fakeStore{}
	storage := &fakeStorage{}
	svc := NewService(store, storage)

	for i := 1; i <= MaxActivePhotos(); i++ {
		photo, err := svc.UploadPhoto(context.Background(), 1, "photo.jpg", "image/jpeg", strings.NewReader("abc"), 3)
		if err != nil {
			t.Fatalf("upload photo #%d: %v", i, err)
		}
		if photo.Position != i {
			t.Fatalf("unexpected photo position: got %d want %d", photo.Position, i)
		}
	}

	_, err := svc.UploadPhoto(context.Background(), 1, "photo4.jpg", "image/jpeg", strings.NewReader("abc"), 3)
	if !errors.Is(err, ErrPhotoLimitReached) {
		t.Fatalf("expected ErrPhotoLimitReached, got %v", err)
	}
	if storage.deleteCalls != 1 {
		t.Fatalf("expected cleanup delete call after limit reached, got %d", storage.deleteCalls)
	}
}
