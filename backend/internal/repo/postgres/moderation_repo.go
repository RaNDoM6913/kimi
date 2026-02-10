package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrModerationItemNotFound = errors.New("moderation item not found")

type ModerationRepo struct {
	pool *pgxpool.Pool
}

type ModerationItemRecord struct {
	ID              int64
	UserID          int64
	Status          string
	ReasonCode      *string
	ReasonText      *string
	RequiredFixStep *string
	ETABucket       string
	LockedByTGID    *int64
	LockedUntil     *time.Time
	LockedAt        *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewModerationRepo(pool *pgxpool.Pool) *ModerationRepo {
	return &ModerationRepo{pool: pool}
}

func (r *ModerationRepo) CreatePendingForMedia(ctx context.Context, userID, mediaID int64) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 || mediaID <= 0 {
		return fmt.Errorf("invalid moderation payload")
	}

	if _, err := r.pool.Exec(ctx, `
INSERT INTO moderation_items (
	user_id,
	target_type,
	target_id,
	status,
	eta_bucket,
	created_at,
	updated_at
) VALUES ($1, 'media', $2, 'PENDING', 'up_to_10', NOW(), NOW())
`, userID, mediaID); err != nil {
		return fmt.Errorf("create moderation item: %w", err)
	}

	return nil
}

