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

type UserDeviceRepo struct {
	pool *pgxpool.Pool
}

func NewUserDeviceRepo(pool *pgxpool.Pool) *UserDeviceRepo {
	return &UserDeviceRepo{pool: pool}
}

func (r *UserDeviceRepo) UpsertSeen(ctx context.Context, userID int64, deviceID string, seenAt time.Time) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	if r.pool == nil {
		return nil
	}

	if _, err := r.pool.Exec(ctx, `
INSERT INTO user_devices (
	user_id,
	device_id,
	first_seen_at,
	last_seen_at
) VALUES ($1, $2, $3, $3)
ON CONFLICT (user_id, device_id) DO UPDATE SET
	last_seen_at = GREATEST(user_devices.last_seen_at, EXCLUDED.last_seen_at)
`, userID, deviceID, seenAt.UTC()); err != nil {
		return fmt.Errorf("upsert user device: %w", err)
	}

	return nil
}

func (r *UserDeviceRepo) IsKnown(ctx context.Context, userID int64, deviceID string) (bool, error) {
	if userID <= 0 {
		return false, fmt.Errorf("invalid user id")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, fmt.Errorf("device id is required")
	}
	if r.pool == nil {
		// In degraded mode do not penalize all devices as "new".
		return true, nil
	}

	var exists bool
	err := r.pool.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM user_devices
	WHERE user_id = $1
	  AND device_id = $2
)
`, userID, deviceID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check known user device: %w", err)
	}

	return exists, nil
}
