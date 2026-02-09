package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"bot_moderator/internal/domain/model"
)

type BotUsersRepo struct {
	db *sql.DB
}

func NewBotUsersRepo(db *sql.DB) *BotUsersRepo {
	return &BotUsersRepo{db: db}
}

func (r *BotUsersRepo) Upsert(ctx context.Context, user model.BotUser) error {
	if r.db == nil {
		return nil
	}

	lastSeenAt := user.LastSeenAt
	if lastSeenAt.IsZero() {
		lastSeenAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_users (tg_id, username, first_name, last_name, last_seen_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tg_id)
		DO UPDATE
		SET username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			last_seen_at = EXCLUDED.last_seen_at
	`,
		user.TgID,
		nullableString(user.Username),
		nullableString(user.FirstName),
		nullableString(user.LastName),
		lastSeenAt,
	)
	return err
}

func (r *BotUsersRepo) ListRecent(ctx context.Context, limit int) ([]model.BotUser, error) {
	if r.db == nil {
		return []model.BotUser{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT tg_id, COALESCE(username, ''), COALESCE(first_name, ''), COALESCE(last_name, ''), last_seen_at
		FROM bot_users
		ORDER BY last_seen_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.BotUser, 0, limit)
	for rows.Next() {
		var user model.BotUser
		if err := rows.Scan(&user.TgID, &user.Username, &user.FirstName, &user.LastName, &user.LastSeenAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *BotUsersRepo) GetByTGID(ctx context.Context, tgID int64) (model.BotUser, error) {
	if r.db == nil {
		return model.BotUser{TgID: tgID}, nil
	}

	var user model.BotUser
	err := r.db.QueryRowContext(ctx, `
		SELECT tg_id, COALESCE(username, ''), COALESCE(first_name, ''), COALESCE(last_name, ''), last_seen_at
		FROM bot_users
		WHERE tg_id = $1
	`, tgID).Scan(&user.TgID, &user.Username, &user.FirstName, &user.LastName, &user.LastSeenAt)
	if err != nil {
		return model.BotUser{}, err
	}

	return user, nil
}

func nullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
