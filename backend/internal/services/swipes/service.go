package swipes

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
	likessvc "github.com/ivankudzin/tgapp/backend/internal/services/likes"
)

const (
	actionLike      = "LIKE"
	actionSuperLike = "SUPERLIKE"
	actionDislike   = "DISLIKE"
)

var (
	ErrValidation            = errors.New("validation error")
	ErrUnsupportedAction     = errors.New("unsupported action")
	ErrNoActionsToRewind     = errors.New("no actions to rewind")
	ErrSuperLikeRequirements = errors.New("superlike requirements are not met")
	ErrRewindLimitReached    = errors.New("rewind daily limit reached")
)

type SwipeStore interface {
	Create(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, action string, now time.Time) (pgrepo.SwipeRecord, error)
	GetLastByActor(ctx context.Context, tx pgx.Tx, actorUserID int64) (pgrepo.SwipeRecord, error)
	DeleteByID(ctx context.Context, tx pgx.Tx, swipeID int64) error
	ApplyDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) (pgrepo.DislikeStateRecord, error)
	UndoDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) error
}

type LikeStore interface {
	Upsert(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64, isSuperLike bool) error
	Delete(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64) (bool, error)
}

type MatchStore interface {
	CreateIfMutualLike(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error)
	DeleteByUsers(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error)
}

type QuotaStore interface {
	ConsumeLikeWithLimit(ctx context.Context, tx pgx.Tx, userID int64, dayKey, timezone string, limit int) (int, error)
	RefundLike(ctx context.Context, tx pgx.Tx, userID int64, dayKey string) error
	ConsumeRewindWithLimit(ctx context.Context, tx pgx.Tx, userID int64, dayKey, timezone string, limit int) (int, error)
}

type EntitlementStore interface {
	IsPlusActive(ctx context.Context, userID int64, at time.Time) (bool, *time.Time, error)
	ConsumeSuperLike(ctx context.Context, tx pgx.Tx, userID int64) error
	RefundSuperLike(ctx context.Context, tx pgx.Tx, userID int64) error
}

type RateLimiter interface {
	AllowLike(ctx context.Context, userID int64) (int64, bool, error)
}

type QuotaSnapshotProvider interface {
	GetSnapshot(ctx context.Context, userID int64, timezone string) (likessvc.Snapshot, error)
}

type Config struct {
	FreeLikesPerDay   int
	FreeRewindsPerDay int
	PlusRewindsPerDay int
	DefaultTimezone   string
	DefaultIsPlus     bool
}

type SwipeResult struct {
	MatchCreated bool
	Quota        likessvc.Snapshot
}

type RewindResult struct {
	UndoneAction   string
	UndoneTargetID int64
	Quota          likessvc.Snapshot
}

type Service struct {
	pool         *pgxpool.Pool
	swipeStore   SwipeStore
	likeStore    LikeStore
	matchStore   MatchStore
	quotaStore   QuotaStore
	entitlements EntitlementStore
	rateLimiter  RateLimiter
	quotaView    QuotaSnapshotProvider
	cfg          Config
	now          func() time.Time
}

type Dependencies struct {
	Pool         *pgxpool.Pool
	SwipeStore   SwipeStore
	LikeStore    LikeStore
	MatchStore   MatchStore
	QuotaStore   QuotaStore
	Entitlements EntitlementStore
	RateLimiter  RateLimiter
	QuotaView    QuotaSnapshotProvider
}

func NewService(deps Dependencies, cfg Config) *Service {
	if cfg.FreeLikesPerDay <= 0 {
		cfg.FreeLikesPerDay = rules.FreeLikesPerDay
	}
	if cfg.FreeRewindsPerDay <= 0 {
		cfg.FreeRewindsPerDay = 1
	}
	if cfg.PlusRewindsPerDay <= 0 {
		cfg.PlusRewindsPerDay = 3
	}
	if strings.TrimSpace(cfg.DefaultTimezone) == "" {
		cfg.DefaultTimezone = "UTC"
	}

	return &Service{
		pool:         deps.Pool,
		swipeStore:   deps.SwipeStore,
		likeStore:    deps.LikeStore,
		matchStore:   deps.MatchStore,
		quotaStore:   deps.QuotaStore,
		entitlements: deps.Entitlements,
		rateLimiter:  deps.RateLimiter,
		quotaView:    deps.QuotaView,
		cfg:          cfg,
		now:          time.Now,
	}
}

