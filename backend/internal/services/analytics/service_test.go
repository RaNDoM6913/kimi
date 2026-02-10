package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

type analyticsStoreStub struct {
	userID *int64
	events []pgrepo.EventWriteRecord
}

func (s *analyticsStoreStub) InsertBatch(_ context.Context, userID *int64, events []pgrepo.EventWriteRecord) error {
	s.userID = userID
	s.events = append([]pgrepo.EventWriteRecord(nil), events...)
	return nil
}

type antiAbuseObserverStub struct {
	names []string
}

func (s *antiAbuseObserverStub) ObserveEvent(_ context.Context, _ *int64, name string, _ map[string]any) error {
	s.names = append(s.names, name)
	return nil
}

func TestIngestBatchLimitValidation(t *testing.T) {
	store := &analyticsStoreStub{}
	svc := NewService(store, Config{MaxBatchSize: 100})

	events := make([]BatchEvent, 0, 101)
	for i := 0; i < 101; i++ {
		events = append(events, BatchEvent{Name: "evt", TS: 1})
	}

	err := svc.IngestBatch(context.Background(), nil, events)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestIngestBatchSavesRows(t *testing.T) {
	store := &analyticsStoreStub{}
	svc := NewService(store, Config{MaxBatchSize: 100})
	fixedNow := time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	uid := int64(42)
	err := svc.IngestBatch(context.Background(), &uid, []BatchEvent{
		{Name: "feed_open", TS: 1_700_000_000, Props: map[string]any{"tab": "feed"}},
		{Name: "like_click", TS: 1_700_000_000_500, Props: map[string]any{"target_id": 1001}},
		{Name: "app_background", TS: 0, Props: nil},
	})
	if err != nil {
		t.Fatalf("ingest batch: %v", err)
	}

	if store.userID == nil || *store.userID != uid {
		t.Fatalf("unexpected user id in store: %+v", store.userID)
	}
	if len(store.events) != 3 {
		t.Fatalf("unexpected event rows count: got %d want 3", len(store.events))
	}
	if store.events[0].OccurredAt.Unix() != 1_700_000_000 {
		t.Fatalf("unexpected seconds ts conversion: %v", store.events[0].OccurredAt)
	}
	if store.events[1].OccurredAt.UnixMilli() != 1_700_000_000_500 {
		t.Fatalf("unexpected milliseconds ts conversion: %v", store.events[1].OccurredAt)
	}
	if !store.events[2].OccurredAt.Equal(fixedNow) {
		t.Fatalf("unexpected fallback ts: got %v want %v", store.events[2].OccurredAt, fixedNow)
	}
}

func TestIngestBatchObservesAntiAbuseEvents(t *testing.T) {
	store := &analyticsStoreStub{}
	observer := &antiAbuseObserverStub{}
	svc := NewService(store, Config{MaxBatchSize: 100})
	svc.AttachAntiAbuseDashboard(observer)

	uid := int64(10)
	err := svc.IngestBatch(context.Background(), &uid, []BatchEvent{
		{Name: "antiabuse_too_fast", TS: 1},
		{Name: "feed_open", TS: 1},
		{Name: "antiabuse_shadow_enabled", TS: 1},
	})
	if err != nil {
		t.Fatalf("ingest batch: %v", err)
	}

	if len(observer.names) != 3 {
		t.Fatalf("expected observer call per event, got %d", len(observer.names))
	}
	if observer.names[0] != "antiabuse_too_fast" || observer.names[2] != "antiabuse_shadow_enabled" {
		t.Fatalf("unexpected observed events: %+v", observer.names)
	}
}
