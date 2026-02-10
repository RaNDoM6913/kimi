package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

type ProfileRepo struct {
	pool *pgxpool.Pool
}

type ProfileModerationSnapshot struct {
	Status   string
	Approved bool
}

type ProfileQueueSummary struct {
	UserID      int64
	DisplayName string
	CityID      string
	Gender      string
	LookingFor  string
	Goals       []string
	Birthdate   *time.Time
	Occupation  string
	Education   string
}

func NewProfileRepo(pool *pgxpool.Pool) *ProfileRepo {
	return &ProfileRepo{pool: pool}
}

func (r *ProfileRepo) SaveLocation(ctx context.Context, userID int64, cityID string, lat, lon float64, at time.Time) error {
	if r.pool == nil {
		return nil
	}

	const query = `
INSERT INTO profiles (
	user_id,
	display_name,
	city_id,
	city,
	last_geo_at,
	last_lat,
	last_lon,
	updated_at
) VALUES ($1, '', $2, $2, $3, $4, $5, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	city_id = EXCLUDED.city_id,
	city = EXCLUDED.city,
	last_geo_at = EXCLUDED.last_geo_at,
	last_lat = EXCLUDED.last_lat,
	last_lon = EXCLUDED.last_lon,
	updated_at = NOW()
`

	if _, err := r.pool.Exec(ctx, query, userID, cityID, at.UTC(), lat, lon); err != nil {
		return fmt.Errorf("save profile location: %w", err)
	}

	return nil
}

func (r *ProfileRepo) SaveCore(
	ctx context.Context,
	userID int64,
	birthdate time.Time,
	gender string,
	lookingFor string,
	occupation string,
	education string,
	heightCM int,
	eyeColor string,
	zodiac string,
	languages []string,
	goals []string,
	profileCompleted bool,
) error {
	if r.pool == nil {
		return nil
	}

	const query = `
INSERT INTO profiles (
	user_id,
	display_name,
	birthdate,
	gender,
	looking_for,
	occupation,
	education,
	height_cm,
	eye_color,
	zodiac,
	languages,
	goals,
	profile_completed,
	updated_at
) VALUES ($1, '', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	birthdate = EXCLUDED.birthdate,
	gender = EXCLUDED.gender,
	looking_for = EXCLUDED.looking_for,
	occupation = EXCLUDED.occupation,
	education = EXCLUDED.education,
	height_cm = EXCLUDED.height_cm,
	eye_color = EXCLUDED.eye_color,
	zodiac = EXCLUDED.zodiac,
	languages = EXCLUDED.languages,
	goals = EXCLUDED.goals,
	profile_completed = EXCLUDED.profile_completed,
	updated_at = NOW()
`

	if _, err := r.pool.Exec(
		ctx,
		query,
		userID,
		birthdate.UTC(),
		gender,
		lookingFor,
		occupation,
		education,
		heightCM,
		eyeColor,
		zodiac,
		languages,
		goals,
		profileCompleted,
	); err != nil {
		return fmt.Errorf("save profile core: %w", err)
	}

	return nil
}

func (r *ProfileRepo) SetModerationStatus(ctx context.Context, userID int64, status string) error {
	if r.pool == nil {
		return nil
	}
	if userID <= 0 || status == "" {
		return fmt.Errorf("invalid moderation status payload")
	}

	const query = `
INSERT INTO profiles (
	user_id,
	display_name,
	moderation_status,
	updated_at
) VALUES ($1, '', $2, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	moderation_status = EXCLUDED.moderation_status,
	updated_at = NOW()
`

	if _, err := r.pool.Exec(ctx, query, userID, status); err != nil {
		return fmt.Errorf("set profile moderation status: %w", err)
	}

	return nil
}

func (r *ProfileRepo) ApplyModerationDecision(ctx context.Context, userID int64, status string, approved bool) error {
	if r.pool == nil {
		return nil
	}
	if userID <= 0 || status == "" {
		return fmt.Errorf("invalid moderation decision payload")
	}

	const query = `
INSERT INTO profiles (
	user_id,
	display_name,
	moderation_status,
	approved,
	updated_at
) VALUES ($1, '', $2, $3, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	moderation_status = EXCLUDED.moderation_status,
	approved = EXCLUDED.approved,
	updated_at = NOW()
`

	if _, err := r.pool.Exec(ctx, query, userID, status, approved); err != nil {
		return fmt.Errorf("apply moderation decision: %w", err)
	}

	return nil
}

func (r *ProfileRepo) GetModerationSnapshot(ctx context.Context, userID int64) (ProfileModerationSnapshot, error) {
	if r.pool == nil {
		return ProfileModerationSnapshot{}, fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 {
		return ProfileModerationSnapshot{}, fmt.Errorf("invalid user id")
	}

	var snapshot ProfileModerationSnapshot
	err := r.pool.QueryRow(ctx, `
SELECT moderation_status, approved
FROM profiles
WHERE user_id = $1
LIMIT 1
`, userID).Scan(&snapshot.Status, &snapshot.Approved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProfileModerationSnapshot{}, ErrProfileNotFound
		}
		return ProfileModerationSnapshot{}, fmt.Errorf("get moderation snapshot: %w", err)
	}

	return snapshot, nil
}

func (r *ProfileRepo) GetQueueSummary(ctx context.Context, userID int64) (ProfileQueueSummary, error) {
	if r.pool == nil {
		return ProfileQueueSummary{}, fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 {
		return ProfileQueueSummary{}, fmt.Errorf("invalid user id")
	}

	var summary ProfileQueueSummary
	err := r.pool.QueryRow(ctx, `
SELECT user_id, display_name, city_id, gender, looking_for, goals, birthdate, occupation, education
FROM profiles
WHERE user_id = $1
LIMIT 1
`, userID).Scan(
		&summary.UserID,
		&summary.DisplayName,
		&summary.CityID,
		&summary.Gender,
		&summary.LookingFor,
		&summary.Goals,
		&summary.Birthdate,
		&summary.Occupation,
		&summary.Education,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProfileQueueSummary{UserID: userID}, nil
		}
		return ProfileQueueSummary{}, fmt.Errorf("get profile queue summary: %w", err)
	}

	return summary, nil
}

func (r *ProfileRepo) ClearExactGeoOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if r.pool == nil {
		return 0, nil
	}

	tag, err := r.pool.Exec(ctx, `
UPDATE profiles
SET
	last_lat = NULL,
	last_lon = NULL,
	updated_at = NOW()
WHERE last_geo_at IS NOT NULL
  AND last_geo_at < $1
  AND (last_lat IS NOT NULL OR last_lon IS NOT NULL)
`, cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("clear exact geo older than cutoff: %w", err)
	}

	return tag.RowsAffected(), nil
}