func (r *ModerationRepo) GetLatestByUser(ctx context.Context, userID int64) (ModerationItemRecord, error) {
	if r.pool == nil {
		return ModerationItemRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 {
		return ModerationItemRecord{}, fmt.Errorf("invalid user id")
	}

	item, err := r.queryOne(ctx, `
SELECT id, user_id, status, reason_text, required_fix_step, eta_bucket, created_at, updated_at
FROM moderation_items
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1
`, userID)
	if err != nil {
		return ModerationItemRecord{}, err
	}

	return item, nil
}

func (r *ModerationRepo) CountPending(ctx context.Context) (int, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("postgres pool is nil")
	}

	var count int
	if err := r.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM moderation_items
WHERE UPPER(status) = 'PENDING'
`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending moderation items: %w", err)
	}

	return count, nil
}

func (r *ModerationRepo) AcquireNextPending(ctx context.Context, actorTGID int64, lockDuration time.Duration) (ModerationItemRecord, error) {
	if r.pool == nil {
		return ModerationItemRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if actorTGID == 0 {
		return ModerationItemRecord{}, fmt.Errorf("invalid actor tg id")
	}
	if lockDuration <= 0 {
		lockDuration = 10 * time.Minute
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ModerationItemRecord{}, fmt.Errorf("begin acquire transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	seconds := int64(lockDuration / time.Second)
	if seconds <= 0 {
		seconds = int64((10 * time.Minute) / time.Second)
	}

	item := ModerationItemRecord{}
	err = tx.QueryRow(ctx, `
WITH candidate AS (
	SELECT id
	FROM moderation_items
	WHERE UPPER(status) = 'PENDING'
	  AND (locked_until IS NULL OR locked_until < NOW())
	ORDER BY created_at ASC, id ASC
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
UPDATE moderation_items mi
SET
	locked_by_tg_id = $1,
	locked_at = NOW(),
	locked_until = NOW() + make_interval(secs => $2),
	updated_at = NOW()
FROM candidate
WHERE mi.id = candidate.id
RETURNING mi.id, mi.user_id, mi.status, mi.reason_text, mi.required_fix_step, mi.eta_bucket, mi.locked_by_tg_id, mi.locked_until, mi.locked_at, mi.created_at, mi.updated_at
`, actorTGID, seconds).Scan(
		&item.ID,
		&item.UserID,
		&item.Status,
		&item.ReasonText,
		&item.RequiredFixStep,
		&item.ETABucket,
		&item.LockedByTGID,
		&item.LockedUntil,
		&item.LockedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationItemRecord{}, ErrModerationItemNotFound
		}
		return ModerationItemRecord{}, fmt.Errorf("acquire next pending moderation item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ModerationItemRecord{}, fmt.Errorf("commit acquire transaction: %w", err)
	}

	return item, nil
}

func (r *ModerationRepo) GetNextPending(ctx context.Context) (ModerationItemRecord, error) {
	if r.pool == nil {
		return ModerationItemRecord{}, fmt.Errorf("postgres pool is nil")
	}

	item, err := r.queryOne(ctx, `
SELECT id, user_id, status, reason_text, required_fix_step, eta_bucket, created_at, updated_at
FROM moderation_items
WHERE UPPER(status) = 'PENDING'
  AND (locked_until IS NULL OR locked_until < NOW())
ORDER BY created_at ASC, id ASC
LIMIT 1
`)
	if err != nil {
		return ModerationItemRecord{}, err
	}

	return item, nil
}

func (r *ModerationRepo) GetByID(ctx context.Context, itemID int64) (ModerationItemRecord, error) {
	if r.pool == nil {
		return ModerationItemRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if itemID <= 0 {
		return ModerationItemRecord{}, fmt.Errorf("invalid moderation item id")
	}

	item, err := r.queryOne(ctx, `
SELECT id, user_id, status, reason_text, required_fix_step, eta_bucket, created_at, updated_at
FROM moderation_items
WHERE id = $1
LIMIT 1
`, itemID)
	if err != nil {
		return ModerationItemRecord{}, err
	}

	return item, nil
}

func (r *ModerationRepo) UpdateETABucket(ctx context.Context, itemID int64, etaBucket string) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if itemID <= 0 || strings.TrimSpace(etaBucket) == "" {
		return fmt.Errorf("invalid eta bucket payload")
	}

	if _, err := r.pool.Exec(ctx, `
UPDATE moderation_items
SET eta_bucket = $2, updated_at = NOW()
WHERE id = $1
`, itemID, etaBucket); err != nil {
		return fmt.Errorf("update moderation eta bucket: %w", err)
	}

	return nil
}

func (r *ModerationRepo) MarkApproved(ctx context.Context, itemID int64, moderatorTGID int64, etaBucket string) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if itemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}

	if _, err := r.pool.Exec(ctx, `
UPDATE moderation_items
SET
	status = 'APPROVED',
	moderator_tg_id = $2,
	reason_code = NULL,
	reason_text = NULL,
	required_fix_step = NULL,
	locked_by_tg_id = NULL,
	locked_at = NULL,
	locked_until = NULL,
	decided_at = NOW(),
	eta_bucket = COALESCE(NULLIF($3, ''), eta_bucket),
	updated_at = NOW()
WHERE id = $1
`, itemID, moderatorTGID, etaBucket); err != nil {
		return fmt.Errorf("mark moderation approved: %w", err)
	}

	return nil
}

func (r *ModerationRepo) MarkRejected(
	ctx context.Context,
	itemID int64,
	moderatorTGID int64,
	reasonCode, reasonText, requiredFixStep, etaBucket string,
) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if itemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}

	if _, err := r.pool.Exec(ctx, `
UPDATE moderation_items
SET
	status = 'REJECTED',
	moderator_tg_id = $2,
	reason_code = NULLIF($3, ''),
	reason_text = $4,
	required_fix_step = $5,
	locked_by_tg_id = NULL,
	locked_at = NULL,
	locked_until = NULL,
	decided_at = NOW(),
	eta_bucket = COALESCE(NULLIF($6, ''), eta_bucket),
	updated_at = NOW()
WHERE id = $1
`, itemID, moderatorTGID, strings.TrimSpace(reasonCode), strings.TrimSpace(reasonText), strings.TrimSpace(requiredFixStep), etaBucket); err != nil {
		return fmt.Errorf("mark moderation rejected: %w", err)
	}

	return nil
}

func (r *ModerationRepo) DeleteByMediaID(ctx context.Context, mediaID int64) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if mediaID <= 0 {
		return nil
	}

	if _, err := r.pool.Exec(ctx, `
DELETE FROM moderation_items
WHERE target_type = 'media' AND target_id = $1
`, mediaID); err != nil {
		return fmt.Errorf("delete moderation item by media id: %w", err)
	}

	return nil
}

func (r *ModerationRepo) queryOne(ctx context.Context, query string, args ...interface{}) (ModerationItemRecord, error) {
	var item ModerationItemRecord
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.UserID,
		&item.Status,
		&item.ReasonText,
		&item.RequiredFixStep,
		&item.ETABucket,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationItemRecord{}, ErrModerationItemNotFound
		}
		return ModerationItemRecord{}, fmt.Errorf("query moderation item: %w", err)
	}
	return item, nil
}
