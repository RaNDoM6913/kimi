package matches

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
)

func TestCheckReportRateBlocksFourthReportInTenMinutes(t *testing.T) {
	mr, client := newMiniRedisClient(t)
	defer mr.Close()
	defer func() { _ = client.Close() }()

	repo := redrepo.NewRateRepo(client)
	svc := NewService(Dependencies{
		ReportRateStore:   repo,
		ReportMaxPer10Min: 3,
	})

	ctx := context.Background()
	userID := int64(701)

	for i := 0; i < 3; i++ {
		if err := svc.checkReportRate(ctx, userID); err != nil {
			t.Fatalf("unexpected report rate error on attempt %d: %v", i+1, err)
		}
	}

	err := svc.checkReportRate(ctx, userID)
	rl, ok := IsTooManyReports(err)
	if !ok {
		t.Fatalf("expected TooManyReportsError on 4th report, got %v", err)
	}
	if rl.RetryAfter() <= 0 {
		t.Fatalf("expected positive retry_after for blocked report, got %d", rl.RetryAfter())
	}
}

func TestReporterTrustScoreByRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected int
	}{
		{name: "admin", role: "admin", expected: 100},
		{name: "moderator", role: "moderator", expected: 50},
		{name: "user", role: "user", expected: 10},
		{name: "unknown falls back to user", role: "unknown", expected: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := reporterTrustScore(tc.role); got != tc.expected {
				t.Fatalf("unexpected trust score: got %d want %d", got, tc.expected)
			}
		})
	}
}

func TestCheckReportRateFailsClosedOnRedisError(t *testing.T) {
	svc := NewService(Dependencies{
		ReportRateStore:   reportRateStoreErrStub{err: errors.New("redis unavailable")},
		ReportMaxPer10Min: 3,
	})

	err := svc.checkReportRate(context.Background(), 999)
	tu, ok := IsTempUnavailable(err)
	if !ok {
		t.Fatalf("expected TempUnavailableError, got %v", err)
	}
	if tu.RetryAfter() != 10 {
		t.Fatalf("unexpected retry_after: got %d want %d", tu.RetryAfter(), 10)
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

type reportRateStoreErrStub struct {
	err error
}

func (s reportRateStoreErrStub) IncrementWindow(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, s.err
}
