package rate

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
)

func TestLimiterBlocksOn10SecondWindow(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	limiter := NewLimiter(repo, 100, 2)

	ctx := context.Background()
	userID := int64(42)

	for i := 0; i < 2; i++ {
		retryAfter, allowed, err := limiter.AllowLike(ctx, userID)
		if err != nil {
			t.Fatalf("allow like #%d: %v", i+1, err)
		}
		if !allowed || retryAfter != 0 {
			t.Fatalf("unexpected result on allow #%d: allowed=%v retry_after=%d", i+1, allowed, retryAfter)
		}
	}

	retryAfter, allowed, err := limiter.AllowLike(ctx, userID)
	if err != nil {
		t.Fatalf("allow like #3: %v", err)
	}
	if allowed {
		t.Fatalf("expected limiter block on third action in 10s window")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry_after, got %d", retryAfter)
	}

	currentRetry, err := limiter.RetryAfterLike(ctx, userID)
	if err != nil {
		t.Fatalf("retry_after state: %v", err)
	}
	if currentRetry <= 0 {
		t.Fatalf("expected positive retry_after state, got %d", currentRetry)
	}

	mr.FastForward(11 * time.Second)

	retryAfter, allowed, err = limiter.AllowLike(ctx, userID)
	if err != nil {
		t.Fatalf("allow like after 10s window: %v", err)
	}
	if !allowed || retryAfter != 0 {
		t.Fatalf("unexpected result after fast forward: allowed=%v retry_after=%d", allowed, retryAfter)
	}
}

func TestLimiterBlocksOnMinuteWindow(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	limiter := NewLimiter(repo, 3, 100)

	ctx := context.Background()
	userID := int64(77)

	for i := 0; i < 3; i++ {
		retryAfter, allowed, err := limiter.AllowLike(ctx, userID)
		if err != nil {
			t.Fatalf("allow like #%d: %v", i+1, err)
		}
		if !allowed || retryAfter != 0 {
			t.Fatalf("unexpected result on allow #%d: allowed=%v retry_after=%d", i+1, allowed, retryAfter)
		}
	}

	retryAfter, allowed, err := limiter.AllowLike(ctx, userID)
	if err != nil {
		t.Fatalf("allow like #4: %v", err)
	}
	if allowed {
		t.Fatalf("expected limiter block on fourth action in minute window")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry_after, got %d", retryAfter)
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
