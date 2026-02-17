package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChallengeStatus string

const (
	ChallengeTelegramVerified ChallengeStatus = "telegram_verified"
	ChallengeTOTPVerified     ChallengeStatus = "totp_verified"
	ChallengeCompleted        ChallengeStatus = "completed"
)

type LoginChallenge struct {
	ID          uuid.UUID
	AdminUserID int64
	Status      ChallengeStatus
	ExpiresAt   time.Time
}

type ChallengeRepo struct {
	pool *pgxpool.Pool
}

func NewChallengeRepo(pool *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{pool: pool}
}

func (r *ChallengeRepo) Create(ctx context.Context, adminUserID int64, ttl time.Duration, ip, userAgent string) (LoginChallenge, error) {
	const query = `
INSERT INTO admin_login_challenges (id, admin_user_id, status, expires_at, ip_address, user_agent)
VALUES ($1, $2, $3, NOW() + ($4 * INTERVAL '1 second'), NULLIF($5, ''), NULLIF($6, ''))
RETURNING id, admin_user_id, status, expires_at
`
	seconds := int64(ttl.Seconds())
	if seconds <= 0 {
		seconds = 600
	}
	id := uuid.New()
	var out LoginChallenge
	err := r.pool.QueryRow(ctx, query, id, adminUserID, string(ChallengeTelegramVerified), seconds, ip, userAgent).Scan(
		&out.ID,
		&out.AdminUserID,
		&out.Status,
		&out.ExpiresAt,
	)
	if err != nil {
		return LoginChallenge{}, fmt.Errorf("create login challenge: %w", err)
	}
	return out, nil
}

func (r *ChallengeRepo) GetActive(ctx context.Context, id uuid.UUID) (LoginChallenge, error) {
	const query = `
SELECT id, admin_user_id, status, expires_at
FROM admin_login_challenges
WHERE id = $1
  AND expires_at > NOW()
`
	var out LoginChallenge
	err := r.pool.QueryRow(ctx, query, id).Scan(&out.ID, &out.AdminUserID, &out.Status, &out.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginChallenge{}, ErrNotFound
		}
		return LoginChallenge{}, fmt.Errorf("get login challenge: %w", err)
	}
	return out, nil
}

func (r *ChallengeRepo) AdvanceStatus(ctx context.Context, id uuid.UUID, from, to ChallengeStatus) error {
	const query = `
UPDATE admin_login_challenges
SET status = $3
WHERE id = $1
  AND status = $2
  AND expires_at > NOW()
`
	res, err := r.pool.Exec(ctx, query, id, string(from), string(to))
	if err != nil {
		return fmt.Errorf("advance challenge status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ChallengeRepo) Expire(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE admin_login_challenges
SET expires_at = NOW()
WHERE id = $1
`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("expire challenge: %w", err)
	}
	return nil
}
