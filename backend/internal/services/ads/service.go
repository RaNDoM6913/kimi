package ads

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	EventTypeImpression = "IMPRESSION"
	EventTypeClick      = "CLICK"
)

var (
	ErrValidation = errors.New("validation error")
	ErrAdNotFound = errors.New("ad not found")
)

type Store interface {
	ExistsActive(ctx context.Context, adID int64, at time.Time) (bool, error)
	InsertEvent(ctx context.Context, adID, userID int64, eventType string, meta map[string]any) error
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func (s *Service) Impression(ctx context.Context, userID, adID int64, meta map[string]any) error {
	return s.record(ctx, userID, adID, EventTypeImpression, meta)
}

func (s *Service) Click(ctx context.Context, userID, adID int64, meta map[string]any) error {
	return s.record(ctx, userID, adID, EventTypeClick, meta)
}

func (s *Service) record(ctx context.Context, userID, adID int64, eventType string, meta map[string]any) error {
	if userID <= 0 || adID <= 0 || strings.TrimSpace(eventType) == "" {
		return ErrValidation
	}
	if s.store == nil {
		return fmt.Errorf("ads store is nil")
	}

	now := s.now().UTC()
	exists, err := s.store.ExistsActive(ctx, adID, now)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAdNotFound
	}

	return s.store.InsertEvent(ctx, adID, userID, eventType, meta)
}
