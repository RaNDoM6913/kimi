package dualrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/adminhttp"
)

const (
	ModeDual = "dual"
	ModeHTTP = "http"
	ModeDB   = "db"
)

type ModerationRepo interface {
	AcquireNextPending(context.Context, int64, time.Duration) (model.ModerationItem, error)
	GetProfile(context.Context, int64) (model.ModerationProfile, error)
	ListPhotoKeys(context.Context, int64, int) ([]string, error)
	GetLatestCircleKey(context.Context, int64) (string, error)
	GetByID(context.Context, int64) (model.ModerationItem, error)
	MarkApproved(context.Context, int64) error
	MarkRejected(context.Context, int64, string, string, string) error
	InsertModerationAction(context.Context, model.BotModerationAction) error
}

type DualRepo struct {
	httpRepo ModerationRepo
	dbRepo   ModerationRepo
	mode     string
}

func NewModerationRepo(httpRepo ModerationRepo, dbRepo ModerationRepo, mode string) *DualRepo {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	switch normalizedMode {
	case ModeDB, ModeHTTP, ModeDual:
	default:
		normalizedMode = ModeDual
	}
	return &DualRepo{
		httpRepo: httpRepo,
		dbRepo:   dbRepo,
		mode:     normalizedMode,
	}
}

func (r *DualRepo) AcquireNextPending(ctx context.Context, actorTGID int64, lockDuration time.Duration) (model.ModerationItem, error) {
	return callWithFallback(
		r,
		func(repo ModerationRepo) (model.ModerationItem, error) {
			return repo.AcquireNextPending(ctx, actorTGID, lockDuration)
		},
		func(repo ModerationRepo) (model.ModerationItem, error) {
			return repo.AcquireNextPending(ctx, actorTGID, lockDuration)
		},
	)
}

func (r *DualRepo) GetProfile(ctx context.Context, userID int64) (model.ModerationProfile, error) {
	return callWithFallback(
		r,
		func(repo ModerationRepo) (model.ModerationProfile, error) {
			return repo.GetProfile(ctx, userID)
		},
		func(repo ModerationRepo) (model.ModerationProfile, error) {
			return repo.GetProfile(ctx, userID)
		},
	)
}

func (r *DualRepo) ListPhotoKeys(ctx context.Context, userID int64, limit int) ([]string, error) {
	return callWithFallback(
		r,
		func(repo ModerationRepo) ([]string, error) {
			return repo.ListPhotoKeys(ctx, userID, limit)
		},
		func(repo ModerationRepo) ([]string, error) {
			return repo.ListPhotoKeys(ctx, userID, limit)
		},
	)
}

func (r *DualRepo) GetLatestCircleKey(ctx context.Context, userID int64) (string, error) {
	return callWithFallback(
		r,
		func(repo ModerationRepo) (string, error) {
			return repo.GetLatestCircleKey(ctx, userID)
		},
		func(repo ModerationRepo) (string, error) {
			return repo.GetLatestCircleKey(ctx, userID)
		},
	)
}

func (r *DualRepo) GetByID(ctx context.Context, moderationItemID int64) (model.ModerationItem, error) {
	return callWithFallback(
		r,
		func(repo ModerationRepo) (model.ModerationItem, error) {
			return repo.GetByID(ctx, moderationItemID)
		},
		func(repo ModerationRepo) (model.ModerationItem, error) {
			return repo.GetByID(ctx, moderationItemID)
		},
	)
}

func (r *DualRepo) MarkApproved(ctx context.Context, moderationItemID int64) error {
	return callWithFallbackErr(
		r,
		func(repo ModerationRepo) error {
			return repo.MarkApproved(ctx, moderationItemID)
		},
		func(repo ModerationRepo) error {
			return repo.MarkApproved(ctx, moderationItemID)
		},
	)
}

func (r *DualRepo) MarkRejected(ctx context.Context, moderationItemID int64, reasonCode string, reasonText string, requiredFixStep string) error {
	return callWithFallbackErr(
		r,
		func(repo ModerationRepo) error {
			return repo.MarkRejected(ctx, moderationItemID, reasonCode, reasonText, requiredFixStep)
		},
		func(repo ModerationRepo) error {
			return repo.MarkRejected(ctx, moderationItemID, reasonCode, reasonText, requiredFixStep)
		},
	)
}

func (r *DualRepo) InsertModerationAction(ctx context.Context, action model.BotModerationAction) error {
	return callWithFallbackErr(
		r,
		func(repo ModerationRepo) error {
			return repo.InsertModerationAction(ctx, action)
		},
		func(repo ModerationRepo) error {
			return repo.InsertModerationAction(ctx, action)
		},
	)
}

func callWithFallback[T any](
	dualRepo *DualRepo,
	httpCall func(ModerationRepo) (T, error),
	dbCall func(ModerationRepo) (T, error),
) (T, error) {
	var zero T
	if dualRepo == nil {
		return zero, errors.New("dual moderation repo is nil")
	}

	switch dualRepo.mode {
	case ModeDB:
		if dualRepo.dbRepo == nil {
			return zero, errors.New("db moderation repo is not configured")
		}
		return dbCall(dualRepo.dbRepo)
	case ModeHTTP:
		if dualRepo.httpRepo == nil {
			return zero, errors.New("http moderation repo is not configured")
		}
		return httpCall(dualRepo.httpRepo)
	default:
		if dualRepo.httpRepo == nil {
			if dualRepo.dbRepo == nil {
				return zero, errors.New("moderation repos are not configured")
			}
			return dbCall(dualRepo.dbRepo)
		}

		value, err := httpCall(dualRepo.httpRepo)
		if err == nil {
			return value, nil
		}
		if dualRepo.dbRepo == nil {
			return zero, err
		}
		if !adminhttp.IsFallbackable(err) {
			return zero, err
		}
		dbValue, dbErr := dbCall(dualRepo.dbRepo)
		if dbErr != nil {
			return zero, fmt.Errorf("http err: %v; db fallback err: %w", err, dbErr)
		}
		return dbValue, nil
	}
}

func callWithFallbackErr(
	dualRepo *DualRepo,
	httpCall func(ModerationRepo) error,
	dbCall func(ModerationRepo) error,
) error {
	if dualRepo == nil {
		return errors.New("dual moderation repo is nil")
	}

	switch dualRepo.mode {
	case ModeDB:
		if dualRepo.dbRepo == nil {
			return errors.New("db moderation repo is not configured")
		}
		return dbCall(dualRepo.dbRepo)
	case ModeHTTP:
		if dualRepo.httpRepo == nil {
			return errors.New("http moderation repo is not configured")
		}
		return httpCall(dualRepo.httpRepo)
	default:
		if dualRepo.httpRepo == nil {
			if dualRepo.dbRepo == nil {
				return errors.New("moderation repos are not configured")
			}
			return dbCall(dualRepo.dbRepo)
		}

		err := httpCall(dualRepo.httpRepo)
		if err == nil {
			return nil
		}
		if dualRepo.dbRepo == nil {
			return err
		}
		if !adminhttp.IsFallbackable(err) {
			return err
		}
		if dbErr := dbCall(dualRepo.dbRepo); dbErr != nil {
			return fmt.Errorf("http err: %v; db fallback err: %w", err, dbErr)
		}
		return nil
	}
}
