package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockRepo struct {
	pool *pgxpool.Pool
}

func NewBlockRepo(pool *pgxpool.Pool) *BlockRepo {
	return &BlockRepo{pool: pool}
}

func (r *BlockRepo) Upsert(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, reason string) error {
	if actorUserID <= 0 || targetUserID <= 0 || actorUserID == targetUserID {
		return fmt.Errorf("invalid block payload")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO blocks (
	actor_user_id,
	target_user_id,
	reason,
	created_at
) VALUES ($1, $2, $3, NOW())
ON CONFLICT (actor_user_id, target_user_id) DO UPDATE SET
	reason = EXCLUDED.reason
`, actorUserID, targetUserID, strings.TrimSpace(reason)); err != nil {
		return fmt.Errorf("upsert block: %w", err)
	}

	return nil
}
