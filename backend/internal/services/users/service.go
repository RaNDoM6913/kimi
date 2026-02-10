package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

const signedURLTTL = 5 * time.Minute

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("user not found")
)

type URLSigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type LookupUser struct {
	UserID           int64
	TGID             int64
	Username         string
	CityID           string
	Birthdate        *time.Time
	Age              int
	Gender           string
	LookingFor       string
	Goals            []string
	Languages        []string
	Occupation       string
	Education        string
	ModerationStatus string
	Approved         bool
	PhotoKeys        []string
	CircleKey        string
	PhotoURLs        []string
	CircleURL        string
	PlusExpiresAt    *time.Time
	BoostUntil       *time.Time
	SuperlikeCredits int
	RevealCredits    int
	LikeTokens       int
	IsBanned         bool
	BanReason        string
}

type PrivateUser struct {
	UserID    int64
	PhoneE164 *string
	Lat       *float64
	Lon       *float64
	LastGeoAt *time.Time
}

type Service struct {
	pool      *pgxpool.Pool
	media     *pgrepo.MediaRepo
	urlSigner URLSigner
	now       func() time.Time
}

func NewService(pool *pgxpool.Pool, media *pgrepo.MediaRepo, signer URLSigner) *Service {
	return &Service{
		pool:      pool,
		media:     media,
		urlSigner: signer,
		now:       time.Now,
	}
}

func (s *Service) LookupUser(ctx context.Context, query string) (LookupUser, error) {
	if s.pool == nil {
		return LookupUser{}, ErrNotFound
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return LookupUser{}, ErrValidation
	}

	if strings.HasPrefix(cleanQuery, "@") {
		return s.findByUsername(ctx, strings.TrimPrefix(cleanQuery, "@"))
	}

	if numeric, err := strconv.ParseInt(cleanQuery, 10, 64); err == nil {
		user, lookupErr := s.findByUserID(ctx, numeric)
		if lookupErr == nil {
			return user, nil
		}
		if !errors.Is(lookupErr, ErrNotFound) {
			return LookupUser{}, lookupErr
		}

		user, lookupErr = s.findByTGID(ctx, numeric)
		if lookupErr == nil {
			return user, nil
		}
		if !errors.Is(lookupErr, ErrNotFound) {
			return LookupUser{}, lookupErr
		}
		return LookupUser{}, ErrNotFound
	}

	return s.findByUsername(ctx, cleanQuery)
}

