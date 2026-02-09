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
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
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

type TooFastError struct {
	RetryAfterSec int64
	CooldownUntil *time.Time
	Reason        string
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

type CooldownActiveError struct {
	RetryAfterSec int64
	CooldownUntil *time.Time
}

func (e CooldownActiveError) Error() string {
	return "cooldown active"
}

func (e CooldownActiveError) RetryAfter() int64 {
	if e.RetryAfterSec <= 0 {
		return 1
	}
	return e.RetryAfterSec
}

func IsCooldownActive(err error) (*CooldownActiveError, bool) {
	var cd CooldownActiveError
	if errors.As(err, &cd) {
		return &cd, true
	}
	return nil, false
}

type SwipeStore interface {
	Create(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, action string, now time.Time) (pgrepo.SwipeRecord, error)
	GetLastByActor(ctx context.Context, tx pgx.Tx, actorUserID int64) (pgrepo.SwipeRecord, error)
	DeleteByID(ctx context.Context, tx pgx.Tx, swipeID int64) error
	ApplyDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) (pgrepo.DislikeStateRecord, error)
	UndoDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) error
}

type LikeStore interface {
	Upsert(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64, isSuperLike, isSuspect bool) error
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
	CheckLikeRate(ctx context.Context, userID int64, sid, ip string) (allowed bool, retryAfterSec int, reason string)
}

type QuotaSnapshotProvider interface {
	GetSnapshot(ctx context.Context, userID int64, timezone string) (likessvc.Snapshot, error)
}

type AntiAbuseService interface {
	ApplyDecay(ctx context.Context, userID int64, now time.Time) (antiabusesvc.State, error)
	ApplyViolation(ctx context.Context, userID int64, weight int, now time.Time) (antiabusesvc.State, error)
}

type TelemetryService interface {
	IngestBatch(ctx context.Context, userID *int64, events []analyticsvc.BatchEvent) error
}

type SwipeClientTelemetry struct {
	CardViewMS    int
	SwipeVelocity *float64
	Screen        string
}

type Config struct {
	FreeLikesPerDay      int
	FreeRewindsPerDay    int
	PlusRewindsPerDay    int
	DefaultTimezone      string
	DefaultIsPlus        bool
	MinCardViewMS        int
	SuspectLikeThreshold int
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
	antiAbuse    AntiAbuseService
	telemetry    TelemetryService
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
	AntiAbuse    AntiAbuseService
	Telemetry    TelemetryService
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
	if cfg.MinCardViewMS <= 0 {
		cfg.MinCardViewMS = 700
	}
	if cfg.SuspectLikeThreshold <= 0 {
		cfg.SuspectLikeThreshold = 8
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
		antiAbuse:    deps.AntiAbuse,
		telemetry:    deps.Telemetry,
		cfg:          cfg,
		now:          time.Now,
	}
}

