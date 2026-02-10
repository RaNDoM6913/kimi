package analytics

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

const defaultMaxBatchSize = 100

var ErrValidation = errors.New("validation error")

type Store interface {
	InsertBatch(ctx context.Context, userID *int64, events []pgrepo.EventWriteRecord) error
}

type AntiAbuseDashboardObserver interface {
	ObserveEvent(ctx context.Context, userID *int64, name string, props map[string]any) error
}

type Config struct {
	MaxBatchSize int
}

type Service struct {
	store              Store
	antiAbuseDashboard AntiAbuseDashboardObserver
	cfg                Config
	now                func() time.Time
}

type BatchEvent struct {
	Name  string
	TS    int64
	Props map[string]any
}

func NewService(store Store, cfg Config) *Service {
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = defaultMaxBatchSize
	}

	return &Service{
		store: store,
		cfg:   cfg,
		now:   time.Now,
	}
}

func (s *Service) AttachAntiAbuseDashboard(observer AntiAbuseDashboardObserver) {
	s.antiAbuseDashboard = observer
}

func (s *Service) IngestBatch(ctx context.Context, userID *int64, events []BatchEvent) error {
	if s.store == nil {
		return fmt.Errorf("analytics store is nil")
	}
	if len(events) == 0 || len(events) > s.cfg.MaxBatchSize {
		return ErrValidation
	}

	now := s.now().UTC()
	rows := make([]pgrepo.EventWriteRecord, 0, len(events))
	for _, event := range events {
		name := strings.TrimSpace(event.Name)
		if name == "" {
			return ErrValidation
		}

		rows = append(rows, pgrepo.EventWriteRecord{
			Name:       name,
			OccurredAt: parseTS(event.TS, now),
			Props:      cloneProps(event.Props),
		})

		if s.antiAbuseDashboard != nil {
			if err := s.antiAbuseDashboard.ObserveEvent(ctx, userID, name, event.Props); err != nil {
				log.Printf("warning: observe antiabuse dashboard event failed: %v", err)
			}
		}
	}

	if err := s.store.InsertBatch(ctx, userID, rows); err != nil {
		return fmt.Errorf("insert events batch: %w", err)
	}

	return nil
}

func parseTS(ts int64, fallback time.Time) time.Time {
	if ts <= 0 {
		return fallback
	}
	if ts >= 1_000_000_000_000 {
		return time.UnixMilli(ts).UTC()
	}
	return time.Unix(ts, 0).UTC()
}

func cloneProps(props map[string]any) map[string]any {
	if len(props) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(props))
	for key, value := range props {
		out[key] = value
	}
	return out
}
