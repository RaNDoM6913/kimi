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

type EntitlementRepo struct {
	pool *pgxpool.Pool
}

type EntitlementSnapshotRecord struct {
	UserID                int64
	PlusExpiresAt         *time.Time
	BoostUntil            *time.Time
	SuperLikeCredits      int
	RevealCredits         int
	MessageWoMatchCredits int
	LikeTokens            int
	IncognitoUntil        *time.Time
}

func NewEntitlementRepo(pool *pgxpool.Pool) *EntitlementRepo {
	return &EntitlementRepo{pool: pool}
}

func (r *EntitlementRepo) IsPlusActive(ctx context.Context, userID int64, at time.Time) (bool, *time.Time, error) {
	if userID <= 0 {
		return false, nil, fmt.Errorf("invalid user id")
	}
	if r.pool == nil {
		return false, nil, nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	var plusUntil *time.Time
	err := r.pool.QueryRow(ctx, `
SELECT plus_expires_at
FROM entitlements
WHERE user_id = $1
LIMIT 1
`, userID).Scan(&plusUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("get entitlement plus status: %w", err)
	}

	if plusUntil == nil || !plusUntil.After(at.UTC()) {
		return false, plusUntil, nil
	}

	return true, plusUntil, nil
}

func (r *EntitlementRepo) GetSnapshot(ctx context.Context, userID int64) (EntitlementSnapshotRecord, error) {
	if userID <= 0 {
		return EntitlementSnapshotRecord{}, fmt.Errorf("invalid user id")
	}
	if r.pool == nil {
		return EntitlementSnapshotRecord{UserID: userID}, nil
	}

	var snapshot EntitlementSnapshotRecord
	err := r.pool.QueryRow(ctx, `
SELECT
	user_id,
	plus_expires_at,
	boost_until,
	superlike_credits,
	reveal_credits,
	message_wo_match_credits,
	like_tokens,
	incognito_until
FROM entitlements
WHERE user_id = $1
LIMIT 1
`, userID).Scan(
		&snapshot.UserID,
		&snapshot.PlusExpiresAt,
		&snapshot.BoostUntil,
		&snapshot.SuperLikeCredits,
		&snapshot.RevealCredits,
		&snapshot.MessageWoMatchCredits,
		&snapshot.LikeTokens,
		&snapshot.IncognitoUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EntitlementSnapshotRecord{UserID: userID}, nil
		}
		return EntitlementSnapshotRecord{}, fmt.Errorf("get entitlement snapshot: %w", err)
	}

	return snapshot, nil
}

var ErrInsufficientSuperLikeResources = errors.New("insufficient superlike resources")
var ErrInsufficientRevealCredits = errors.New("insufficient reveal credits")

func (r *EntitlementRepo) ConsumeSuperLike(ctx context.Context, tx pgx.Tx, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO entitlements (
	user_id,
	superlike_credits,
	reveal_credits,
	like_tokens,
	message_wo_match_credits,
	updated_at
) VALUES ($1, 0, 0, 0, 0, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure entitlements row: %w", err)
	}

	result, err := tx.Exec(ctx, `
UPDATE entitlements
SET
	superlike_credits = superlike_credits - 1,
	like_tokens = like_tokens - 1,
	updated_at = NOW()
WHERE
	user_id = $1
	AND superlike_credits >= 1
	AND like_tokens >= 1
`, userID)
	if err != nil {
		return fmt.Errorf("consume superlike resources: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInsufficientSuperLikeResources
	}

	return nil
}

func (r *EntitlementRepo) RefundSuperLike(ctx context.Context, tx pgx.Tx, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO entitlements (
	user_id,
	superlike_credits,
	reveal_credits,
	like_tokens,
	message_wo_match_credits,
	updated_at
) VALUES ($1, 1, 0, 1, 0, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	superlike_credits = entitlements.superlike_credits + 1,
	like_tokens = entitlements.like_tokens + 1,
	updated_at = NOW()
`, userID); err != nil {
		return fmt.Errorf("refund superlike resources: %w", err)
	}

	return nil
}

func (r *EntitlementRepo) ConsumeRevealCredit(ctx context.Context, tx pgx.Tx, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO entitlements (
	user_id,
	superlike_credits,
	reveal_credits,
	like_tokens,
	message_wo_match_credits,
	updated_at
) VALUES ($1, 0, 0, 0, 0, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure entitlements row for reveal: %w", err)
	}

	result, err := tx.Exec(ctx, `
UPDATE entitlements
SET
	reveal_credits = reveal_credits - 1,
	updated_at = NOW()
WHERE
	user_id = $1
	AND reveal_credits >= 1
`, userID)
	if err != nil {
		return fmt.Errorf("consume reveal credit: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInsufficientRevealCredits
	}

	return nil
}

func (r *EntitlementRepo) ApplyPurchaseSKU(ctx context.Context, userID int64, sku string, now time.Time) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	normalizedSKU := strings.ToLower(strings.TrimSpace(sku))
	if normalizedSKU == "" {
		return fmt.Errorf("sku is required")
	}

	if _, err := r.pool.Exec(ctx, `
INSERT INTO entitlements (
	user_id,
	superlike_credits,
	reveal_credits,
	like_tokens,
	message_wo_match_credits,
	updated_at
) VALUES ($1, 0, 0, 0, 0, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure entitlements row: %w", err)
	}

	switch normalizedSKU {
	case "boost_30m":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	boost_until = CASE
		WHEN boost_until IS NOT NULL AND boost_until > $2::timestamptz
			THEN boost_until + INTERVAL '30 minutes'
		ELSE $2::timestamptz + INTERVAL '30 minutes'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC())
		if err != nil {
			return fmt.Errorf("apply boost entitlement: %w", err)
		}
		return nil
	case "superlike_pack_3":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	superlike_credits = superlike_credits + 3,
	updated_at = NOW()
WHERE user_id = $1
`, userID)
		if err != nil {
			return fmt.Errorf("apply superlike entitlement: %w", err)
		}
		return nil
	case "reveal_1":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	reveal_credits = reveal_credits + 1,
	updated_at = NOW()
WHERE user_id = $1
`, userID)
		if err != nil {
			return fmt.Errorf("apply reveal entitlement: %w", err)
		}
		return nil
	case "incognito_24h":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	incognito_until = CASE
		WHEN incognito_until IS NOT NULL AND incognito_until > $2::timestamptz
			THEN incognito_until + INTERVAL '24 hours'
		ELSE $2::timestamptz + INTERVAL '24 hours'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC())
		if err != nil {
			return fmt.Errorf("apply incognito entitlement: %w", err)
		}
		return nil
	case "message_wo_match_1":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	message_wo_match_credits = message_wo_match_credits + 1,
	updated_at = NOW()
WHERE user_id = $1
`, userID)
		if err != nil {
			return fmt.Errorf("apply message without match entitlement: %w", err)
		}
		return nil
	case "plus_1m":
		_, err := r.pool.Exec(ctx, `
UPDATE entitlements
SET
	plus_expires_at = CASE
		WHEN plus_expires_at IS NOT NULL AND plus_expires_at > $2::timestamptz
			THEN plus_expires_at + INTERVAL '30 days'
		ELSE $2::timestamptz + INTERVAL '30 days'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC())
		if err != nil {
			return fmt.Errorf("apply plus entitlement: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported purchase sku: %s", normalizedSKU)
	}
}
