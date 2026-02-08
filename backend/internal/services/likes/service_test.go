package likes

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type memoryQuotaStore struct {
	usage map[string]int
}

func newMemoryQuotaStore() *memoryQuotaStore {
	return &memoryQuotaStore{
		usage: make(map[string]int),
	}
}

func (s *memoryQuotaStore) GetLikesUsed(_ context.Context, userID int64, dayKey string) (int, error) {
	return s.usage[s.key(userID, dayKey)], nil
}

func (s *memoryQuotaStore) IncrementLikesUsed(_ context.Context, userID int64, dayKey, _ string, delta int) (int, error) {
	k := s.key(userID, dayKey)
	s.usage[k] += delta
	return s.usage[k], nil
}

func (s *memoryQuotaStore) key(userID int64, dayKey string) string {
	return fmt.Sprintf("%d:%s", userID, dayKey)
}

func TestConsumeLikeResetsOnLocalMidnight(t *testing.T) {
	store := newMemoryQuotaStore()
	service := NewService(store, nil, nil, Config{
		FreeLikesPerDay: 35,
		DefaultTimezone: "Europe/Minsk",
		DefaultIsPlus:   false,
		PlusUnlimitedUI: true,
	})

	now := time.Date(2026, 2, 8, 20, 30, 0, 0, time.UTC) // 23:30 local
	service.now = func() time.Time { return now }

	ctx := context.Background()
	userID := int64(101)

	for i := 0; i < 35; i++ {
		snapshot, err := service.ConsumeLike(ctx, userID, "")
		if err != nil {
			t.Fatalf("consume like #%d: %v", i+1, err)
		}
		wantLeft := 34 - i
		if snapshot.LikesLeft != wantLeft {
			t.Fatalf("unexpected likes_left after #%d: got %d want %d", i+1, snapshot.LikesLeft, wantLeft)
		}
	}

	if _, err := service.ConsumeLike(ctx, userID, ""); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("expected ErrDailyLimit, got %v", err)
	}

	now = time.Date(2026, 2, 8, 21, 1, 0, 0, time.UTC) // 00:01 next local day
	snapshot, err := service.ConsumeLike(ctx, userID, "")
	if err != nil {
		t.Fatalf("consume like after reset: %v", err)
	}
	if snapshot.LikesLeft != 34 {
		t.Fatalf("unexpected likes_left after reset: got %d want %d", snapshot.LikesLeft, 34)
	}
}
