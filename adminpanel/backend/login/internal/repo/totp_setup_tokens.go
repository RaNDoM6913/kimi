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

type TOTPSetupToken struct {
	ID          uuid.UUID
	AdminUserID int64
	Secret      string
	ExpiresAt   time.Time
}

type TOTPSetupTokenRepo struct {
	pool *pgxpool.Pool
}

func NewTOTPSetupTokenRepo(pool *pgxpool.Pool) *TOTPSetupTokenRepo {
	return &TOTPSetupTokenRepo{pool: pool}
}

func (r *TOTPSetupTokenRepo) Create(ctx context.Context, adminUserID int64, secret string, ttl time.Duration) (TOTPSetupToken, error) {
	const query = `
INSERT INTO admin_totp_setup_tokens (id, admin_user_id, secret, expires_at)
VALUES ($1, $2, $3, NOW() + ($4 * INTERVAL '1 second'))
RETURNING id, admin_user_id, secret, expires_at
`
	seconds := int64(ttl.Seconds())
	if seconds <= 0 {
		seconds = 600
	}
	id := uuid.New()
	var token TOTPSetupToken
	err := r.pool.QueryRow(ctx, query, id, adminUserID, secret, seconds).Scan(
		&token.ID,
		&token.AdminUserID,
		&token.Secret,
		&token.ExpiresAt,
	)
	if err != nil {
		return TOTPSetupToken{}, fmt.Errorf("create totp setup token: %w", err)
	}
	return token, nil
}

func (r *TOTPSetupTokenRepo) Get(ctx context.Context, id uuid.UUID) (TOTPSetupToken, error) {
	const query = `
SELECT id, admin_user_id, secret, expires_at
FROM admin_totp_setup_tokens
WHERE id = $1
  AND expires_at > NOW()
`
	var token TOTPSetupToken
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&token.ID,
		&token.AdminUserID,
		&token.Secret,
		&token.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TOTPSetupToken{}, ErrNotFound
		}
		return TOTPSetupToken{}, fmt.Errorf("get totp setup token: %w", err)
	}
	return token, nil
}

func (r *TOTPSetupTokenRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM admin_totp_setup_tokens WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete totp setup token: %w", err)
	}
	return nil
}
