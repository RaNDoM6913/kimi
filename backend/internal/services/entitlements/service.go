package entitlements

import (
	"context"
	"errors"
	"fmt"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

var ErrValidation = errors.New("validation error")

type Store interface {
	GetSnapshot(ctx context.Context, userID int64) (pgrepo.EntitlementSnapshotRecord, error)
}

type Config struct {
	DefaultIsPlus bool
}

type Service struct {
	store Store
	cfg   Config
	now   func() time.Time
}

type Snapshot struct {
	UserID                int64
	IsPlus                bool
	PlusUntil             *time.Time
	BoostUntil            *time.Time
	SuperLikeCredits      int
	RevealCredits         int
	MessageWoMatchCredits int
	LikeTokens            int
	IncognitoUntil        *time.Time
}

func NewService(store Store, cfg Config) *Service {
	return &Service{
		store: store,
		cfg:   cfg,
		now:   time.Now,
	}
}

func (s *Service) Get(ctx context.Context, userID int64) (Snapshot, error) {
	if userID <= 0 {
		return Snapshot{}, ErrValidation
	}
	if s.store == nil {
		return Snapshot{}, fmt.Errorf("entitlement store is nil")
	}

	rec, err := s.store.GetSnapshot(ctx, userID)
	if err != nil {
		return Snapshot{}, err
	}

	now := s.now().UTC()
	isPlus := s.cfg.DefaultIsPlus
	if rec.PlusExpiresAt != nil {
		isPlus = rec.PlusExpiresAt.After(now)
	}

	return Snapshot{
		UserID:                userID,
		IsPlus:                isPlus,
		PlusUntil:             rec.PlusExpiresAt,
		BoostUntil:            rec.BoostUntil,
		SuperLikeCredits:      rec.SuperLikeCredits,
		RevealCredits:         rec.RevealCredits,
		MessageWoMatchCredits: rec.MessageWoMatchCredits,
		LikeTokens:            rec.LikeTokens,
		IncognitoUntil:        rec.IncognitoUntil,
	}, nil
}
