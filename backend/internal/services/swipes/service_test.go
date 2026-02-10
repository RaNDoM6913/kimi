package swipes

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
	ratesvc "github.com/ivankudzin/tgapp/backend/internal/services/rate"
)

type antiAbuseStub struct {
	calls        int
	user         int64
	at           time.Time
	lastWeight   int
	lastCooldown int
}

func (s *antiAbuseStub) ApplyDecay(_ context.Context, _ int64, _ time.Time) (antiabusesvc.State, error) {
	return antiabusesvc.State{}, nil
}

func (s *antiAbuseStub) ApplyViolation(_ context.Context, userID int64, weight int, now time.Time) (antiabusesvc.State, error) {
	s.calls++
	s.user = userID
	s.at = now
	s.lastWeight = weight
	return antiabusesvc.State{RiskScore: weight, Exists: true}, nil
}

func (s *antiAbuseStub) ApplyViolationWithCooldown(_ context.Context, userID int64, weight int, cooldownSec int, now time.Time) (antiabusesvc.State, error) {
	s.calls++
	s.user = userID
	s.at = now
	s.lastWeight = weight
	s.lastCooldown = cooldownSec
	cooldown := now.Add(time.Duration(cooldownSec) * time.Second)
	return antiabusesvc.State{
		RiskScore:     weight,
		CooldownUntil: &cooldown,
		Exists:        true,
	}, nil
}

type telemetryStub struct {
	userID *int64
	events []analyticsvc.BatchEvent
}

func (s *telemetryStub) IngestBatch(_ context.Context, userID *int64, events []analyticsvc.BatchEvent) error {
	if userID != nil {
		uid := *userID
		s.userID = &uid
	}
	s.events = append([]analyticsvc.BatchEvent(nil), events...)
	return nil
}

type rateLimiterStub struct {
	allowed    bool
	retryAfter int
	reason     string
}

func (s rateLimiterStub) CheckLikeRate(context.Context, int64, string, string, string) (bool, int, string) {
	return s.allowed, s.retryAfter, s.reason
}

type deviceRegistryStub struct {
	known         bool
	knownErr      error
	upsertErr     error
	upsertCalls   int
	lastUserID    int64
	lastDeviceID  string
	lastSeenValue time.Time
}

func (s *deviceRegistryStub) IsKnown(context.Context, int64, string) (bool, error) {
	return s.known, s.knownErr
}

func (s *deviceRegistryStub) UpsertSeen(_ context.Context, userID int64, deviceID string, seenAt time.Time) error {
	s.upsertCalls++
	s.lastUserID = userID
	s.lastDeviceID = deviceID
	s.lastSeenValue = seenAt
	return s.upsertErr
}

func TestApplyLowCardViewViolationForLike(t *testing.T) {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	velocity := 1.35

	ab := &antiAbuseStub{}
	tel := &telemetryStub{}
	svc := &Service{
		antiAbuse: ab,
		telemetry: tel,
		cfg: Config{
			MinCardViewMS: 700,
		},
	}

	svc.applyLowCardViewViolation(context.Background(), 101, 202, actionLike, SwipeClientTelemetry{
		CardViewMS:    420,
		SwipeVelocity: &velocity,
		Screen:        "feed",
	}, now)

	if ab.calls != 1 {
		t.Fatalf("expected ApplyViolation to be called once, got %d", ab.calls)
	}
	if ab.user != 101 {
		t.Fatalf("unexpected user id passed to ApplyViolation: got %d", ab.user)
	}
	if !ab.at.Equal(now) {
		t.Fatalf("unexpected violation timestamp: got %v want %v", ab.at, now)
	}

	if tel.userID == nil || *tel.userID != 101 {
		t.Fatalf("unexpected telemetry user id: %+v", tel.userID)
	}
	if len(tel.events) != 1 {
		t.Fatalf("expected one telemetry event, got %d", len(tel.events))
	}

	event := tel.events[0]
	if event.Name != "antiabuse_low_card_view" {
		t.Fatalf("unexpected event name: %s", event.Name)
	}
	if event.TS != now.UnixMilli() {
		t.Fatalf("unexpected event ts: got %d want %d", event.TS, now.UnixMilli())
	}
	if got, ok := event.Props["card_view_ms"].(int); !ok || got != 420 {
		t.Fatalf("unexpected card_view_ms prop: %+v", event.Props["card_view_ms"])
	}
	if got, ok := event.Props["min_card_view_ms"].(int); !ok || got != 700 {
		t.Fatalf("unexpected min_card_view_ms prop: %+v", event.Props["min_card_view_ms"])
	}
	if got, ok := event.Props["action"].(string); !ok || got != actionLike {
		t.Fatalf("unexpected action prop: %+v", event.Props["action"])
	}
	if got, ok := event.Props["screen"].(string); !ok || got != "feed" {
		t.Fatalf("unexpected screen prop: %+v", event.Props["screen"])
	}
}

