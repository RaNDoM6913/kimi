package rate

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
)

func TestCheckLikeRateBlocksThirdActionInOneSecondWindow(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	limiter := NewLimiter(repo, 2, 12, 45)

	ctx := context.Background()
	userID := int64(42)
	sid := "sid-42"
	ip := "127.0.0.1"

	allowed, retryAfter, reason := limiter.CheckLikeRate(ctx, userID, sid, ip)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("first like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, userID, sid, ip)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("second like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, userID, sid, ip)
	if allowed {
		t.Fatalf("expected third like in <1s to be blocked")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry_after for blocked third like, got %d", retryAfter)
	}
	if reason != "user_1s" {
		t.Fatalf("unexpected block reason: %q", reason)
	}
}

func TestLimiterAllowsAfterWindowExpires(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	limiter := NewLimiter(repo, 2, 12, 45)

	ctx := context.Background()
	userID := int64(77)

	for i := 0; i < 3; i++ {
		limiter.CheckLikeRate(ctx, userID, "sid-77", "127.0.0.1")
	}

	mr.FastForward(1100 * time.Millisecond)
	allowed, retryAfter, reason := limiter.CheckLikeRate(ctx, userID, "sid-77", "127.0.0.1")
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("unexpected result after 1s window expiration: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}
}

func newMiniRedisClient(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}
