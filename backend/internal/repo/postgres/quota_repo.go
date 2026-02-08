package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuotaRepo struct {
	pool *pgxpool.Pool
}

func NewQuotaRepo(pool *pgxpool.Pool) *QuotaRepo {
	return &QuotaRepo{pool: pool}
}

func (r *QuotaRepo) GetLikesUsed(ctx context.Context, userID int64, dayKey string) (int, error) {
	if userID <= 0 || strings.TrimSpace(dayKey) == "" {
		return 0, fmt.Errorf("invalid quota lookup payload")
	}
	if r.pool == nil {
		return 0, nil
	}

	var likesUsed int
	err := r.pool.QueryRow(ctx, `
SELECT likes_used
FROM quotas_daily
WHERE user_id = $1 AND day_key = $2::date
LIMIT 1
`, userID, dayKey).Scan(&likesUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get daily quota usage: %w", err)
	}

	return likesUsed, nil
}

func (r *QuotaRepo) IncrementLikesUsed(ctx context.Context, userID int64, dayKey, timezone string, delta int) (int, error) {
	if userID <= 0 || strings.TrimSpace(dayKey) == "" {
		return 0, fmt.Errorf("invalid quota update payload")
	}
	if delta <= 0 {
		delta = 1
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}
	if r.pool == nil {
		return delta, nil
	}

	var likesUsed int
	err := r.pool.QueryRow(ctx, `
INSERT INTO quotas_daily (
	user_id,
	day_key,
	tz_name,
	likes_used,
	updated_at
) VALUES ($1, $2::date, $3, $4, NOW())
ON CONFLICT (user_id, day_key) DO UPDATE SET
	likes_used = quotas_daily.likes_used + EXCLUDED.likes_used,
	tz_name = EXCLUDED.tz_name,
	updated_at = NOW()
RETURNING likes_used
`, userID, dayKey, timezone, delta).Scan(&likesUsed)
	if err != nil {
		return 0, fmt.Errorf("increment daily quota usage: %w", err)
	}

	return likesUsed, nil
}

var (
	ErrLikesLimitReached  = errors.New("likes daily limit reached")
	ErrRewindLimitReached = errors.New("rewind daily limit reached")
)

func (r *QuotaRepo) ConsumeLikeWithLimit(ctx context.Context, tx pgx.Tx, userID int64, dayKey, timezone string, limit int) (int, error) {
	if userID <= 0 || strings.TrimSpace(dayKey) == "" || limit <= 0 {
		return 0, fmt.Errorf("invalid like quota consume payload")
	}
	if tx == nil {
		return 0, fmt.Errorf("transaction is required")
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}

	var likesUsed int
	err := tx.QueryRow(ctx, `
INSERT INTO quotas_daily (
	user_id,
	day_key,
	tz_name,
	likes_used,
	rewind_used,
	updated_at
) VALUES ($1, $2::date, $3, 1, 0, NOW())
ON CONFLICT (user_id, day_key) DO UPDATE SET
	likes_used = quotas_daily.likes_used + 1,
	tz_name = EXCLUDED.tz_name,
	updated_at = NOW()
WHERE quotas_daily.likes_used < $4
RETURNING likes_used
`, userID, dayKey, timezone, limit).Scan(&likesUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrLikesLimitReached
		}
		return 0, fmt.Errorf("consume likes quota with limit: %w", err)
	}

	return likesUsed, nil
}

func (r *QuotaRepo) RefundLike(ctx context.Context, tx pgx.Tx, userID int64, dayKey string) error {
	if userID <= 0 || strings.TrimSpace(dayKey) == "" {
		return fmt.Errorf("invalid like quota refund payload")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
UPDATE quotas_daily
SET
	likes_used = GREATEST(likes_used - 1, 0),
	updated_at = NOW()
WHERE user_id = $1 AND day_key = $2::date
`, userID, dayKey); err != nil {
		return fmt.Errorf("refund likes quota: %w", err)
	}

	return nil
}

func (r *QuotaRepo) ConsumeRewindWithLimit(ctx context.Context, tx pgx.Tx, userID int64, dayKey, timezone string, limit int) (int, error) {
	if userID <= 0 || strings.TrimSpace(dayKey) == "" || limit <= 0 {
		return 0, fmt.Errorf("invalid rewind quota consume payload")
	}
	if tx == nil {
		return 0, fmt.Errorf("transaction is required")
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}

	var rewindUsed int
	err := tx.QueryRow(ctx, `
INSERT INTO quotas_daily (
	user_id,
	day_key,
	tz_name,
	likes_used,
	rewind_used,
	updated_at
) VALUES ($1, $2::date, $3, 0, 1, NOW())
ON CONFLICT (user_id, day_key) DO UPDATE SET
	rewind_used = quotas_daily.rewind_used + 1,
	tz_name = EXCLUDED.tz_name,
	updated_at = NOW()
WHERE quotas_daily.rewind_used < $4
RETURNING rewind_used
`, userID, dayKey, timezone, limit).Scan(&rewindUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRewindLimitReached
		}
		return 0, fmt.Errorf("consume rewind quota with limit: %w", err)
	}

	return rewindUsed, nil
}
