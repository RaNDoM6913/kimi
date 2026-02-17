package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAdminSessionNotFound = errors.New("admin session not found")

type AdminSessionRepo struct {
	pool *pgxpool.Pool
}

func NewAdminSessionRepo(pool *pgxpool.Pool) *AdminSessionRepo {
	return &AdminSessionRepo{pool: pool}
}

func (r *AdminSessionRepo) Touch(ctx context.Context, sid uuid.UUID, adminUserID int64, idleTimeout time.Duration) (string, error) {
	if r.pool == nil {
		return "", fmt.Errorf("postgres pool is nil")
	}
	if adminUserID <= 0 {
		return "", fmt.Errorf("invalid admin_user_id")
	}

	const query = `
UPDATE admin_sessions AS s
SET last_seen_at = NOW(),
    idle_expires_at = NOW() + ($3 * INTERVAL '1 second')
FROM admin_users AS u
WHERE s.id = $1
  AND s.admin_user_id = $2
  AND s.admin_user_id = u.id
  AND u.is_active = TRUE
  AND s.revoked_at IS NULL
  AND s.expires_at > NOW()
  AND s.idle_expires_at > NOW()
RETURNING u.role
`
	seconds := int64(idleTimeout.Seconds())
	if seconds <= 0 {
		seconds = 1800
	}

	var role string
	err := r.pool.QueryRow(ctx, query, sid, adminUserID, seconds).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAdminSessionNotFound
		}
		return "", fmt.Errorf("touch admin session: %w", err)
	}

	role = strings.TrimSpace(role)
	if role == "" {
		role = "ADMIN"
	}
	return role, nil
}