func (s *Service) Swipe(ctx context.Context, userID, targetID int64, action, timezone string) (SwipeResult, error) {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return SwipeResult{}, ErrValidation
	}

	normalizedAction, err := normalizeAction(action)
	if err != nil {
		return SwipeResult{}, err
	}

	if s.pool == nil || s.swipeStore == nil || s.likeStore == nil || s.matchStore == nil || s.quotaStore == nil || s.entitlements == nil {
		return SwipeResult{}, fmt.Errorf("swipe dependencies are not configured")
	}

	now := s.now().UTC()
	loc, tzName := s.resolveTimezone(timezone)
	dayKey := rules.DayKey(now, loc)

	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return SwipeResult{}, err
	}

	if (normalizedAction == actionLike || normalizedAction == actionSuperLike) && isPlus && s.rateLimiter != nil {
		retryAfter, allowed, err := s.rateLimiter.AllowLike(ctx, userID)
		if err != nil {
			return SwipeResult{}, fmt.Errorf("apply plus rate limiter: %w", err)
		}
		if !allowed {
			return SwipeResult{}, likessvc.TooFastError{RetryAfterSec: retryAfter}
		}
	}

	matchCreated := false
	if err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		switch normalizedAction {
		case actionLike, actionSuperLike:
			if normalizedAction == actionSuperLike {
				if err := s.entitlements.ConsumeSuperLike(txCtx, tx, userID); err != nil {
					if errors.Is(err, pgrepo.ErrInsufficientSuperLikeResources) {
						return ErrSuperLikeRequirements
					}
					return err
				}
			}

			if !isPlus {
				if _, err := s.quotaStore.ConsumeLikeWithLimit(txCtx, tx, userID, dayKey, tzName, s.cfg.FreeLikesPerDay); err != nil {
					if errors.Is(err, pgrepo.ErrLikesLimitReached) {
						return likessvc.ErrDailyLimit
					}
					return err
				}
			}

			if _, err := s.swipeStore.Create(txCtx, tx, userID, targetID, normalizedAction, now); err != nil {
				return err
			}
			if err := s.likeStore.Upsert(txCtx, tx, userID, targetID, normalizedAction == actionSuperLike); err != nil {
				return err
			}
			created, err := s.matchStore.CreateIfMutualLike(txCtx, tx, userID, targetID)
			if err != nil {
				return err
			}
			matchCreated = created
		case actionDislike:
			if _, err := s.swipeStore.Create(txCtx, tx, userID, targetID, normalizedAction, now); err != nil {
				return err
			}
			if _, err := s.swipeStore.ApplyDislike(txCtx, tx, userID, targetID, now); err != nil {
				return err
			}
		default:
			return ErrUnsupportedAction
		}
		return nil
	}); err != nil {
		return SwipeResult{}, err
	}

	snapshot, err := s.snapshot(ctx, userID, timezone)
	if err != nil {
		return SwipeResult{}, err
	}

	return SwipeResult{
		MatchCreated: matchCreated,
		Quota:        snapshot,
	}, nil
}

func (s *Service) Rewind(ctx context.Context, userID int64, timezone string) (RewindResult, error) {
	if userID <= 0 {
		return RewindResult{}, ErrValidation
	}
	if s.pool == nil || s.swipeStore == nil || s.likeStore == nil || s.matchStore == nil || s.quotaStore == nil || s.entitlements == nil {
		return RewindResult{}, fmt.Errorf("swipe dependencies are not configured")
	}

	now := s.now().UTC()
	loc, tzName := s.resolveTimezone(timezone)
	dayKey := rules.DayKey(now, loc)

	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return RewindResult{}, err
	}

	rewindLimit := s.cfg.FreeRewindsPerDay
	if isPlus {
		rewindLimit = s.cfg.PlusRewindsPerDay
	}

	var undone pgrepo.SwipeRecord
	if err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		if _, err := s.quotaStore.ConsumeRewindWithLimit(txCtx, tx, userID, dayKey, tzName, rewindLimit); err != nil {
			if errors.Is(err, pgrepo.ErrRewindLimitReached) {
				return ErrRewindLimitReached
			}
			return err
		}

		lastSwipe, err := s.swipeStore.GetLastByActor(txCtx, tx, userID)
		if err != nil {
			if errors.Is(err, pgrepo.ErrSwipeNotFound) {
				return ErrNoActionsToRewind
			}
			return err
		}
		undone = lastSwipe
		action := normalizeActionValue(lastSwipe.Action)
		actionDayKey := rules.DayKey(lastSwipe.CreatedAt.UTC(), loc)

		switch action {
		case actionLike:
			if _, err := s.likeStore.Delete(txCtx, tx, userID, lastSwipe.TargetUserID); err != nil {
				return err
			}
			if _, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, lastSwipe.TargetUserID); err != nil {
				return err
			}
			if !isPlus {
				if err := s.quotaStore.RefundLike(txCtx, tx, userID, actionDayKey); err != nil {
					return err
				}
			}
		case actionSuperLike:
			if _, err := s.likeStore.Delete(txCtx, tx, userID, lastSwipe.TargetUserID); err != nil {
				return err
			}
			if _, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, lastSwipe.TargetUserID); err != nil {
				return err
			}
			if err := s.entitlements.RefundSuperLike(txCtx, tx, userID); err != nil {
				return err
			}
			if !isPlus {
				if err := s.quotaStore.RefundLike(txCtx, tx, userID, actionDayKey); err != nil {
					return err
				}
			}
		case actionDislike:
			if err := s.swipeStore.UndoDislike(txCtx, tx, userID, lastSwipe.TargetUserID, now); err != nil {
				return err
			}
		default:
			return ErrUnsupportedAction
		}

		if err := s.swipeStore.DeleteByID(txCtx, tx, lastSwipe.ID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return RewindResult{}, err
	}

	snapshot, err := s.snapshot(ctx, userID, timezone)
	if err != nil {
		return RewindResult{}, err
	}

	return RewindResult{
		UndoneAction:   normalizeActionValue(undone.Action),
		UndoneTargetID: undone.TargetUserID,
		Quota:          snapshot,
	}, nil
}

func (s *Service) snapshot(ctx context.Context, userID int64, timezone string) (likessvc.Snapshot, error) {
	if s.quotaView == nil {
		return likessvc.Snapshot{}, nil
	}
	snapshot, err := s.quotaView.GetSnapshot(ctx, userID, timezone)
	if err != nil {
		return likessvc.Snapshot{}, fmt.Errorf("read quota snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Service) resolvePlus(ctx context.Context, userID int64, at time.Time) (bool, error) {
	isPlus, _, err := s.entitlements.IsPlusActive(ctx, userID, at)
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

func normalizeAction(input string) (string, error) {
	value := normalizeActionValue(input)
	switch value {
	case actionLike, actionSuperLike, actionDislike:
		return value, nil
	default:
		return "", ErrUnsupportedAction
	}
}

func normalizeActionValue(input string) string {
	value := strings.ToUpper(strings.TrimSpace(input))
	value = strings.ReplaceAll(value, "_", "")
	return value
}
