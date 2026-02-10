package rate

import (
	"context"
	"errors"
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
	deviceID := "device-42"

	allowed, retryAfter, reason := limiter.CheckLikeRate(ctx, userID, sid, ip, deviceID)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("first like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, userID, sid, ip, deviceID)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("second like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, userID, sid, ip, deviceID)
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
		limiter.CheckLikeRate(ctx, userID, "sid-77", "127.0.0.1", "device-77")
	}

	mr.FastForward(1100 * time.Millisecond)
	allowed, retryAfter, reason := limiter.CheckLikeRate(ctx, userID, "sid-77", "127.0.0.1", "device-77")
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("unexpected result after 1s window expiration: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}
}

func TestCheckLikeRateBlocksByDeviceAcrossUsers(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	limiter := NewLimiter(repo, 2, 12, 45)

	ctx := context.Background()
	deviceID := "shared-device-1"

	allowed, retryAfter, reason := limiter.CheckLikeRate(ctx, 1, "sid-1", "127.0.0.1", deviceID)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("first like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, 1, "sid-1", "127.0.0.1", deviceID)
	if !allowed || retryAfter != 0 || reason != "" {
		t.Fatalf("second like unexpected result: allowed=%v retry_after=%d reason=%q", allowed, retryAfter, reason)
	}

	allowed, retryAfter, reason = limiter.CheckLikeRate(ctx, 2, "sid-2", "127.0.0.1", deviceID)
	if allowed {
		t.Fatalf("expected device-level block on third like across users")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry_after for device-level block, got %d", retryAfter)
	}
	if reason != "device_1s" {
		t.Fatalf("unexpected block reason: %q", reason)
	}
}

func TestCheckLikeRateFailClosedWhenRedisUnavailable(t *testing.T) {
	limiter := NewLimiter(rateStoreStub{
		evalErr: errors.New("redis down"),
	}, 2, 12, 45)

	allowed, retryAfter, reason := limiter.CheckLikeRate(context.Background(), 42, "sid-42", "127.0.0.1", "device-42")
	if allowed {
		t.Fatalf("expected fail-closed behavior when redis is unavailable")
	}
	if retryAfter != 10 {
		t.Fatalf("unexpected retry_after on redis outage: got %d want %d", retryAfter, 10)
	}
	if reason != ReasonTempUnavailable {
		t.Fatalf("unexpected reason: got %q want %q", reason, ReasonTempUnavailable)
	}
}

type rateStoreStub struct {
	evalResult interface{}
	evalErr    error
}

func (s rateStoreStub) Eval(context.Context, string, []string, ...interface{}) (interface{}, error) {
	return s.evalResult, s.evalErr
}

func (s rateStoreStub) WindowState(context.Context, string) (int64, time.Duration, error) {
	return 0, 0, nil
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
