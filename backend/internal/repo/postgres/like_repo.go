package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoIncomingLikes = errors.New("no incoming likes")

type LikeRepo struct {
	pool *pgxpool.Pool
}

func NewLikeRepo(pool *pgxpool.Pool) *LikeRepo {
	return &LikeRepo{pool: pool}
}

type IncomingLikeRecord struct {
	FromUserID  int64
	DisplayName string
	Age         int
	CityID      string
	City        string
	LikedAt     time.Time
}

func (r *LikeRepo) Upsert(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64, isSuperLike bool) error {
	if fromUserID <= 0 || toUserID <= 0 {
		return fmt.Errorf("invalid like payload")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO likes (
	from_user_id,
	to_user_id,
	is_super_like,
	created_at
) VALUES ($1, $2, $3, NOW())
ON CONFLICT (from_user_id, to_user_id) DO UPDATE SET
	is_super_like = likes.is_super_like OR EXCLUDED.is_super_like,
	created_at = NOW()
`, fromUserID, toUserID, isSuperLike); err != nil {
		return fmt.Errorf("upsert like: %w", err)
	}

	return nil
}

func (r *LikeRepo) Exists(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64) (bool, error) {
	if fromUserID <= 0 || toUserID <= 0 {
		return false, fmt.Errorf("invalid like lookup payload")
	}
	if tx == nil {
		return false, fmt.Errorf("transaction is required")
	}

	var one int
	err := tx.QueryRow(ctx, `
SELECT 1
FROM likes
WHERE from_user_id = $1 AND to_user_id = $2
LIMIT 1
`, fromUserID, toUserID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("lookup like: %w", err)
	}

	return true, nil
}

func (r *LikeRepo) Delete(ctx context.Context, tx pgx.Tx, fromUserID, toUserID int64) (bool, error) {
	if fromUserID <= 0 || toUserID <= 0 {
		return false, fmt.Errorf("invalid like delete payload")
	}
	if tx == nil {
		return false, fmt.Errorf("transaction is required")
	}

	result, err := tx.Exec(ctx, `
DELETE FROM likes
WHERE from_user_id = $1 AND to_user_id = $2
`, fromUserID, toUserID)
	if err != nil {
		return false, fmt.Errorf("delete like: %w", err)
	}

	return result.RowsAffected() > 0, nil
}

func (r *LikeRepo) CountIncomingVisible(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id")
	}
	if r.pool == nil {
		return 0, nil
	}

	var count int
	if err := r.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM likes l
JOIN profiles p ON p.user_id = l.from_user_id
WHERE
	l.to_user_id = $1
	AND p.approved = TRUE
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE b.actor_user_id = l.from_user_id
			AND b.target_user_id = $1
	)
`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count incoming likes: %w", err)
	}

	return count, nil
}

func (r *LikeRepo) ListIncomingProfiles(ctx context.Context, userID int64, limit int) ([]IncomingLikeRecord, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if limit <= 0 {
		limit = 50
	}
	if r.pool == nil {
		return []IncomingLikeRecord{}, nil
	}

	rows, err := r.pool.Query(ctx, `
SELECT
	l.from_user_id,
	COALESCE(p.display_name, ''),
	COALESCE(DATE_PART('year', AGE(NOW(), p.birthdate::timestamp))::int, 0),
	COALESCE(p.city_id, ''),
	COALESCE(p.city, ''),
	l.created_at
FROM likes l
JOIN profiles p ON p.user_id = l.from_user_id
WHERE
	l.to_user_id = $1
	AND p.approved = TRUE
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE b.actor_user_id = l.from_user_id
			AND b.target_user_id = $1
	)
ORDER BY l.created_at DESC, l.id DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list incoming likes: %w", err)
	}
	defer rows.Close()

	items := make([]IncomingLikeRecord, 0, limit)
	for rows.Next() {
		var rec IncomingLikeRecord
		if err := rows.Scan(
			&rec.FromUserID,
			&rec.DisplayName,
			&rec.Age,
			&rec.CityID,
			&rec.City,
			&rec.LikedAt,
		); err != nil {
			return nil, fmt.Errorf("scan incoming like: %w", err)
		}
		items = append(items, rec)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate incoming likes: %w", rows.Err())
	}

	return items, nil
}

func (r *LikeRepo) NextIncomingUnrevealed(ctx context.Context, tx pgx.Tx, userID int64) (IncomingLikeRecord, error) {
	if userID <= 0 {
		return IncomingLikeRecord{}, fmt.Errorf("invalid user id")
	}
	if tx == nil {
		return IncomingLikeRecord{}, fmt.Errorf("transaction is required")
	}

	var rec IncomingLikeRecord
	err := tx.QueryRow(ctx, `
SELECT
	l.from_user_id,
	COALESCE(p.display_name, ''),
	COALESCE(DATE_PART('year', AGE(NOW(), p.birthdate::timestamp))::int, 0),
	COALESCE(p.city_id, ''),
	COALESCE(p.city, ''),
	l.created_at
FROM likes l
JOIN profiles p ON p.user_id = l.from_user_id
WHERE
	l.to_user_id = $1
	AND p.approved = TRUE
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE b.actor_user_id = l.from_user_id
			AND b.target_user_id = $1
	)
	AND NOT EXISTS (
		SELECT 1
		FROM likes_reveals lr
		WHERE lr.user_id = $1
			AND lr.liker_user_id = l.from_user_id
	)
ORDER BY l.created_at DESC, l.id DESC
LIMIT 1
`, userID).Scan(
		&rec.FromUserID,
		&rec.DisplayName,
		&rec.Age,
		&rec.CityID,
		&rec.City,
		&rec.LikedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IncomingLikeRecord{}, ErrNoIncomingLikes
		}
		return IncomingLikeRecord{}, fmt.Errorf("get next unrevealed incoming like: %w", err)
	}

	return rec, nil
}

func (r *LikeRepo) MarkRevealed(ctx context.Context, tx pgx.Tx, userID, likerUserID int64) error {
	if userID <= 0 || likerUserID <= 0 {
		return fmt.Errorf("invalid reveal payload")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO likes_reveals (
	user_id,
	liker_user_id,
	created_at
) VALUES ($1, $2, NOW())
ON CONFLICT (user_id, liker_user_id) DO NOTHING
`, userID, likerUserID); err != nil {
		return fmt.Errorf("mark incoming like as revealed: %w", err)
	}

	return nil
}
