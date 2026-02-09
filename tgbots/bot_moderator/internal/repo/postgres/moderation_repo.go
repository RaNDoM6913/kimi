package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"bot_moderator/internal/domain/model"
	"github.com/lib/pq"
)

var ErrModerationQueueEmpty = errors.New("moderation queue is empty")
var ErrModerationItemNotFound = errors.New("moderation item not found")
var ErrModerationItemNotPending = errors.New("moderation item is not pending")

type ModerationRepo struct {
	db *sql.DB
}

func NewModerationRepo(db *sql.DB) *ModerationRepo {
	return &ModerationRepo{db: db}
}

func (r *ModerationRepo) AcquireNextPending(ctx context.Context, actorTGID int64, lockDuration time.Duration) (model.ModerationItem, error) {
	if r.db == nil {
		return model.ModerationItem{}, ErrModerationQueueEmpty
	}
	if actorTGID == 0 {
		return model.ModerationItem{}, fmt.Errorf("invalid actor tg id")
	}
	if lockDuration <= 0 {
		lockDuration = 10 * time.Minute
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.ModerationItem{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	item := model.ModerationItem{}
	var status string
	var intervalSeconds int64 = int64(lockDuration / time.Second)

	err = tx.QueryRowContext(ctx, `
		WITH candidate AS (
			SELECT id
			FROM moderation_items
			WHERE UPPER(status) = 'PENDING'
			  AND (locked_until IS NULL OR locked_until < NOW())
			ORDER BY created_at ASC, id ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE moderation_items mi
		SET locked_by_tg_id = $1,
		    locked_at = NOW(),
		    locked_until = NOW() + make_interval(secs => $2),
		    updated_at = NOW()
		FROM candidate
		WHERE mi.id = candidate.id
		RETURNING mi.id,
		          mi.user_id,
		          mi.status,
		          mi.eta_bucket,
		          mi.created_at,
		          mi.locked_at,
		          mi.locked_until,
		          mi.updated_at,
		          mi.target_type,
		          mi.target_id,
		          mi.moderator_tg_id,
		          mi.locked_by_tg_id
	`, actorTGID, intervalSeconds).Scan(
		&item.ID,
		&item.UserID,
		&status,
		&item.ETABucket,
		&item.CreatedAt,
		&item.LockedAt,
		&item.LockedUntil,
		&item.UpdatedAt,
		&item.TargetType,
		&item.TargetID,
		&item.ModeratorTGID,
		&item.LockedByTGID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ModerationItem{}, ErrModerationQueueEmpty
		}
		return model.ModerationItem{}, fmt.Errorf("acquire next pending moderation item: %w", err)
	}

	item.Status = model.ModerationStatus(strings.ToUpper(strings.TrimSpace(status)))

	if err := tx.Commit(); err != nil {
		return model.ModerationItem{}, fmt.Errorf("commit transaction: %w", err)
	}

	return item, nil
}

func (r *ModerationRepo) GetByID(ctx context.Context, moderationItemID int64) (model.ModerationItem, error) {
	if r.db == nil {
		return model.ModerationItem{}, ErrModerationItemNotFound
	}
	if moderationItemID <= 0 {
		return model.ModerationItem{}, fmt.Errorf("invalid moderation item id")
	}

	item := model.ModerationItem{}
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id,
		       user_id,
		       status,
		       eta_bucket,
		       created_at,
		       locked_at,
		       locked_until,
		       updated_at,
		       target_type,
		       target_id,
		       moderator_tg_id,
		       locked_by_tg_id
		FROM moderation_items
		WHERE id = $1
		LIMIT 1
	`, moderationItemID).Scan(
		&item.ID,
		&item.UserID,
		&status,
		&item.ETABucket,
		&item.CreatedAt,
		&item.LockedAt,
		&item.LockedUntil,
		&item.UpdatedAt,
		&item.TargetType,
		&item.TargetID,
		&item.ModeratorTGID,
		&item.LockedByTGID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ModerationItem{}, ErrModerationItemNotFound
		}
		return model.ModerationItem{}, fmt.Errorf("get moderation item by id: %w", err)
	}

	item.Status = model.ModerationStatus(strings.ToUpper(strings.TrimSpace(status)))
	return item, nil
}

func (r *ModerationRepo) MarkRejected(
	ctx context.Context,
	moderationItemID int64,
	reasonCode string,
	reasonText string,
	requiredFixStep string,
) error {
	if r.db == nil {
		return ErrModerationItemNotFound
	}
	if moderationItemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE moderation_items
		SET status = 'REJECTED',
		    reason_code = $2,
		    reason_text = $3,
		    required_fix_step = $4,
		    decided_at = NOW(),
		    locked_by_tg_id = NULL,
		    locked_until = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND UPPER(status) = 'PENDING'
	`, moderationItemID, strings.TrimSpace(reasonCode), strings.TrimSpace(reasonText), strings.TrimSpace(requiredFixStep))
	if err != nil {
		return fmt.Errorf("mark moderation item rejected: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for reject moderation item: %w", err)
	}
	if affected == 0 {
		return ErrModerationItemNotPending
	}

	return nil
}

