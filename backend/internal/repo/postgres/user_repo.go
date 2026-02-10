package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepo struct {
	pool *pgxpool.Pool
}

type UserRecord struct {
	ID         int64
	TelegramID int64
	Username   string
	Role       string
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) FindByTelegramID(ctx context.Context, telegramID int64) (UserRecord, error) {
	if r.pool == nil {
		return UserRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if telegramID <= 0 {
		return UserRecord{}, fmt.Errorf("invalid telegram_id")
	}

	var user UserRecord
	err := r.pool.QueryRow(ctx, `
SELECT id, telegram_id, username, role
FROM users
WHERE telegram_id = $1
`, telegramID).Scan(&user.ID, &user.TelegramID, &user.Username, &user.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, fmt.Errorf("find user by telegram_id: %w", err)
	}

	return user, nil
}

func (r *UserRepo) GetOrCreateByTelegramID(ctx context.Context, telegramID int64) (UserRecord, error) {
	if telegramID <= 0 {
		return UserRecord{}, fmt.Errorf("invalid telegram_id")
	}
	if r.pool == nil {
		return UserRecord{
			ID:         telegramID,
			TelegramID: telegramID,
			Role:       "user",
		}, nil
	}

	var user UserRecord
	err := r.pool.QueryRow(ctx, `
INSERT INTO users (telegram_id, username, role, created_at, updated_at)
VALUES ($1, '', 'user', NOW(), NOW())
ON CONFLICT (telegram_id) DO UPDATE SET
	updated_at = NOW()
RETURNING id, telegram_id, username, role
`, telegramID).Scan(&user.ID, &user.TelegramID, &user.Username, &user.Role)
	if err != nil {
		return UserRecord{}, fmt.Errorf("get or create user by telegram_id: %w", err)
	}
	if strings.TrimSpace(user.Role) == "" {
		user.Role = "user"
	}

	return user, nil
}

func (r *UserRepo) UpdateUsername(ctx context.Context, userID int64, username string) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 || strings.TrimSpace(username) == "" {
		return nil
	}

	if _, err := r.pool.Exec(ctx, `
UPDATE users
SET username = $2, updated_at = NOW()
WHERE id = $1
`, userID, strings.TrimSpace(username)); err != nil {
		return fmt.Errorf("update user username: %w", err)
	}

	return nil
}
