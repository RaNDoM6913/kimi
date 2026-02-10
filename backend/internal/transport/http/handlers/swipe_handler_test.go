package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	ratesvc "github.com/ivankudzin/tgapp/backend/internal/services/rate"
	swipesvc "github.com/ivankudzin/tgapp/backend/internal/services/swipes"
)

func TestSwipeHandlerReturnsTooFastOnThirdLikeBurst(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	rateRepo := redrepo.NewRateRepo(redisClient)
	riskRepo := redrepo.NewRiskRepo(redisClient)
	rateLimiter := ratesvc.NewLimiter(rateRepo, 2, 12, 45)
	antiAbuse := antiabusesvc.NewService(riskRepo, antiabusesvc.Config{
		RiskDecayHours:   6,
		CooldownStepsSec: []int{30, 60, 300, 1800, 86400},
		ShadowThreshold:  5,
	})

	svc := swipesvc.NewService(swipesvc.Dependencies{
		RateLimiter: rateLimiter,
		AntiAbuse:   antiAbuse,
	}, swipesvc.Config{})

	h := NewSwipeHandler(svc)

	for i := 0; i < 2; i++ {
		_ = performSwipeRequest(t, h, 1000+int64(i), "LIKE").Code
	}

	resp := performSwipeRequest(t, h, 1002, "LIKE")
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status on third like: got %d want %d", resp.Code, http.StatusTooManyRequests)
	}

	var payload struct {
		Code          string      `json:"code"`
		Message       string      `json:"message"`
		RetryAfterSec int64       `json:"retry_after_sec"`
		CooldownUntil interface{} `json:"cooldown_until"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Code != "TOO_FAST" {
		t.Fatalf("unexpected error code: got %q want %q", payload.Code, "TOO_FAST")
	}
	if payload.RetryAfterSec <= 0 {
		t.Fatalf("expected positive retry_after_sec, got %d", payload.RetryAfterSec)
	}
	if payload.CooldownUntil == nil {
		t.Fatalf("expected cooldown_until in response")
	}
}

func TestSwipeHandlerReturnsTempUnavailableOnRedisError(t *testing.T) {
	rateLimiter := ratesvc.NewLimiter(swipeRateStoreStub{
		evalErr: errors.New("redis unavailable"),
	}, 2, 12, 45)

	svc := swipesvc.NewService(swipesvc.Dependencies{
		RateLimiter: rateLimiter,
	}, swipesvc.Config{})

	h := NewSwipeHandler(svc)
	resp := performSwipeRequest(t, h, 1002, "LIKE")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: got %d want %d", resp.Code, http.StatusServiceUnavailable)
	}

	var payload struct {
		Code          string `json:"code"`
		RetryAfterSec int64  `json:"retry_after_sec"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "TEMP_UNAVAILABLE" {
		t.Fatalf("unexpected error code: got %q want %q", payload.Code, "TEMP_UNAVAILABLE")
	}
	if payload.RetryAfterSec != 10 {
		t.Fatalf("unexpected retry_after_sec: got %d want %d", payload.RetryAfterSec, 10)
	}
}

func performSwipeRequest(t *testing.T, h *SwipeHandler, targetID int64, action string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"target_id": targetID,
		"action":    action,
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/swipe", bytes.NewReader(body))
	ctx := authsvc.WithIdentity(context.Background(), authsvc.Identity{
		UserID: 101,
		SID:    "sid-101",
		Role:   "user",
	})
	ctx = authsvc.WithDeviceID(ctx, "7adf6f94-8cd6-4f6f-9b54-c8eaf9f19fbf")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	return rec
}

type swipeRateStoreStub struct {
	evalErr error
}

func (s swipeRateStoreStub) Eval(context.Context, string, []string, ...interface{}) (interface{}, error) {
	return nil, s.evalErr
}

func (s swipeRateStoreStub) WindowState(context.Context, string) (int64, time.Duration, error) {
	return 0, 0, nil
}
