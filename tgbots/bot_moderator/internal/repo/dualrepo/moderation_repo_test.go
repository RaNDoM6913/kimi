package dualrepo

import (
	"context"
	"net/http"
	"testing"
	"time"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/adminhttp"
)

type stubModerationRepo struct {
	acquireFn    func(context.Context, int64, time.Duration) (model.ModerationItem, error)
	acquireCalls int
}

func (s *stubModerationRepo) AcquireNextPending(ctx context.Context, actorTGID int64, lockDuration time.Duration) (model.ModerationItem, error) {
	s.acquireCalls++
	if s.acquireFn != nil {
		return s.acquireFn(ctx, actorTGID, lockDuration)
	}
	return model.ModerationItem{}, nil
}

func (s *stubModerationRepo) GetProfile(context.Context, int64) (model.ModerationProfile, error) {
	return model.ModerationProfile{}, nil
}

func (s *stubModerationRepo) ListPhotoKeys(context.Context, int64, int) ([]string, error) {
	return []string{}, nil
}

func (s *stubModerationRepo) GetLatestCircleKey(context.Context, int64) (string, error) {
	return "", nil
}

func (s *stubModerationRepo) GetByID(context.Context, int64) (model.ModerationItem, error) {
	return model.ModerationItem{}, nil
}

func (s *stubModerationRepo) MarkApproved(context.Context, int64) error {
	return nil
}

func (s *stubModerationRepo) MarkRejected(context.Context, int64, string, string, string) error {
	return nil
}

func (s *stubModerationRepo) InsertModerationAction(context.Context, model.BotModerationAction) error {
	return nil
}

func TestDualRepoHTTPOkDoesNotCallDB(t *testing.T) {
	t.Parallel()

	httpRepo := &stubModerationRepo{
		acquireFn: func(context.Context, int64, time.Duration) (model.ModerationItem, error) {
			return model.ModerationItem{ID: 77, UserID: 42}, nil
		},
	}
	dbRepo := &stubModerationRepo{}
	repo := NewModerationRepo(httpRepo, dbRepo, ModeDual)

	item, err := repo.AcquireNextPending(context.Background(), 1001, 10*time.Minute)
	if err != nil {
		t.Fatalf("acquire next pending: %v", err)
	}
	if item.ID != 77 {
		t.Fatalf("unexpected item id: %d", item.ID)
	}
	if httpRepo.acquireCalls != 1 {
		t.Fatalf("expected http repo call, got %d", httpRepo.acquireCalls)
	}
	if dbRepo.acquireCalls != 0 {
		t.Fatalf("expected db repo not called, got %d", dbRepo.acquireCalls)
	}
}

func TestDualRepoHTTPTimeoutFallsBackToDB(t *testing.T) {
	t.Parallel()

	httpRepo := &stubModerationRepo{
		acquireFn: func(context.Context, int64, time.Duration) (model.ModerationItem, error) {
			return model.ModerationItem{}, &adminhttp.RequestError{
				Op:           "execute http request",
				Fallbackable: true,
				Err:          context.DeadlineExceeded,
			}
		},
	}
	dbRepo := &stubModerationRepo{
		acquireFn: func(context.Context, int64, time.Duration) (model.ModerationItem, error) {
			return model.ModerationItem{ID: 88, UserID: 51}, nil
		},
	}
	repo := NewModerationRepo(httpRepo, dbRepo, ModeDual)

	item, err := repo.AcquireNextPending(context.Background(), 1002, 10*time.Minute)
	if err != nil {
		t.Fatalf("acquire next pending with fallback: %v", err)
	}
	if item.ID != 88 {
		t.Fatalf("unexpected fallback item id: %d", item.ID)
	}
	if httpRepo.acquireCalls != 1 {
		t.Fatalf("expected http repo called once, got %d", httpRepo.acquireCalls)
	}
	if dbRepo.acquireCalls != 1 {
		t.Fatalf("expected db fallback called once, got %d", dbRepo.acquireCalls)
	}
}

func TestDualRepoHTTP401DoesNotCallDB(t *testing.T) {
	t.Parallel()

	httpRepo := &stubModerationRepo{
		acquireFn: func(context.Context, int64, time.Duration) (model.ModerationItem, error) {
			return model.ModerationItem{}, &adminhttp.RequestError{
				Op:           "unexpected http status",
				StatusCode:   http.StatusUnauthorized,
				Fallbackable: false,
			}
		},
	}
	dbRepo := &stubModerationRepo{}
	repo := NewModerationRepo(httpRepo, dbRepo, ModeDual)

	_, err := repo.AcquireNextPending(context.Background(), 1003, 10*time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
	if httpRepo.acquireCalls != 1 {
		t.Fatalf("expected http repo called once, got %d", httpRepo.acquireCalls)
	}
	if dbRepo.acquireCalls != 0 {
		t.Fatalf("expected db repo not called on 401, got %d", dbRepo.acquireCalls)
	}
}
