package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bot_moderator/internal/domain/model"
	"github.com/lib/pq"
)

var ErrLookupUserNotFound = errors.New("lookup user not found")

type UsersLookupRepo struct {
	db *sql.DB
}

func NewUsersLookupRepo(db *sql.DB) *UsersLookupRepo {
	return &UsersLookupRepo{db: db}
}

func (r *UsersLookupRepo) FindUser(ctx context.Context, query string) (model.LookupUser, error) {
	if r.db == nil {
		return model.LookupUser{}, ErrLookupUserNotFound
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return model.LookupUser{}, ErrLookupUserNotFound
	}

	if tgID, err := strconv.ParseInt(cleanQuery, 10, 64); err == nil {
		user, lookupErr := r.findByTelegramID(ctx, tgID)
		if lookupErr == nil {
			return user, nil
		}
		if !errors.Is(lookupErr, ErrLookupUserNotFound) {
			return model.LookupUser{}, lookupErr
		}
	}

	username := strings.TrimPrefix(cleanQuery, "@")
	if username == "" {
		return model.LookupUser{}, ErrLookupUserNotFound
	}
	return r.findByUsername(ctx, username)
}

func (r *UsersLookupRepo) FindByUserID(ctx context.Context, userID int64) (model.LookupUser, error) {
	if r.db == nil {
		return model.LookupUser{}, ErrLookupUserNotFound
	}
	if userID <= 0 {
		return model.LookupUser{}, ErrLookupUserNotFound
	}

	return r.findUserBase(ctx, `
		SELECT u.id,
		       u.telegram_id,
		       COALESCE(u.username, ''),
		       COALESCE(p.city_id, ''),
		       p.birthdate,
		       COALESCE(p.gender, ''),
		       COALESCE(p.looking_for, ''),
		       COALESCE(p.goals, '{}'::text[]),
		       COALESCE(p.languages, '{}'::text[]),
		       COALESCE(p.occupation, ''),
		       COALESCE(p.education, ''),
		       COALESCE(p.moderation_status, ''),
		       COALESCE(p.approved, FALSE),
		       e.plus_expires_at,
		       e.boost_until,
		       COALESCE(e.superlike_credits, 0),
		       COALESCE(e.reveal_credits, 0),
		       COALESCE(e.like_tokens, 0)
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN entitlements e ON e.user_id = u.id
		WHERE u.id = $1
		LIMIT 1
	`, userID)
}

