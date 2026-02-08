package likes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivankudzin/tgapp/backend/internal/domain/rules"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	ratesvc "github.com/ivankudzin/tgapp/backend/internal/services/rate"
)

var (
	ErrValidation      = errors.New("validation error")
	ErrDailyLimit      = errors.New("daily likes limit reached")
	ErrRateLimited     = errors.New("too fast")
	ErrDependenciesNil = errors.New("likes dependencies are not configured")
	ErrNothingToReveal = errors.New("nothing to reveal")
	ErrRevealRequired  = errors.New("reveal credit required")
)

type TooFastError struct {
	RetryAfterSec int64
}

func (e TooFastError) Error() string {
	return "too fast"
}

func (e TooFastError) RetryAfter() int64 {
	if e.RetryAfterSec <= 0 {
		return 1
	}
	return e.RetryAfterSec
}

func IsTooFast(err error) (*TooFastError, bool) {
	var tf TooFastError
	if errors.As(err, &tf) {
		return &tf, true
	}
	return nil, false
}

type QuotaStore interface {
	GetLikesUsed(ctx context.Context, userID int64, dayKey string) (int, error)
	IncrementLikesUsed(ctx context.Context, userID int64, dayKey, timezone string, delta int) (int, error)
}

type EntitlementStore interface {
	IsPlusActive(ctx context.Context, userID int64, at time.Time) (bool, *time.Time, error)
}

type IncomingStore interface {
	CountIncomingVisible(ctx context.Context, userID int64) (int, error)
	ListIncomingProfiles(ctx context.Context, userID int64, limit int) ([]pgrepo.IncomingLikeRecord, error)
	NextIncomingUnrevealed(ctx context.Context, tx pgx.Tx, userID int64) (pgrepo.IncomingLikeRecord, error)
	MarkRevealed(ctx context.Context, tx pgx.Tx, userID, likerUserID int64) error
}

type RevealCreditStore interface {
	ConsumeRevealCredit(ctx context.Context, tx pgx.Tx, userID int64) error
}

type Config struct {
	FreeLikesPerDay int
	DefaultTimezone string
	DefaultIsPlus   bool
	PlusUnlimitedUI bool
}

type Snapshot struct {
	LikesLeft         int
	ResetAt           time.Time
	TooFastRetryAfter *int64
	IsPlus            bool
}

type Service struct {
	quotaStore  QuotaStore
	plusStore   EntitlementStore
	rateLimiter *ratesvc.Limiter
	pool        *pgxpool.Pool
	incoming    IncomingStore
	reveal      RevealCreditStore
	cfg         Config
	now         func() time.Time
}

type IncomingPreview struct {
	UserID  int64
	LikedAt time.Time
}

type IncomingProfile struct {
	UserID      int64
	DisplayName string
	Age         int
	CityID      string
	City        string
	LikedAt     time.Time
}

type IncomingResult struct {
	Blurred    bool
	TotalCount int
	Preview    []IncomingPreview
	Profiles   []IncomingProfile
}

func NewService(quotaStore QuotaStore, plusStore EntitlementStore, rateLimiter *ratesvc.Limiter, cfg Config) *Service {
	if cfg.FreeLikesPerDay <= 0 {
		cfg.FreeLikesPerDay = rules.FreeLikesPerDay
	}
	if strings.TrimSpace(cfg.DefaultTimezone) == "" {
		cfg.DefaultTimezone = "UTC"
	}

	return &Service{
		quotaStore:  quotaStore,
		plusStore:   plusStore,
		rateLimiter: rateLimiter,
		cfg:         cfg,
		now:         time.Now,
	}
}

func (s *Service) AttachIncoming(pool *pgxpool.Pool, incoming IncomingStore, reveal RevealCreditStore) {
	s.pool = pool
	s.incoming = incoming
	s.reveal = reveal
}

func (s *Service) GetSnapshot(ctx context.Context, userID int64, timezone string) (Snapshot, error) {
	if userID <= 0 {
		return Snapshot{}, ErrValidation
	}
	if s.quotaStore == nil {
		return Snapshot{}, ErrDependenciesNil
	}

	now := s.now().UTC()
	loc, _ := s.resolveTimezone(timezone)
	dayKey := rules.DayKey(now, loc)
	resetAt := rules.NextResetAt(now, loc)

	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := Snapshot{
		LikesLeft: s.cfg.FreeLikesPerDay,
		ResetAt:   resetAt,
		IsPlus:    isPlus,
	}

	if isPlus && s.cfg.PlusUnlimitedUI {
		snapshot.LikesLeft = -1
	} else {
		used, err := s.quotaStore.GetLikesUsed(ctx, userID, dayKey)
		if err != nil {
			return Snapshot{}, fmt.Errorf("read daily quota: %w", err)
		}
		left := s.cfg.FreeLikesPerDay - used
		if left < 0 {
			left = 0
		}
		snapshot.LikesLeft = left
	}

	if isPlus && s.rateLimiter != nil {
		retryAfter, err := s.rateLimiter.RetryAfterLike(ctx, userID)
		if err != nil {
			return Snapshot{}, fmt.Errorf("read plus rate limiter state: %w", err)
		}
		if retryAfter > 0 {
			v := retryAfter
			snapshot.TooFastRetryAfter = &v
		}
	}
	return snapshot, nil
}

