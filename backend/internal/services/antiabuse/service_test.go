package antiabuse

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
)

func TestApplyViolationEscalatesCooldown(t *testing.T) {
	repo, cleanup := newRiskRepo(t)
	defer cleanup()

	svc := NewService(repo, Config{
		RiskDecayHours:   6,
		CooldownStepsSec: []int{30, 60, 300, 1800, 86400},
		ShadowThreshold:  5,
	})

	ctx := context.Background()
	userID := int64(101)
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	first, err := svc.ApplyViolation(ctx, userID, 1, now)
	if err != nil {
		t.Fatalf("apply first violation: %v", err)
	}
	if first.RiskScore != 1 {
		t.Fatalf("unexpected risk after first violation: %d", first.RiskScore)
	}
	if first.CooldownUntil == nil {
		t.Fatalf("expected cooldown after first violation")
	}

	second, err := svc.ApplyViolation(ctx, userID, 1, now.Add(time.Second))
	if err != nil {
		t.Fatalf("apply second violation: %v", err)
	}
	if second.RiskScore != 2 {
		t.Fatalf("unexpected risk after second violation: %d", second.RiskScore)
	}
	if second.CooldownUntil == nil {
		t.Fatalf("expected cooldown after second violation")
	}
	if !second.CooldownUntil.After(*first.CooldownUntil) {
		t.Fatalf("expected cooldown escalation: first=%v second=%v", *first.CooldownUntil, *second.CooldownUntil)
	}
}

func TestApplyDecayReducesRiskByHoursWithoutViolations(t *testing.T) {
	repo, cleanup := newRiskRepo(t)
	defer cleanup()

	svc := NewService(repo, Config{
		RiskDecayHours:   6,
		CooldownStepsSec: []int{30, 60, 300, 1800, 86400},
		ShadowThreshold:  5,
	})

	ctx := context.Background()
	userID := int64(202)
	start := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	state, err := svc.ApplyViolation(ctx, userID, 3, start)
	if err != nil {
		t.Fatalf("apply violation: %v", err)
	}
	if state.RiskScore != 3 {
		t.Fatalf("unexpected initial risk: %d", state.RiskScore)
	}

	decayed, err := svc.ApplyDecay(ctx, userID, start.Add(6*time.Hour+time.Minute))
	if err != nil {
		t.Fatalf("apply decay #1: %v", err)
	}
	if decayed.RiskScore != 2 {
		t.Fatalf("unexpected risk after first decay: %d", decayed.RiskScore)
	}

	decayedAgain, err := svc.ApplyDecay(ctx, userID, start.Add(6*time.Hour+time.Minute))
	if err != nil {
		t.Fatalf("apply decay idempotency check: %v", err)
	}
	if decayedAgain.RiskScore != 2 {
		t.Fatalf("risk should not decay twice for same interval: %d", decayedAgain.RiskScore)
	}

	decayedTwice, err := svc.ApplyDecay(ctx, userID, start.Add(12*time.Hour+2*time.Minute))
	if err != nil {
		t.Fatalf("apply decay #2: %v", err)
	}
	if decayedTwice.RiskScore != 1 {
		t.Fatalf("unexpected risk after second decay: %d", decayedTwice.RiskScore)
	}
}

type antiabuseTelemetryStub struct {
	userID *int64
	events []analyticsvc.BatchEvent
}

func (s *antiabuseTelemetryStub) IngestBatch(_ context.Context, userID *int64, events []analyticsvc.BatchEvent) error {
	if userID != nil {
		uid := *userID
		s.userID = &uid
	}
	s.events = append(s.events, events...)
	return nil
}

func TestApplyViolationEmitsShadowEnabledOnceOnThresholdCross(t *testing.T) {
	repo, cleanup := newRiskRepo(t)
	defer cleanup()

	telemetry := &antiabuseTelemetryStub{}
	svc := NewService(repo, Config{
		RiskDecayHours:   6,
		CooldownStepsSec: []int{30, 60, 300, 1800, 86400},
		ShadowThreshold:  2,
	})
	svc.AttachTelemetry(telemetry)

	ctx := context.Background()
	userID := int64(303)
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	if _, err := svc.ApplyViolation(ctx, userID, 1, now); err != nil {
		t.Fatalf("apply violation #1: %v", err)
	}
	if len(telemetry.events) != 0 {
		t.Fatalf("unexpected event before threshold: %+v", telemetry.events)
	}

	if _, err := svc.ApplyViolation(ctx, userID, 1, now.Add(time.Second)); err != nil {
		t.Fatalf("apply violation #2: %v", err)
	}
	if len(telemetry.events) != 1 {
		t.Fatalf("expected one shadow event on threshold cross, got %d", len(telemetry.events))
	}
	if telemetry.events[0].Name != "antiabuse_shadow_enabled" {
		t.Fatalf("unexpected event name: %s", telemetry.events[0].Name)
	}

	if _, err := svc.ApplyViolation(ctx, userID, 1, now.Add(2*time.Second)); err != nil {
		t.Fatalf("apply violation #3: %v", err)
	}
	if len(telemetry.events) != 1 {
		t.Fatalf("expected single shadow event after already-shadowed user, got %d", len(telemetry.events))
	}
}

func newRiskRepo(t *testing.T) (*redrepo.RiskRepo, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	repo := redrepo.NewRiskRepo(client)

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}
	return repo, cleanup
}