func TestApplyLowCardViewViolationSkippedForDislike(t *testing.T) {
	ab := &antiAbuseStub{}
	tel := &telemetryStub{}
	svc := &Service{
		antiAbuse: ab,
		telemetry: tel,
		cfg: Config{
			MinCardViewMS: 700,
		},
	}

	svc.applyLowCardViewViolation(context.Background(), 101, 202, actionDislike, SwipeClientTelemetry{CardViewMS: 100}, time.Now().UTC())

	if ab.calls != 0 {
		t.Fatalf("expected no risk violation call for dislike, got %d", ab.calls)
	}
	if len(tel.events) != 0 {
		t.Fatalf("expected no telemetry events for dislike, got %d", len(tel.events))
	}
}

func TestSwipeLikeGatesBlockAndEscalateCooldown(t *testing.T) {
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

	svc := NewService(Dependencies{
		RateLimiter: rateLimiter,
		AntiAbuse:   antiAbuse,
	}, Config{})

	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	ctx := context.Background()

	_, _ = svc.Swipe(ctx, 101, 202, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})
	_, _ = svc.Swipe(ctx, 101, 203, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})

	_, err = svc.Swipe(ctx, 101, 204, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})
	firstRateLimit, ok := IsTooFast(err)
	if !ok {
		t.Fatalf("expected TooFastError on third like, got %v", err)
	}
	if firstRateLimit.RetryAfter() <= 0 {
		t.Fatalf("expected positive retry_after for TooFast, got %d", firstRateLimit.RetryAfter())
	}
	if firstRateLimit.CooldownUntil == nil {
		t.Fatalf("expected cooldown_until in TooFast response")
	}
	firstCooldownUntil := *firstRateLimit.CooldownUntil

	_, err = svc.Swipe(ctx, 101, 205, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})
	cooldownActive, ok := IsCooldownActive(err)
	if !ok {
		t.Fatalf("expected CooldownActiveError after rate limit, got %v", err)
	}
	if cooldownActive.RetryAfter() <= 0 {
		t.Fatalf("expected positive retry_after for cooldown, got %d", cooldownActive.RetryAfter())
	}
	if cooldownActive.CooldownUntil == nil {
		t.Fatalf("expected cooldown_until for cooldown-active response")
	}

	advance := firstCooldownUntil.Add(time.Second).Sub(now)
	mr.FastForward(advance)
	now = now.Add(advance)

	_, _ = svc.Swipe(ctx, 101, 206, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})
	_, _ = svc.Swipe(ctx, 101, 207, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})

	_, err = svc.Swipe(ctx, 101, 208, actionLike, "UTC", "sid-101", "127.0.0.1", "device-101", SwipeClientTelemetry{})
	secondRateLimit, ok := IsTooFast(err)
	if !ok {
		t.Fatalf("expected TooFastError on repeated burst, got %v", err)
	}
	if secondRateLimit.CooldownUntil == nil {
		t.Fatalf("expected cooldown_until on repeated TooFast")
	}
	if !secondRateLimit.CooldownUntil.After(firstCooldownUntil) {
		t.Fatalf("expected cooldown escalation, first=%v second=%v", firstCooldownUntil, *secondRateLimit.CooldownUntil)
	}
}

func TestShouldMarkLikeAsSuspectByRiskThreshold(t *testing.T) {
	svc := &Service{
		cfg: Config{
			SuspectLikeThreshold: 8,
		},
	}

	if svc.shouldMarkLikeAsSuspect(antiabusesvc.State{RiskScore: 7}) {
		t.Fatalf("risk 7 should not be suspect for threshold 8")
	}
	if !svc.shouldMarkLikeAsSuspect(antiabusesvc.State{RiskScore: 8}) {
		t.Fatalf("risk 8 should be suspect for threshold 8")
	}
	if !svc.shouldMarkLikeAsSuspect(antiabusesvc.State{RiskScore: 12}) {
		t.Fatalf("risk 12 should be suspect for threshold 8")
	}
}

