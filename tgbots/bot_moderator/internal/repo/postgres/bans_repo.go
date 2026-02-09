package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type BansRepo struct {
	db *sql.DB
}

func NewBansRepo(db *sql.DB) *BansRepo {
	return &BansRepo{db: db}
}

func (r *BansRepo) SetBan(ctx context.Context, userID int64, banned bool, reason string, updatedByTGID int64) error {
	if r.db == nil {
		return nil
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if updatedByTGID == 0 {
		return fmt.Errorf("invalid updated_by_tg_id")
	}

	uuid := pseudoUUID(userID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_bans (user_id, banned, reason, updated_by_tg_id, updated_at)
		VALUES ($1::uuid, $2, NULLIF($3, ''), $4, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			banned = EXCLUDED.banned,
			reason = EXCLUDED.reason,
			updated_by_tg_id = EXCLUDED.updated_by_tg_id,
			updated_at = EXCLUDED.updated_at
	`, uuid, banned, strings.TrimSpace(reason), updatedByTGID)
	if err != nil {
		return fmt.Errorf("upsert user ban: %w", err)
	}
	return nil
}

func (r *BansRepo) GetBanState(ctx context.Context, userID int64) (bool, string, error) {
	if r.db == nil {
		return false, "", nil
	}
	if userID <= 0 {
		return false, "", fmt.Errorf("invalid user id")
	}

	var banned bool
	var reason sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT banned, reason
		FROM user_bans
		WHERE user_id = $1::uuid
		LIMIT 1
	`, pseudoUUID(userID)).Scan(&banned, &reason)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("get user ban state: %w", err)
	}

	return banned, strings.TrimSpace(reason.String), nil
}

func (r *BansRepo) Ban(ctx context.Context, userID int64, reason string, updatedByTGID int64) error {
	return r.SetBan(ctx, userID, true, reason, updatedByTGID)
}

func (r *BansRepo) Unban(ctx context.Context, userID int64, updatedByTGID int64) error {
	return r.SetBan(ctx, userID, false, "", updatedByTGID)
}
