package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, sessionID uuid.UUID, adminUserID int64, expiresAt time.Time, idleTimeout time.Duration, ip, userAgent string) error {
	const query = `
INSERT INTO admin_sessions (id, admin_user_id, expires_at, last_seen_at, idle_expires_at, ip_address, user_agent)
VALUES ($1, $2, $3, NOW(), NOW() + ($4 * INTERVAL '1 second'), NULLIF($5, ''), NULLIF($6, ''))
`
	seconds := secondsOrDefault(idleTimeout, 1800)
	_, err := r.pool.Exec(ctx, query, sessionID, adminUserID, expiresAt.UTC(), seconds, ip, userAgent)
	if err != nil {
		return fmt.Errorf("create admin session: %w", err)
	}
	return nil
}

func (r *SessionRepo) Touch(ctx context.Context, sessionID uuid.UUID, adminUserID int64, idleTimeout time.Duration) error {
	const query = `
UPDATE admin_sessions
SET last_seen_at = NOW(),
    idle_expires_at = NOW() + ($3 * INTERVAL '1 second')
WHERE id = $1
  AND admin_user_id = $2
  AND revoked_at IS NULL
  AND expires_at > NOW()
  AND idle_expires_at > NOW()
`
	seconds := secondsOrDefault(idleTimeout, 1800)
	res, err := r.pool.Exec(ctx, query, sessionID, adminUserID, seconds)
	if err != nil {
		return fmt.Errorf("touch admin session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SessionRepo) Revoke(ctx context.Context, sessionID uuid.UUID, adminUserID int64) error {
	const query = `
UPDATE admin_sessions
SET revoked_at = NOW()
WHERE id = $1
  AND admin_user_id = $2
  AND revoked_at IS NULL
`
	res, err := r.pool.Exec(ctx, query, sessionID, adminUserID)
	if err != nil {
		return fmt.Errorf("revoke admin session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func secondsOrDefault(d time.Duration, fallback int64) int64 {
	seconds := int64(d.Seconds())
	if seconds <= 0 {
		return fallback
	}
	return seconds
}