func (s *Service) ConsumeLike(ctx context.Context, userID int64, timezone string) (Snapshot, error) {
	if userID <= 0 {
		return Snapshot{}, ErrValidation
	}
	if s.quotaStore == nil {
		return Snapshot{}, ErrDependenciesNil
	}

	now := s.now().UTC()
	loc, tzName := s.resolveTimezone(timezone)
	dayKey := rules.DayKey(now, loc)
	resetAt := rules.NextResetAt(now, loc)

	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return Snapshot{}, err
	}

	if isPlus && s.rateLimiter != nil {
		retryAfter, allowed, err := s.rateLimiter.AllowLike(ctx, userID)
		if err != nil {
			return Snapshot{}, fmt.Errorf("consume plus rate limit: %w", err)
		}
		if !allowed {
			return Snapshot{}, TooFastError{RetryAfterSec: retryAfter}
		}
		return Snapshot{
			LikesLeft: -1,
			ResetAt:   resetAt,
			IsPlus:    true,
		}, nil
	}

	used, err := s.quotaStore.GetLikesUsed(ctx, userID, dayKey)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read daily quota: %w", err)
	}
	if used >= s.cfg.FreeLikesPerDay {
		return Snapshot{}, ErrDailyLimit
	}

	updatedUsed, err := s.quotaStore.IncrementLikesUsed(ctx, userID, dayKey, tzName, 1)
	if err != nil {
		return Snapshot{}, fmt.Errorf("update daily quota: %w", err)
	}

	left := s.cfg.FreeLikesPerDay - updatedUsed
	if left < 0 {
		left = 0
	}

	return Snapshot{
		LikesLeft: left,
		ResetAt:   resetAt,
		IsPlus:    false,
	}, nil
}

func (s *Service) GetIncoming(ctx context.Context, userID int64) (IncomingResult, error) {
	if userID <= 0 {
		return IncomingResult{}, ErrValidation
	}
	if s.incoming == nil {
		return IncomingResult{}, ErrDependenciesNil
	}

	now := s.now().UTC()
	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return IncomingResult{}, err
	}

	count, err := s.incoming.CountIncomingVisible(ctx, userID)
	if err != nil {
		return IncomingResult{}, fmt.Errorf("count incoming likes: %w", err)
	}

	previewRows, err := s.incoming.ListIncomingProfiles(ctx, userID, 3)
	if err != nil {
		return IncomingResult{}, fmt.Errorf("load incoming preview: %w", err)
	}
	preview := make([]IncomingPreview, 0, len(previewRows))
	for _, row := range previewRows {
		preview = append(preview, IncomingPreview{
			UserID:  row.FromUserID,
			LikedAt: row.LikedAt,
		})
	}

	result := IncomingResult{
		Blurred:    !isPlus,
		TotalCount: count,
		Preview:    preview,
	}

	if !isPlus {
		return result, nil
	}

	rows, err := s.incoming.ListIncomingProfiles(ctx, userID, 100)
	if err != nil {
		return IncomingResult{}, fmt.Errorf("load incoming likes: %w", err)
	}
	result.Profiles = make([]IncomingProfile, 0, len(rows))
	for _, row := range rows {
		result.Profiles = append(result.Profiles, mapIncomingRow(row))
	}

	return result, nil
}

func (s *Service) RevealOne(ctx context.Context, userID int64) (IncomingProfile, error) {
	if userID <= 0 {
		return IncomingProfile{}, ErrValidation
	}
	if s.pool == nil || s.incoming == nil || s.reveal == nil {
		return IncomingProfile{}, ErrDependenciesNil
	}

	var profile IncomingProfile
	err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		next, err := s.incoming.NextIncomingUnrevealed(txCtx, tx, userID)
		if err != nil {
			if errors.Is(err, pgrepo.ErrNoIncomingLikes) {
				return ErrNothingToReveal
			}
			return err
		}

		if err := s.reveal.ConsumeRevealCredit(txCtx, tx, userID); err != nil {
			if errors.Is(err, pgrepo.ErrInsufficientRevealCredits) {
				return ErrRevealRequired
			}
			return err
		}

		if err := s.incoming.MarkRevealed(txCtx, tx, userID, next.FromUserID); err != nil {
			return err
		}

		profile = mapIncomingRow(next)
		return nil
	})
	if err != nil {
		return IncomingProfile{}, err
	}

	return profile, nil
}

func (s *Service) resolvePlus(ctx context.Context, userID int64, at time.Time) (bool, error) {
	if s.plusStore == nil {
		return s.cfg.DefaultIsPlus, nil
	}

	isPlus, _, err := s.plusStore.IsPlusActive(ctx, userID, at)
	if err != nil {
		return false, fmt.Errorf("resolve plus entitlement: %w", err)
	}
	if isPlus {
		return true, nil
	}

	return s.cfg.DefaultIsPlus, nil
}

func (s *Service) resolveTimezone(explicit string) (*time.Location, string) {
	candidate := strings.TrimSpace(explicit)
	if candidate == "" {
		candidate = strings.TrimSpace(s.cfg.DefaultTimezone)
	}
	if candidate == "" {
		candidate = "UTC"
	}

	loc, err := time.LoadLocation(candidate)
	if err != nil {
		return time.UTC, "UTC"
	}
	return loc, candidate
}

func mapIncomingRow(row pgrepo.IncomingLikeRecord) IncomingProfile {
	return IncomingProfile{
		UserID:      row.FromUserID,
		DisplayName: row.DisplayName,
		Age:         row.Age,
		CityID:      row.CityID,
		City:        row.City,
		LikedAt:     row.LikedAt,
	}
}
