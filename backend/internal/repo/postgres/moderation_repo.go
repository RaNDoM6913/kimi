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
	ReasonText      *string
	RequiredFixStep *string
	ETABucket       string
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

func (r *ModerationRepo) GetNextPending(ctx context.Context) (ModerationItemRecord, error) {
	if r.pool == nil {
		return ModerationItemRecord{}, fmt.Errorf("postgres pool is nil")
	}

	item, err := r.queryOne(ctx, `
SELECT id, user_id, status, reason_text, required_fix_step, eta_bucket, created_at, updated_at
FROM moderation_items
WHERE UPPER(status) = 'PENDING'
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
	reason_text = NULL,
	required_fix_step = NULL,
	eta_bucket = COALESCE(NULLIF($3, ''), eta_bucket),
	updated_at = NOW()
WHERE id = $1
`, itemID, moderatorTGID, etaBucket); err != nil {
		return fmt.Errorf("mark moderation approved: %w", err)
	}

	return nil
}

func (r *ModerationRepo) MarkRejected(ctx context.Context, itemID int64, moderatorTGID int64, reasonText, requiredFixStep, etaBucket string) error {
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
	reason_text = $3,
	required_fix_step = $4,
	eta_bucket = COALESCE(NULLIF($5, ''), eta_bucket),
	updated_at = NOW()
WHERE id = $1
`, itemID, moderatorTGID, strings.TrimSpace(reasonText), strings.TrimSpace(requiredFixStep), etaBucket); err != nil {
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
