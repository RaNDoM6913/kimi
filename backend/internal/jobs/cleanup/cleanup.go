package cleanup

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
)

type Job struct {
	mediaRepo      *pgrepo.MediaRepo
	moderationRepo *pgrepo.ModerationRepo
	storage        *mediasvc.S3Storage
	retention      time.Duration
	geoCleaner     exactGeoCleaner
	exactRetention time.Duration
	now            func() time.Time
	logger         *zap.Logger
}

type exactGeoCleaner interface {
	ClearExactGeoOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

func New() *Job {
	return &Job{
		retention:      365 * 24 * time.Hour,
		exactRetention: 48 * time.Hour,
		now:            time.Now,
		logger:         zap.NewNop(),
	}
}

func NewCircleCleanupJob(
	mediaRepo *pgrepo.MediaRepo,
	moderationRepo *pgrepo.ModerationRepo,
	storage *mediasvc.S3Storage,
	retention time.Duration,
	logger *zap.Logger,
) *Job {
	if retention <= 0 {
		retention = 365 * 24 * time.Hour
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Job{
		mediaRepo:      mediaRepo,
		moderationRepo: moderationRepo,
		storage:        storage,
		retention:      retention,
		exactRetention: 48 * time.Hour,
		now:            time.Now,
		logger:         logger,
	}
}

func (j *Job) AttachExactGeoCleanup(cleaner exactGeoCleaner, retention time.Duration) {
	j.geoCleaner = cleaner
	if retention > 0 {
		j.exactRetention = retention
	}
}

func (j *Job) Run(ctx context.Context) error {
	if j.geoCleaner != nil && j.exactRetention > 0 {
		exactCutoff := j.now().Add(-j.exactRetention)
		rows, err := j.geoCleaner.ClearExactGeoOlderThan(ctx, exactCutoff)
		if err != nil {
			return fmt.Errorf("cleanup exact geo coordinates: %w", err)
		}
		if rows > 0 {
			j.logger.Info("cleanup exact geo coordinates completed", zap.Int64("cleared", rows))
		}
	}

	if j.mediaRepo == nil || j.moderationRepo == nil || j.storage == nil {
		return nil
	}

	cutoff := j.now().Add(-j.retention)
	circles, err := j.mediaRepo.ListCirclesOlderThan(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list stale circles: %w", err)
	}

	if len(circles) == 0 {
		return nil
	}

	for _, circle := range circles {
		if err := j.storage.Delete(ctx, circle.ObjectKey); err != nil {
			j.logger.Warn("failed to delete circle object from storage", zap.Error(err), zap.String("object_key", circle.ObjectKey))
		}
		if err := j.moderationRepo.DeleteByMediaID(ctx, circle.ID); err != nil {
			return fmt.Errorf("delete moderation item by media id: %w", err)
		}
		if err := j.mediaRepo.DeleteCircle(ctx, circle.ID); err != nil {
			return fmt.Errorf("delete circle media: %w", err)
		}
	}

	j.logger.Info("cleanup stale circles completed", zap.Int("deleted", len(circles)))
	return nil
}