func (r *UsersLookupRepo) InsertAction(ctx context.Context, action model.BotLookupAction) error {
	if r.db == nil {
		return nil
	}
	if action.ActorTGID == 0 {
		return fmt.Errorf("invalid actor tg id")
	}

	query := strings.TrimSpace(action.Query)
	if query == "" {
		query = "-"
	}

	payload := action.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}

	createdAt := action.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	var foundUserID interface{}
	if action.FoundUserID != nil && *action.FoundUserID > 0 {
		foundUserID = pseudoUUID(*action.FoundUserID)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_lookup_actions (
			actor_tg_id,
			actor_role,
			query,
			found_user_id,
			action,
			payload,
			created_at
		) VALUES ($1, $2, $3, $4::uuid, $5, $6, $7)
	`,
		action.ActorTGID,
		strings.TrimSpace(action.ActorRole),
		query,
		foundUserID,
		strings.TrimSpace(action.Action),
		string(payload),
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert bot lookup action: %w", err)
	}
	return nil
}

func (r *UsersLookupRepo) ForceReview(ctx context.Context, userID int64) error {
	if r.db == nil {
		return nil
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin force review transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO profiles (user_id, display_name, moderation_status, approved, updated_at)
		VALUES ($1, '', 'PENDING', FALSE, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			moderation_status = 'PENDING',
			approved = FALSE,
			updated_at = NOW()
	`, userID)
	if err != nil {
		return fmt.Errorf("set profile pending moderation status: %w", err)
	}

	var pendingID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM moderation_items
		WHERE user_id = $1
		  AND UPPER(status) = 'PENDING'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`, userID).Scan(&pendingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find existing pending moderation item: %w", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO moderation_items (
				user_id,
				target_type,
				target_id,
				status,
				eta_bucket,
				moderator_tg_id,
				locked_by_tg_id,
				locked_at,
				locked_until,
				created_at,
				updated_at
			) VALUES ($1, 'profile', NULL, 'PENDING', 'up_to_10', NULL, NULL, NULL, NULL, NOW(), NOW())
		`, userID)
		if err != nil {
			return fmt.Errorf("insert force review moderation item: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE moderation_items
			SET locked_by_tg_id = NULL,
			    locked_at = NULL,
			    locked_until = NULL,
			    updated_at = NOW()
			WHERE id = $1
		`, pendingID)
		if err != nil {
			return fmt.Errorf("reset lock for existing moderation item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit force review transaction: %w", err)
	}
	return nil
}

func (r *UsersLookupRepo) findByTelegramID(ctx context.Context, tgID int64) (model.LookupUser, error) {
	return r.findUserBase(ctx, `
		SELECT u.id,
		       u.telegram_id,
		       COALESCE(u.username, ''),
		       COALESCE(p.city_id, ''),
		       p.birthdate,
		       COALESCE(p.gender, ''),
		       COALESCE(p.looking_for, ''),
		       COALESCE(p.goals, '{}'::text[]),
		       COALESCE(p.languages, '{}'::text[]),
		       COALESCE(p.occupation, ''),
		       COALESCE(p.education, ''),
		       COALESCE(p.moderation_status, ''),
		       COALESCE(p.approved, FALSE),
		       e.plus_expires_at,
		       e.boost_until,
		       COALESCE(e.superlike_credits, 0),
		       COALESCE(e.reveal_credits, 0),
		       COALESCE(e.like_tokens, 0)
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN entitlements e ON e.user_id = u.id
		WHERE u.telegram_id = $1
		LIMIT 1
	`, tgID)
}

func (r *UsersLookupRepo) findByUsername(ctx context.Context, username string) (model.LookupUser, error) {
	return r.findUserBase(ctx, `
		SELECT u.id,
		       u.telegram_id,
		       COALESCE(u.username, ''),
		       COALESCE(p.city_id, ''),
		       p.birthdate,
		       COALESCE(p.gender, ''),
		       COALESCE(p.looking_for, ''),
		       COALESCE(p.goals, '{}'::text[]),
		       COALESCE(p.languages, '{}'::text[]),
		       COALESCE(p.occupation, ''),
		       COALESCE(p.education, ''),
		       COALESCE(p.moderation_status, ''),
		       COALESCE(p.approved, FALSE),
		       e.plus_expires_at,
		       e.boost_until,
		       COALESCE(e.superlike_credits, 0),
		       COALESCE(e.reveal_credits, 0),
		       COALESCE(e.like_tokens, 0)
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN entitlements e ON e.user_id = u.id
		WHERE LOWER(u.username) = LOWER($1)
		LIMIT 1
	`, username)
}

func (r *UsersLookupRepo) findUserBase(ctx context.Context, query string, args ...interface{}) (model.LookupUser, error) {
	user := model.LookupUser{}
	var goals pq.StringArray
	var languages pq.StringArray
	var birthdate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.UserID,
		&user.TGID,
		&user.Username,
		&user.CityID,
		&birthdate,
		&user.Gender,
		&user.LookingFor,
		&goals,
		&languages,
		&user.Occupation,
		&user.Education,
		&user.ModerationStatus,
		&user.Approved,
		&user.PlusExpiresAt,
		&user.BoostUntil,
		&user.SuperlikeCredits,
		&user.RevealCredits,
		&user.LikeTokens,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.LookupUser{}, ErrLookupUserNotFound
		}
		return model.LookupUser{}, fmt.Errorf("lookup user: %w", err)
	}

	if birthdate.Valid {
		dob := birthdate.Time.UTC()
		user.Birthdate = &dob
		user.Age = calculateAge(dob, time.Now().UTC())
	}
	user.Goals = append([]string{}, goals...)
	user.Languages = append([]string{}, languages...)

	photoKeys, err := r.listPhotoKeys(ctx, user.UserID)
	if err != nil {
		return model.LookupUser{}, err
	}
	user.PhotoKeys = photoKeys

	circleKey, err := r.getLatestCircleKey(ctx, user.UserID)
	if err != nil {
		return model.LookupUser{}, err
	}
	user.CircleKey = circleKey

	banned, reason, err := r.getBanState(ctx, user.UserID)
	if err != nil {
		return model.LookupUser{}, err
	}
	user.IsBanned = banned
	user.BanReason = reason

	return user, nil
}

func (r *UsersLookupRepo) listPhotoKeys(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s3_key
		FROM media
		WHERE user_id = $1
		  AND kind = 'photo'
		  AND status = 'active'
		  AND position BETWEEN 1 AND 3
		ORDER BY position ASC, created_at ASC
		LIMIT 3
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list lookup photo keys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0, 3)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan lookup photo key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lookup photo keys: %w", err)
	}
	return keys, nil
}

func (r *UsersLookupRepo) getLatestCircleKey(ctx context.Context, userID int64) (string, error) {
	var key string
	err := r.db.QueryRowContext(ctx, `
		SELECT s3_key
		FROM media
		WHERE user_id = $1
		  AND kind = 'circle'
		  AND status = 'active'
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, userID).Scan(&key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("lookup latest circle key: %w", err)
	}
	return key, nil
}

func (r *UsersLookupRepo) getBanState(ctx context.Context, userID int64) (bool, string, error) {
	var banned bool
	var reason sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT banned, reason
		FROM user_bans
		WHERE user_id = $1::uuid
		LIMIT 1
	`, pseudoUUID(userID)).Scan(&banned, &reason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("lookup ban state: %w", err)
	}
	return banned, strings.TrimSpace(reason.String), nil
}