func TestLogSuspectLikeEvent(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 34, 0, 0, time.UTC)
	tel := &telemetryStub{}
	svc := &Service{
		telemetry: tel,
		cfg: Config{
			SuspectLikeThreshold: 8,
		},
	}

	svc.logSuspectLikeEvent(context.Background(), 101, 202, actionLike, true, 9, now)

	if tel.userID == nil || *tel.userID != 101 {
		t.Fatalf("unexpected telemetry user id: %+v", tel.userID)
	}
	if len(tel.events) != 1 {
		t.Fatalf("expected one suspect-like event, got %d", len(tel.events))
	}
	event := tel.events[0]
	if event.Name != "antiabuse_suspect_like" {
		t.Fatalf("unexpected event name: %s", event.Name)
	}
	if event.TS != now.UnixMilli() {
		t.Fatalf("unexpected event ts: got %d want %d", event.TS, now.UnixMilli())
	}
	if got, ok := event.Props["risk_score"].(int); !ok || got != 9 {
		t.Fatalf("unexpected risk_score prop: %+v", event.Props["risk_score"])
	}
	if got, ok := event.Props["target_id"].(int64); !ok || got != 202 {
		t.Fatalf("unexpected target_id prop: %+v", event.Props["target_id"])
	}
}

func TestApplyLikeGatesLogsTooFastAndCooldownAppliedEvents(t *testing.T) {
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
	tel := &telemetryStub{}

	now := time.Date(2026, 2, 10, 16, 0, 0, 0, time.UTC)
	svc := &Service{
		rateLimiter: rateLimiter,
		antiAbuse:   antiAbuse,
		telemetry:   tel,
		now:         func() time.Time { return now },
	}

	ctx := context.Background()
	if _, err := svc.applyLikeGates(ctx, 501, "sid-501", "127.0.0.1", "device-501", now); err != nil {
		t.Fatalf("gate #1: %v", err)
	}
	if _, err := svc.applyLikeGates(ctx, 501, "sid-501", "127.0.0.1", "device-501", now); err != nil {
		t.Fatalf("gate #2: %v", err)
	}
	if _, err := svc.applyLikeGates(ctx, 501, "sid-501", "127.0.0.1", "device-501", now); err == nil {
		t.Fatalf("expected too-fast error on gate #3")
	}

	if tel.userID == nil || *tel.userID != 501 {
		t.Fatalf("unexpected telemetry user id: %+v", tel.userID)
	}
	if len(tel.events) != 2 {
		t.Fatalf("expected two antiabuse events on too-fast, got %d", len(tel.events))
	}
	if tel.events[0].Name != "antiabuse_too_fast" {
		t.Fatalf("unexpected first event name: %s", tel.events[0].Name)
	}
	if tel.events[1].Name != "antiabuse_cooldown_applied" {
		t.Fatalf("unexpected second event name: %s", tel.events[1].Name)
	}
}

func TestApplyLikeGatesReturnsTempUnavailableWithoutViolation(t *testing.T) {
	ab := &antiAbuseStub{}
	svc := &Service{
		rateLimiter: rateLimiterStub{
			allowed:    false,
			retryAfter: 10,
			reason:     "temp_unavailable",
		},
		antiAbuse: ab,
	}

	_, err := svc.applyLikeGates(context.Background(), 101, "sid-101", "127.0.0.1", "device-101", time.Now().UTC())
	tu, ok := IsTempUnavailable(err)
	if !ok {
		t.Fatalf("expected TempUnavailableError, got %v", err)
	}
	if tu.RetryAfter() != 10 {
		t.Fatalf("unexpected retry_after: got %d want %d", tu.RetryAfter(), 10)
	}
	if ab.calls != 0 {
		t.Fatalf("risk violation must not be applied when redis is unavailable")
	}
}

func TestHandleDeviceRegistrationNewDeviceIncreasesRiskScore(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	riskRepo := redrepo.NewRiskRepo(redisClient)
	antiAbuse := antiabusesvc.NewService(riskRepo, antiabusesvc.Config{
		RiskDecayHours:   6,
		CooldownStepsSec: []int{30, 60, 300, 1800, 86400},
		ShadowThreshold:  5,
	})
	devices := &deviceRegistryStub{known: false}
	now := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)

	svc := &Service{
		antiAbuse: antiAbuse,
		devices:   devices,
		cfg: Config{
			NewDeviceRiskWeight:  3,
			NewDeviceCooldownSec: 30,
		},
		now: func() time.Time { return now },
	}

	state, err := svc.handleDeviceRegistration(context.Background(), 101, "new-device-101", now)
	if err != nil {
		t.Fatalf("handle device registration: %v", err)
	}
	if state.RiskScore != 3 {
		t.Fatalf("expected risk_score=3 for new device, got %d", state.RiskScore)
	}
	if state.CooldownUntil == nil {
		t.Fatalf("expected cooldown to be set for new device")
	}
	if got, want := *state.CooldownUntil, now.Add(30*time.Second); !got.Equal(want) {
		t.Fatalf("unexpected cooldown_until: got %v want %v", got, want)
	}
	if devices.upsertCalls != 1 {
		t.Fatalf("expected one upsert call, got %d", devices.upsertCalls)
	}
}