func (r *ModerationRepo) MarkApproved(ctx context.Context, moderationItemID int64) error {
	if r.db == nil {
		return ErrModerationItemNotFound
	}
	if moderationItemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction for approve moderation item: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var userID int64
	err = tx.QueryRowContext(ctx, `
		UPDATE moderation_items
		SET status = 'APPROVED',
		    reason_code = NULL,
		    reason_text = NULL,
		    required_fix_step = NULL,
		    decided_at = NOW(),
		    locked_by_tg_id = NULL,
		    locked_until = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND UPPER(status) = 'PENDING'
		RETURNING user_id
	`, moderationItemID).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var exists bool
			if existsErr := tx.QueryRowContext(ctx, `
				SELECT EXISTS(SELECT 1 FROM moderation_items WHERE id = $1)
			`, moderationItemID).Scan(&exists); existsErr != nil {
				return fmt.Errorf("check moderation item existence: %w", existsErr)
			}
			if !exists {
				return ErrModerationItemNotFound
			}
			return ErrModerationItemNotPending
		}
		return fmt.Errorf("mark moderation item approved: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO profiles (user_id, display_name, moderation_status, approved, updated_at)
		VALUES ($1, '', 'APPROVED', TRUE, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			moderation_status = EXCLUDED.moderation_status,
			approved = EXCLUDED.approved,
			updated_at = NOW()
	`, userID)
	if err != nil {
		return fmt.Errorf("update profile moderation status on approve: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction for approve moderation item: %w", err)
	}

	return nil
}

func (r *ModerationRepo) InsertModerationAction(ctx context.Context, action model.BotModerationAction) error {
	if r.db == nil {
		return nil
	}

	var duration interface{}
	if action.DurationSec != nil {
		duration = *action.DurationSec
	}

	var moderationItemID interface{}
	if action.ModerationItemID > 0 {
		moderationItemID = pseudoUUID(action.ModerationItemID)
	}

	var reasonCode interface{}
	if trimmed := strings.TrimSpace(action.ReasonCode); trimmed != "" {
		reasonCode = trimmed
	}

	createdAt := action.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_moderation_actions (
			actor_tg_id,
			actor_role,
			target_user_id,
			moderation_item_id,
			decision,
			reason_code,
			duration_sec,
			created_at
		) VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6, $7, $8)
	`,
		action.ActorTGID,
		strings.TrimSpace(action.ActorRole),
		pseudoUUID(action.TargetUserID),
		moderationItemID,
		strings.TrimSpace(action.Decision),
		reasonCode,
		duration,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert bot moderation action: %w", err)
	}
	return nil
}

func (r *ModerationRepo) GetProfile(ctx context.Context, userID int64) (model.ModerationProfile, error) {
	if r.db == nil {
		return model.ModerationProfile{UserID: userID}, nil
	}

	profile := model.ModerationProfile{UserID: userID}
	var birthdate sql.NullTime
	goals := pq.StringArray{}
	languages := pq.StringArray{}
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id,
		       u.telegram_id,
		       COALESCE(u.username, ''),
		       COALESCE(p.display_name, ''),
		       COALESCE(p.city_id, ''),
		       p.birthdate,
		       COALESCE(p.gender, ''),
		       COALESCE(p.looking_for, ''),
		       COALESCE(p.goals, '{}'::text[]),
		       COALESCE(p.languages, '{}'::text[]),
		       COALESCE(p.occupation, ''),
		       COALESCE(p.education, '')
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE u.id = $1
		LIMIT 1
	`, userID).Scan(
		&profile.UserID,
		&profile.TGID,
		&profile.Username,
		&profile.DisplayName,
		&profile.CityID,
		&birthdate,
		&profile.Gender,
		&profile.LookingFor,
		&goals,
		&languages,
		&profile.Occupation,
		&profile.Education,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profile, nil
		}
		return model.ModerationProfile{}, fmt.Errorf("get moderation profile: %w", err)
	}

	if birthdate.Valid {
		b := birthdate.Time
		profile.Birthdate = &b
		profile.Age = calculateAge(b, time.Now().UTC())
	}
	profile.Goals = append([]string{}, goals...)
	profile.Languages = append([]string{}, languages...)

	return profile, nil
}

func (r *ModerationRepo) ListPhotoKeys(ctx context.Context, userID int64, limit int) ([]string, error) {
	if r.db == nil {
		return []string{}, nil
	}
	if limit <= 0 {
		limit = 3
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT s3_key
		FROM media
		WHERE user_id = $1
		  AND kind = 'photo'
		  AND status = 'active'
		  AND position BETWEEN 1 AND 3
		ORDER BY position ASC, created_at ASC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list photo keys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0, limit)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan photo key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate photo keys: %w", err)
	}

	return keys, nil
}

func (r *ModerationRepo) GetLatestCircleKey(ctx context.Context, userID int64) (string, error) {
	if r.db == nil {
		return "", nil
	}

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
		return "", fmt.Errorf("get latest circle key: %w", err)
	}

	return key, nil
}

func calculateAge(birthdate time.Time, now time.Time) int {
	by, bm, bd := birthdate.Date()
	ny, nm, nd := now.Date()
	age := ny - by
	if nm < bm || (nm == bm && nd < bd) {
		age--
	}
	if age < 0 {
		return 0
	}
	return age
}

func pseudoUUID(id int64) string {
	var value uint64
	if id < 0 {
		value = uint64(-(id + 1))
		value++
	} else {
		value = uint64(id)
	}
	// Deterministic UUID-like value for BIGINT identifiers.
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", value&0xFFFFFFFFFFFF)
}