func (s *Service) SetBan(ctx context.Context, userID int64, banned bool, reason string, updatedByTGID int64) error {
	if s.pool == nil {
		return nil
	}
	if userID <= 0 || updatedByTGID == 0 {
		return ErrValidation
	}
	exists, err := s.userExists(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	if _, err := s.pool.Exec(ctx, `
INSERT INTO user_bans (user_id, banned, reason, updated_by_tg_id, updated_at)
VALUES ($1::uuid, $2, NULLIF($3, ''), $4, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	banned = EXCLUDED.banned,
	reason = EXCLUDED.reason,
	updated_by_tg_id = EXCLUDED.updated_by_tg_id,
	updated_at = EXCLUDED.updated_at
`, pseudoUUID(userID), banned, strings.TrimSpace(reason), updatedByTGID); err != nil {
		return fmt.Errorf("upsert user ban: %w", err)
	}
	return nil
}

func (s *Service) ForceReview(ctx context.Context, userID int64) error {
	if s.pool == nil {
		return nil
	}
	if userID <= 0 {
		return ErrValidation
	}
	exists, err := s.userExists(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin force-review transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `
INSERT INTO profiles (user_id, display_name, moderation_status, approved, updated_at)
VALUES ($1, '', 'PENDING', FALSE, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	moderation_status = 'PENDING',
	approved = FALSE,
	updated_at = NOW()
`, userID); err != nil {
		return fmt.Errorf("set profile pending moderation status: %w", err)
	}

	var pendingID int64
	err = tx.QueryRow(ctx, `
SELECT id
FROM moderation_items
WHERE user_id = $1
  AND UPPER(status) = 'PENDING'
ORDER BY created_at ASC, id ASC
LIMIT 1
`, userID).Scan(&pendingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find pending moderation item: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := tx.Exec(ctx, `
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
`, userID); err != nil {
			return fmt.Errorf("insert force-review moderation item: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
UPDATE moderation_items
SET
	locked_by_tg_id = NULL,
	locked_at = NULL,
	locked_until = NULL,
	updated_at = NOW()
WHERE id = $1
`, pendingID); err != nil {
			return fmt.Errorf("unlock pending moderation item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit force-review transaction: %w", err)
	}
	return nil
}

func (s *Service) GetPrivate(ctx context.Context, userID int64) (PrivateUser, error) {
	if userID <= 0 {
		return PrivateUser{}, ErrValidation
	}
	if s.pool == nil {
		return PrivateUser{}, ErrNotFound
	}

	var (
		private   PrivateUser
		phoneE164 sql.NullString
		lat       sql.NullFloat64
		lon       sql.NullFloat64
		lastGeoAt sql.NullTime
	)

	err := s.pool.QueryRow(ctx, `
SELECT u.id,
       up.phone_e164,
       p.last_lat,
       p.last_lon,
       p.last_geo_at
FROM users u
LEFT JOIN user_private up ON up.user_id = u.id
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.id = $1
LIMIT 1
`, userID).Scan(&private.UserID, &phoneE164, &lat, &lon, &lastGeoAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PrivateUser{}, ErrNotFound
		}
		return PrivateUser{}, fmt.Errorf("get private user data: %w", err)
	}

	if phoneE164.Valid {
		value := strings.TrimSpace(phoneE164.String)
		if value != "" {
			private.PhoneE164 = &value
		}
	}
	if lat.Valid {
		value := lat.Float64
		private.Lat = &value
	}
	if lon.Valid {
		value := lon.Float64
		private.Lon = &value
	}
	if lastGeoAt.Valid {
		value := lastGeoAt.Time.UTC()
		private.LastGeoAt = &value
	}

	return private, nil
}

func (s *Service) findByUserID(ctx context.Context, userID int64) (LookupUser, error) {
	if userID <= 0 {
		return LookupUser{}, ErrValidation
	}
	return s.findUserBase(ctx, "u.id = $1", userID)
}

func (s *Service) findByTGID(ctx context.Context, tgID int64) (LookupUser, error) {
	if tgID <= 0 {
		return LookupUser{}, ErrValidation
	}
	return s.findUserBase(ctx, "u.telegram_id = $1", tgID)
}

func (s *Service) findByUsername(ctx context.Context, username string) (LookupUser, error) {
	clean := strings.TrimSpace(strings.TrimPrefix(username, "@"))
	if clean == "" {
		return LookupUser{}, ErrValidation
	}
	return s.findUserBase(ctx, "LOWER(u.username) = LOWER($1)", clean)
}

func (s *Service) findUserBase(ctx context.Context, predicate string, arg interface{}) (LookupUser, error) {
	if s.pool == nil {
		return LookupUser{}, ErrNotFound
	}

	query := fmt.Sprintf(`
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
WHERE %s
LIMIT 1
`, predicate)

	var user LookupUser
	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&user.UserID,
		&user.TGID,
		&user.Username,
		&user.CityID,
		&user.Birthdate,
		&user.Gender,
		&user.LookingFor,
		&user.Goals,
		&user.Languages,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return LookupUser{}, ErrNotFound
		}
		return LookupUser{}, fmt.Errorf("lookup user: %w", err)
	}

	if user.Birthdate != nil {
		dob := user.Birthdate.UTC()
		user.Birthdate = &dob
		user.Age = calculateAge(dob, s.now().UTC())
	}

	if err := s.attachMedia(ctx, &user); err != nil {
		return LookupUser{}, err
	}
	banned, reason, err := s.banState(ctx, user.UserID)
	if err != nil {
		return LookupUser{}, err
	}
	user.IsBanned = banned
	user.BanReason = reason

	return user, nil
}

func (s *Service) attachMedia(ctx context.Context, user *LookupUser) error {
	if user == nil || user.UserID <= 0 || s.media == nil {
		return nil
	}

	photos, err := s.media.ListUserPhotos(ctx, user.UserID, 3)
	if err != nil {
		return fmt.Errorf("list user photos: %w", err)
	}
	user.PhotoKeys = make([]string, 0, len(photos))
	user.PhotoURLs = make([]string, 0, len(photos))
	for _, photo := range photos {
		key := strings.TrimSpace(photo.ObjectKey)
		if key == "" {
			continue
		}
		user.PhotoKeys = append(user.PhotoKeys, key)
		url, err := s.signKey(ctx, key)
		if err != nil {
			return err
		}
		if strings.TrimSpace(url) != "" {
			user.PhotoURLs = append(user.PhotoURLs, url)
		}
	}

	circle, err := s.media.GetLatestCircle(ctx, user.UserID)
	if err != nil {
		return fmt.Errorf("get latest circle: %w", err)
	}
	if circle == nil {
		return nil
	}

	user.CircleKey = strings.TrimSpace(circle.ObjectKey)
	url, err := s.signKey(ctx, user.CircleKey)
	if err != nil {
		return err
	}
	user.CircleURL = strings.TrimSpace(url)
	return nil
}

func (s *Service) signKey(ctx context.Context, key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed, nil
	}
	if s.urlSigner == nil {
		return "", nil
	}
	url, err := s.urlSigner.PresignGet(ctx, trimmed, signedURLTTL)
	if err != nil {
		return "", fmt.Errorf("sign media key: %w", err)
	}
	return url, nil
}

func (s *Service) banState(ctx context.Context, userID int64) (bool, string, error) {
	if s.pool == nil || userID <= 0 {
		return false, "", nil
	}

	var banned bool
	var reason sql.NullString
	err := s.pool.QueryRow(ctx, `
SELECT banned, reason
FROM user_bans
WHERE user_id = $1::uuid
LIMIT 1
`, pseudoUUID(userID)).Scan(&banned, &reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("lookup user ban state: %w", err)
	}
	return banned, strings.TrimSpace(reason.String), nil
}

func (s *Service) userExists(ctx context.Context, userID int64) (bool, error) {
	if s.pool == nil {
		return false, nil
	}
	if userID <= 0 {
		return false, ErrValidation
	}

	var id int64
	err := s.pool.QueryRow(ctx, `
SELECT id
FROM users
WHERE id = $1
LIMIT 1
`, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check user existence: %w", err)
	}
	return id > 0, nil
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
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", value&0xFFFFFFFFFFFF)
}
