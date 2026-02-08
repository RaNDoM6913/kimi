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

var ErrSwipeNotFound = errors.New("swipe not found")

type SwipeRepo struct {
	pool *pgxpool.Pool
}

func NewSwipeRepo(pool *pgxpool.Pool) *SwipeRepo {
	return &SwipeRepo{pool: pool}
}

type SwipeRecord struct {
	ID           int64
	ActorUserID  int64
	TargetUserID int64
	Action       string
	CreatedAt    time.Time
}

type DislikeStateRecord struct {
	ActorUserID  int64
	TargetUserID int64
	DislikeCount int
	HideUntil    *time.Time
	NeverShow    bool
}

func (r *SwipeRepo) Create(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, action string, now time.Time) (SwipeRecord, error) {
	if actorUserID <= 0 || targetUserID <= 0 || strings.TrimSpace(action) == "" {
		return SwipeRecord{}, fmt.Errorf("invalid swipe payload")
	}
	if tx == nil {
		return SwipeRecord{}, fmt.Errorf("transaction is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var rec SwipeRecord
	err := tx.QueryRow(ctx, `
INSERT INTO swipes (
	actor_user_id,
	target_user_id,
	action,
	created_at
) VALUES ($1, $2, $3, $4)
RETURNING id, actor_user_id, target_user_id, action, created_at
`, actorUserID, targetUserID, strings.ToUpper(strings.TrimSpace(action)), now.UTC()).Scan(
		&rec.ID,
		&rec.ActorUserID,
		&rec.TargetUserID,
		&rec.Action,
		&rec.CreatedAt,
	)
	if err != nil {
		return SwipeRecord{}, fmt.Errorf("create swipe: %w", err)
	}

	return rec, nil
}

func (r *SwipeRepo) GetLastByActor(ctx context.Context, tx pgx.Tx, actorUserID int64) (SwipeRecord, error) {
	if actorUserID <= 0 {
		return SwipeRecord{}, fmt.Errorf("invalid actor user id")
	}
	if tx == nil {
		return SwipeRecord{}, fmt.Errorf("transaction is required")
	}

	var rec SwipeRecord
	err := tx.QueryRow(ctx, `
SELECT id, actor_user_id, target_user_id, action, created_at
FROM swipes
WHERE actor_user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1
`, actorUserID).Scan(
		&rec.ID,
		&rec.ActorUserID,
		&rec.TargetUserID,
		&rec.Action,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SwipeRecord{}, ErrSwipeNotFound
		}
		return SwipeRecord{}, fmt.Errorf("get last swipe by actor: %w", err)
	}

	return rec, nil
}

func (r *SwipeRepo) DeleteByID(ctx context.Context, tx pgx.Tx, swipeID int64) error {
	if swipeID <= 0 {
		return fmt.Errorf("invalid swipe id")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	result, err := tx.Exec(ctx, `
DELETE FROM swipes
WHERE id = $1
`, swipeID)
	if err != nil {
		return fmt.Errorf("delete swipe: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSwipeNotFound
	}
	return nil
}

func (r *SwipeRepo) ApplyDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) (DislikeStateRecord, error) {
	if actorUserID <= 0 || targetUserID <= 0 {
		return DislikeStateRecord{}, fmt.Errorf("invalid dislike payload")
	}
	if tx == nil {
		return DislikeStateRecord{}, fmt.Errorf("transaction is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	hideUntil := now.UTC().Add(24 * time.Hour)
	var state DislikeStateRecord
	err := tx.QueryRow(ctx, `
INSERT INTO dislikes_state (
	actor_user_id,
	target_user_id,
	dislike_count,
	hide_until,
	never_show,
	until_at,
	created_at,
	updated_at
) VALUES (
	$1,
	$2,
	1,
	$3,
	FALSE,
	$3,
	NOW(),
	NOW()
)
ON CONFLICT (actor_user_id, target_user_id) DO UPDATE SET
	dislike_count = dislikes_state.dislike_count + 1,
	hide_until = CASE
		WHEN dislikes_state.dislike_count + 1 >= 2 THEN NULL
		ELSE $3
	END,
	never_show = CASE
		WHEN dislikes_state.dislike_count + 1 >= 2 THEN TRUE
		ELSE FALSE
	END,
	until_at = CASE
		WHEN dislikes_state.dislike_count + 1 >= 2 THEN NULL
		ELSE $3
	END,
	updated_at = NOW()
RETURNING actor_user_id, target_user_id, dislike_count, hide_until, never_show
`, actorUserID, targetUserID, hideUntil).Scan(
		&state.ActorUserID,
		&state.TargetUserID,
		&state.DislikeCount,
		&state.HideUntil,
		&state.NeverShow,
	)
	if err != nil {
		return DislikeStateRecord{}, fmt.Errorf("apply dislike state: %w", err)
	}

	return state, nil
}

func (r *SwipeRepo) UndoDislike(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, now time.Time) error {
	if actorUserID <= 0 || targetUserID <= 0 {
		return fmt.Errorf("invalid dislike payload")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var count int
	err := tx.QueryRow(ctx, `
SELECT dislike_count
FROM dislikes_state
WHERE actor_user_id = $1 AND target_user_id = $2
FOR UPDATE
`, actorUserID, targetUserID).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get dislike state for undo: %w", err)
	}

	newCount := count - 1
	if newCount <= 0 {
		if _, err := tx.Exec(ctx, `
DELETE FROM dislikes_state
WHERE actor_user_id = $1 AND target_user_id = $2
`, actorUserID, targetUserID); err != nil {
			return fmt.Errorf("delete dislike state: %w", err)
		}
		return nil
	}

	if newCount == 1 {
		hideUntil := now.UTC().Add(24 * time.Hour)
		if _, err := tx.Exec(ctx, `
UPDATE dislikes_state
SET
	dislike_count = 1,
	hide_until = $3,
	until_at = $3,
	never_show = FALSE,
	updated_at = NOW()
WHERE actor_user_id = $1 AND target_user_id = $2
`, actorUserID, targetUserID, hideUntil); err != nil {
			return fmt.Errorf("downgrade dislike state to temporary hide: %w", err)
		}
		return nil
	}

	if _, err := tx.Exec(ctx, `
UPDATE dislikes_state
SET
	dislike_count = $3,
	hide_until = NULL,
	until_at = NULL,
	never_show = TRUE,
	updated_at = NOW()
WHERE actor_user_id = $1 AND target_user_id = $2
`, actorUserID, targetUserID, newCount); err != nil {
		return fmt.Errorf("decrement dislike state: %w", err)
	}

	return nil
}
