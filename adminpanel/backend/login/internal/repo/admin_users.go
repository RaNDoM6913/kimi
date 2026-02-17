package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type AdminUser struct {
	ID             int64
	TelegramID     int64
	Username       string
	DisplayName    string
	Role           string
	PasswordHash   string
	TOTPSecret     string
	TOTPEnabled    bool
	IsActive       bool
	FailedAttempts int
	LockedUntil    *time.Time
}

type AdminUserRepo struct {
	pool *pgxpool.Pool
}

func NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo {
	return &AdminUserRepo{pool: pool}
}

func (r *AdminUserRepo) FindByTelegramID(ctx context.Context, telegramID int64) (AdminUser, error) {
	const query = `
SELECT id, telegram_id, COALESCE(username, ''), COALESCE(display_name, ''), role,
       password_hash, COALESCE(totp_secret, ''), totp_enabled, is_active,
       failed_login_attempts, locked_until
FROM admin_users
WHERE telegram_id = $1
`
	return r.scanOne(ctx, query, telegramID)
}

func (r *AdminUserRepo) FindByID(ctx context.Context, userID int64) (AdminUser, error) {
	const query = `
SELECT id, telegram_id, COALESCE(username, ''), COALESCE(display_name, ''), role,
       password_hash, COALESCE(totp_secret, ''), totp_enabled, is_active,
       failed_login_attempts, locked_until
FROM admin_users
WHERE id = $1
`
	return r.scanOne(ctx, query, userID)
}

func (r *AdminUserRepo) scanOne(ctx context.Context, query string, arg any) (AdminUser, error) {
	var user AdminUser
	var lockedUntil *time.Time
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.DisplayName,
		&user.Role,
		&user.PasswordHash,
		&user.TOTPSecret,
		&user.TOTPEnabled,
		&user.IsActive,
		&user.FailedAttempts,
		&lockedUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminUser{}, ErrNotFound
		}
		return AdminUser{}, fmt.Errorf("query admin user: %w", err)
	}
	user.LockedUntil = lockedUntil
	return user, nil
}

func (r *AdminUserRepo) ResetFailures(ctx context.Context, userID int64) error {
	const query = `
UPDATE admin_users
SET failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $1
`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("reset failed login attempts: %w", err)
	}
	return nil
}

func (r *AdminUserRepo) MarkFailure(ctx context.Context, userID int64, maxAttempts int, lockUntil time.Time) (bool, error) {
	const query = `
UPDATE admin_users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= $2 THEN $3
        ELSE locked_until
    END,
    updated_at = NOW()
WHERE id = $1
RETURNING locked_until
`
	var storedLock *time.Time
	err := r.pool.QueryRow(ctx, query, userID, maxAttempts, lockUntil).Scan(&storedLock)
	if err != nil {
		return false, fmt.Errorf("mark login failure: %w", err)
	}
	return storedLock != nil && storedLock.After(time.Now().UTC()), nil
}

func (r *AdminUserRepo) EnableTOTP(ctx context.Context, userID int64, secret string) error {
	const query = `
UPDATE admin_users
SET totp_secret = $2,
    totp_enabled = TRUE,
    updated_at = NOW()
WHERE id = $1
`
	res, err := r.pool.Exec(ctx, query, userID, secret)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