func (s *Service) Swipe(ctx context.Context, userID, targetID int64, action, timezone, sid, ip string, client SwipeClientTelemetry) (SwipeResult, error) {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return SwipeResult{}, ErrValidation
	}

	normalizedAction, err := normalizeAction(action)
	if err != nil {
		return SwipeResult{}, err
	}

	now := s.now().UTC()
	gateState := antiabusesvc.State{}
	if normalizedAction == actionLike || normalizedAction == actionSuperLike {
		var err error
		gateState, err = s.applyLikeGates(ctx, userID, sid, ip, now)
		if err != nil {
			return SwipeResult{}, err
		}
	}
	isSuspectLike := (normalizedAction == actionLike || normalizedAction == actionSuperLike) && s.shouldMarkLikeAsSuspect(gateState)

	if s.pool == nil || s.swipeStore == nil || s.likeStore == nil || s.matchStore == nil || s.quotaStore == nil || s.entitlements == nil {
		return SwipeResult{}, fmt.Errorf("swipe dependencies are not configured")
	}

	loc, tzName := s.resolveTimezone(timezone)
	dayKey := rules.DayKey(now, loc)

	isPlus, err := s.resolvePlus(ctx, userID, now)
	if err != nil {
		return SwipeResult{}, err
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
			if err := s.likeStore.Upsert(txCtx, tx, userID, targetID, normalizedAction == actionSuperLike, isSuspectLike); err != nil {
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

	s.logSuspectLikeEvent(ctx, userID, targetID, normalizedAction, isSuspectLike, gateState.RiskScore, now)
	s.applyLowCardViewViolation(ctx, userID, targetID, normalizedAction, client, now)

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

func (s *Service) applyLowCardViewViolation(ctx context.Context, userID, targetID int64, action string, client SwipeClientTelemetry, now time.Time) {
	if action != actionLike && action != actionSuperLike {
		return
	}
	if client.CardViewMS <= 0 || client.CardViewMS >= s.cfg.MinCardViewMS {
		return
	}

	if s.antiAbuse != nil {
		_, _ = s.antiAbuse.ApplyViolation(ctx, userID, 1, now)
	}

	if s.telemetry != nil {
		props := map[string]any{
			"action":           action,
			"target_id":        targetID,
			"card_view_ms":     client.CardViewMS,
			"min_card_view_ms": s.cfg.MinCardViewMS,
		}
		if client.SwipeVelocity != nil {
			props["swipe_velocity"] = *client.SwipeVelocity
		}
		if screen := strings.TrimSpace(client.Screen); screen != "" {
			props["screen"] = screen
		}

		userIDCopy := userID
		_ = s.telemetry.IngestBatch(ctx, &userIDCopy, []analyticsvc.BatchEvent{
			{
				Name:  "antiabuse_low_card_view",
				TS:    now.UnixMilli(),
				Props: props,
			},
		})
	}
}

func (s *Service) applyLikeGates(ctx context.Context, userID int64, sid, ip string, now time.Time) (antiabusesvc.State, error) {
	state := antiabusesvc.State{}
	if s.antiAbuse != nil {
		decayedState, err := s.antiAbuse.ApplyDecay(ctx, userID, now)
		if err == nil {
			state = decayedState
		}
		if err == nil && decayedState.CooldownUntil != nil && now.Before(*decayedState.CooldownUntil) {
			return decayedState, CooldownActiveError{
				RetryAfterSec: secondsUntil(*decayedState.CooldownUntil, now),
				CooldownUntil: decayedState.CooldownUntil,
			}
		}
	}

	if s.rateLimiter == nil {
		return state, nil
	}

	allowed, retryAfter, reason := s.rateLimiter.CheckLikeRate(ctx, userID, sid, ip)
	if allowed {
		return state, nil
	}

	var cooldownUntil *time.Time
	if s.antiAbuse != nil {
		if violatedState, err := s.antiAbuse.ApplyViolation(ctx, userID, 2, now); err == nil {
			state = violatedState
			cooldownUntil = violatedState.CooldownUntil
		}
	}
	s.logTooFastAndCooldownEvents(ctx, userID, sid, ip, reason, retryAfter, state, now)

	return state, TooFastError{
		RetryAfterSec: int64(retryAfter),
		CooldownUntil: cooldownUntil,
		Reason:        reason,
	}
}

func (s *Service) shouldMarkLikeAsSuspect(state antiabusesvc.State) bool {
	if s.cfg.SuspectLikeThreshold <= 0 {
		return false
	}
	return state.RiskScore >= s.cfg.SuspectLikeThreshold
}

func (s *Service) logSuspectLikeEvent(ctx context.Context, userID, targetID int64, action string, isSuspectLike bool, riskScore int, now time.Time) {
	if !isSuspectLike || s.telemetry == nil || userID <= 0 {
		return
	}
	if action != actionLike && action != actionSuperLike {
		return
	}

	props := map[string]any{
		"action":                 action,
		"target_id":              targetID,
		"risk_score":             riskScore,
		"suspect_like_threshold": s.cfg.SuspectLikeThreshold,
	}

	uid := userID
	_ = s.telemetry.IngestBatch(ctx, &uid, []analyticsvc.BatchEvent{
		{
			Name:  "antiabuse_suspect_like",
			TS:    now.UTC().UnixMilli(),
			Props: props,
		},
	})
}

func (s *Service) logTooFastAndCooldownEvents(
	ctx context.Context,
	userID int64,
	sid, ip, reason string,
	retryAfter int,
	state antiabusesvc.State,
	now time.Time,
) {
	if s.telemetry == nil || userID <= 0 {
		return
	}

	retry := int64(retryAfter)
	if retry <= 0 {
		retry = 1
	}
	cooldownUntilTS := int64(0)
	if state.CooldownUntil != nil {
		cooldownUntilTS = state.CooldownUntil.UTC().Unix()
	}

	events := []analyticsvc.BatchEvent{
		{
			Name: "antiabuse_too_fast",
			TS:   now.UTC().UnixMilli(),
			Props: map[string]any{
				"reason":          strings.TrimSpace(reason),
				"retry_after_sec": retry,
				"has_sid":         strings.TrimSpace(sid) != "",
				"has_ip":          strings.TrimSpace(ip) != "",
				"risk_score":      state.RiskScore,
			},
		},
		{
			Name: "antiabuse_cooldown_applied",
			TS:   now.UTC().UnixMilli(),
			Props: map[string]any{
				"retry_after_sec":   retry,
				"cooldown_until_ts": cooldownUntilTS,
				"risk_score":        state.RiskScore,
			},
		},
	}

	uid := userID
	_ = s.telemetry.IngestBatch(ctx, &uid, events)
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

func secondsUntil(target, now time.Time) int64 {
	diff := target.Sub(now)
	if diff <= 0 {
		return 1
	}
	sec := int64(diff / time.Second)
	if diff%time.Second != 0 {
		sec++
	}
	if sec <= 0 {
		sec = 1
	}
	return sec
}
