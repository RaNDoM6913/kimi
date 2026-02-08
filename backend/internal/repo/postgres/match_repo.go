package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MatchRepo struct {
	pool *pgxpool.Pool
}

type ActiveMatchRecord struct {
	ID           int64
	TargetUserID int64
	DisplayName  string
	Age          int
	CityID       string
	City         string
	CreatedAt    time.Time
}

func NewMatchRepo(pool *pgxpool.Pool) *MatchRepo {
	return &MatchRepo{pool: pool}
}

func (r *MatchRepo) CreateIfMutualLike(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error) {
	if userID <= 0 || targetID <= 0 {
		return false, fmt.Errorf("invalid match payload")
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
`, targetID, userID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("lookup reciprocal like: %w", err)
	}

	userA := userID
	userB := targetID
	if userA > userB {
		userA, userB = userB, userA
	}

	var matchID int64
	err = tx.QueryRow(ctx, `
INSERT INTO matches (
	user_a_id,
	user_b_id,
	status,
	created_at
) VALUES ($1, $2, 'active', NOW())
ON CONFLICT (user_a_id, user_b_id) DO NOTHING
RETURNING id
`, userA, userB).Scan(&matchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("create match: %w", err)
	}

	return matchID > 0, nil
}

func (r *MatchRepo) ListActiveForUser(ctx context.Context, userID int64, limit int) ([]ActiveMatchRecord, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if limit <= 0 {
		limit = 100
	}
	if r.pool == nil {
		return []ActiveMatchRecord{}, nil
	}

	rows, err := r.pool.Query(ctx, `
SELECT
	m.id,
	CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END AS target_user_id,
	COALESCE(p.display_name, ''),
	COALESCE(DATE_PART('year', AGE(NOW(), p.birthdate::timestamp))::int, 0),
	COALESCE(p.city_id, ''),
	COALESCE(p.city, ''),
	m.created_at
FROM matches m
JOIN profiles p ON p.user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
WHERE
	(m.user_a_id = $1 OR m.user_b_id = $1)
	AND COALESCE(m.status, 'active') = 'active'
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE b.actor_user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
			AND b.target_user_id = $1
	)
ORDER BY m.created_at DESC, m.id DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list active matches: %w", err)
	}
	defer rows.Close()

	items := make([]ActiveMatchRecord, 0, limit)
	for rows.Next() {
		var item ActiveMatchRecord
		if err := rows.Scan(
			&item.ID,
			&item.TargetUserID,
			&item.DisplayName,
			&item.Age,
			&item.CityID,
			&item.City,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan active match: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate active matches: %w", rows.Err())
	}

	return items, nil
}

func (r *MatchRepo) DeleteByUsers(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error) {
	if userID <= 0 || targetID <= 0 {
		return false, fmt.Errorf("invalid match delete payload")
	}
	if tx == nil {
		return false, fmt.Errorf("transaction is required")
	}

	userA := userID
	userB := targetID
	if userA > userB {
		userA, userB = userB, userA
	}

	result, err := tx.Exec(ctx, `
DELETE FROM matches
WHERE user_a_id = $1 AND user_b_id = $2
`, userA, userB)
	if err != nil {
		return false, fmt.Errorf("delete match: %w", err)
	}

	return result.RowsAffected() > 0, nil
}
