package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrFeedViewerNotFound = errors.New("feed viewer profile not found")
var ErrFeedCandidateNotFound = errors.New("feed candidate not found")

type FeedRepo struct {
	pool *pgxpool.Pool
}

func NewFeedRepo(pool *pgxpool.Pool) *FeedRepo {
	return &FeedRepo{pool: pool}
}

type FeedViewerContext struct {
	UserID     int64
	CityID     string
	Gender     string
	LookingFor string
	AgeMin     int
	AgeMax     int
	RadiusKM   int
	Goals      []string
	LastLat    *float64
	LastLon    *float64
}

type FeedQuery struct {
	ViewerUserID     int64
	ViewerCityID     string
	ViewerGender     string
	ViewerLookingFor string
	ViewerGoals      []string
	AgeMin           int
	AgeMax           int
	RadiusKM         int
	ViewerLat        *float64
	ViewerLon        *float64
	HasCursor        bool
	CursorPriority   int
	CursorCreatedAt  time.Time
	CursorUserID     int64
	Limit            int
	Now              time.Time
}

type CandidateProfileQuery struct {
	ViewerUserID    int64
	CandidateUserID int64
	Now             time.Time
}

type CandidateProfileRecord struct {
	UserID      int64
	DisplayName string
	Age         int
	Zodiac      string
	CityID      string
	City        string
	DistanceKM  *float64
	Bio         *string
	Occupation  string
	Education   string
	HeightCM    int
	EyeColor    string
	Languages   []string
	Goals       []string
	IsPlus      bool
}

type FeedCandidate struct {
	UserID        int64
	DisplayName   string
	CityID        string
	City          string
	Zodiac        string
	Birthdate     *time.Time
	PrimaryPhoto  string
	PrimaryGoal   string
	Age           int
	GoalsPriority int
	RankScore     *float64
	DistanceKM    *float64
	CreatedAt     time.Time
}

func (r *FeedRepo) GetViewerContext(ctx context.Context, userID int64) (FeedViewerContext, error) {
	if userID <= 0 {
		return FeedViewerContext{}, fmt.Errorf("invalid user id")
	}
	if r.pool == nil {
		return FeedViewerContext{}, ErrFeedViewerNotFound
	}

	var viewer FeedViewerContext
	err := r.pool.QueryRow(ctx, `
SELECT
	user_id,
	COALESCE(city_id, ''),
	COALESCE(gender, ''),
	COALESCE(looking_for, ''),
	age_min,
	age_max,
	radius_km,
	goals,
	last_lat,
	last_lon
FROM profiles
WHERE user_id = $1
LIMIT 1
`, userID).Scan(
		&viewer.UserID,
		&viewer.CityID,
		&viewer.Gender,
		&viewer.LookingFor,
		&viewer.AgeMin,
		&viewer.AgeMax,
		&viewer.RadiusKM,
		&viewer.Goals,
		&viewer.LastLat,
		&viewer.LastLon,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FeedViewerContext{}, ErrFeedViewerNotFound
		}
		return FeedViewerContext{}, fmt.Errorf("get feed viewer context: %w", err)
	}

	return viewer, nil
}

func (r *FeedRepo) ListCandidates(ctx context.Context, q FeedQuery) ([]FeedCandidate, error) {
	if q.ViewerUserID <= 0 {
		return nil, fmt.Errorf("invalid viewer id")
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Now.IsZero() {
		q.Now = time.Now().UTC()
	}
	if r.pool == nil {
		return []FeedCandidate{}, nil
	}

	lookingFor := strings.ToLower(strings.TrimSpace(q.ViewerLookingFor))
	viewerGender := strings.ToLower(strings.TrimSpace(q.ViewerGender))
	viewerCityID := strings.TrimSpace(q.ViewerCityID)
	applyCityFilter := viewerCityID != ""
	applyLookingFor := lookingFor != "" && lookingFor != "all" && lookingFor != "any" && lookingFor != "unknown"
	applyMutualLookingFor := viewerGender != "" && viewerGender != "all" && viewerGender != "any" && viewerGender != "unknown"
	applyRadius := q.ViewerLat != nil && q.ViewerLon != nil && q.RadiusKM > 0
	viewerGoals := normalizeGoals(q.ViewerGoals)
	hasGoals := len(viewerGoals) > 0
	cursorCreatedAt := q.CursorCreatedAt.UTC()
	if cursorCreatedAt.IsZero() {
		cursorCreatedAt = time.Unix(0, 0).UTC()
	}

	rows, err := r.pool.Query(ctx, `
SELECT
	p.user_id,
	p.display_name,
	COALESCE(p.city_id, ''),
	COALESCE(p.city, ''),
	COALESCE(p.zodiac, ''),
	p.birthdate,
	COALESCE(pm.s3_key, ''),
	COALESCE(p.goals[1], ''),
	DATE_PART('year', AGE($2::timestamptz, p.birthdate::timestamp))::int AS age,
	CASE
		WHEN $16::boolean = TRUE AND COALESCE(array_length(p.goals, 1), 0) > 0 AND p.goals && $15::text[]
		THEN 1 ELSE 0
	END AS goals_priority,
	CASE
		WHEN $16::boolean = TRUE AND COALESCE(array_length(p.goals, 1), 0) > 0 AND p.goals && $15::text[]
		THEN 1.0 ELSE 0.0
	END AS rank_score,
	CASE
		WHEN $11::boolean = TRUE AND p.last_lat IS NOT NULL AND p.last_lon IS NOT NULL
		THEN 6371.0 * ACOS(LEAST(1.0, GREATEST(-1.0,
			COS(RADIANS($12::float8)) * COS(RADIANS(p.last_lat)) * COS(RADIANS(p.last_lon) - RADIANS($13::float8))
			+ SIN(RADIANS($12::float8)) * SIN(RADIANS(p.last_lat))
		)))
		ELSE NULL
	END AS distance_km,
	p.created_at
FROM profiles p
LEFT JOIN LATERAL (
	SELECT m.s3_key
	FROM media m
	WHERE
		m.user_id = p.user_id
		AND m.kind = 'photo'
		AND m.status = 'active'
	ORDER BY m.position ASC, m.created_at ASC
	LIMIT 1
) pm ON TRUE
WHERE
	p.approved = TRUE
	AND p.user_id <> $1
	AND p.birthdate IS NOT NULL
	AND ($3::boolean = FALSE OR p.city_id = $4)
	AND ($5::boolean = FALSE OR LOWER(p.gender) = LOWER($6))
	AND (
		$7::boolean = FALSE
		OR LOWER(p.looking_for) IN ('all', 'any', 'unknown', '')
		OR LOWER(p.looking_for) = LOWER($8)
	)
	AND DATE_PART('year', AGE($2::timestamptz, p.birthdate::timestamp))::int BETWEEN $9 AND $10
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE b.actor_user_id = p.user_id
			AND b.target_user_id = $1
	)
	AND NOT EXISTS (
		SELECT 1
		FROM dislikes_state ds
		WHERE ds.actor_user_id = $1
			AND ds.target_user_id = p.user_id
			AND (
				COALESCE(ds.never_show, FALSE) = TRUE
				OR COALESCE(ds.hide_until, ds.until_at) > $2::timestamptz
			)
	)
	AND (
		$11::boolean = FALSE
		OR (
			p.last_lat IS NOT NULL
			AND p.last_lon IS NOT NULL
			AND (
				6371.0 * ACOS(LEAST(1.0, GREATEST(-1.0,
					COS(RADIANS($12::float8)) * COS(RADIANS(p.last_lat)) * COS(RADIANS(p.last_lon) - RADIANS($13::float8))
					+ SIN(RADIANS($12::float8)) * SIN(RADIANS(p.last_lat))
				)))
			) <= $14::float8
		)
	)
	AND (
		$17::boolean = FALSE
		OR (
			(
				CASE
					WHEN $16::boolean = TRUE AND COALESCE(array_length(p.goals, 1), 0) > 0 AND p.goals && $15::text[]
					THEN 1 ELSE 0
				END
			) < $18::int
			OR (
				(
					CASE
						WHEN $16::boolean = TRUE AND COALESCE(array_length(p.goals, 1), 0) > 0 AND p.goals && $15::text[]
						THEN 1 ELSE 0
					END
				) = $18::int
				AND (
					p.created_at < $19::timestamptz
					OR (p.created_at = $19::timestamptz AND p.user_id < $20::bigint)
				)
			)
		)
	)
ORDER BY goals_priority DESC, p.created_at DESC, p.user_id DESC
LIMIT $21
`,
		q.ViewerUserID,           // $1
		q.Now.UTC(),              // $2
		applyCityFilter,          // $3
		viewerCityID,             // $4
		applyLookingFor,          // $5
		lookingFor,               // $6
		applyMutualLookingFor,    // $7
		viewerGender,             // $8
		q.AgeMin,                 // $9
		q.AgeMax,                 // $10
		applyRadius,              // $11
		floatOrZero(q.ViewerLat), // $12
		floatOrZero(q.ViewerLon), // $13
		float64(q.RadiusKM),      // $14
		viewerGoals,              // $15
		hasGoals,                 // $16
		q.HasCursor,              // $17
		q.CursorPriority,         // $18
		cursorCreatedAt,          // $19
		q.CursorUserID,           // $20
		q.Limit,                  // $21
	)
	if err != nil {
		return nil, fmt.Errorf("list feed candidates: %w", err)
	}
	defer rows.Close()

	items := make([]FeedCandidate, 0, q.Limit)
	for rows.Next() {
		var item FeedCandidate
		var rankScore float64
		if err := rows.Scan(
			&item.UserID,
			&item.DisplayName,
			&item.CityID,
			&item.City,
			&item.Zodiac,
			&item.Birthdate,
			&item.PrimaryPhoto,
			&item.PrimaryGoal,
			&item.Age,
			&item.GoalsPriority,
			&rankScore,
			&item.DistanceKM,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan feed candidate: %w", err)
		}
		item.RankScore = &rankScore
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate feed candidates: %w", rows.Err())
	}

	return items, nil
}

func (r *FeedRepo) GetCandidateProfile(ctx context.Context, q CandidateProfileQuery) (CandidateProfileRecord, error) {
	if q.ViewerUserID <= 0 || q.CandidateUserID <= 0 {
		return CandidateProfileRecord{}, fmt.Errorf("invalid candidate profile query")
	}
	if q.Now.IsZero() {
		q.Now = time.Now().UTC()
	}
	if q.ViewerUserID == q.CandidateUserID {
		return CandidateProfileRecord{}, ErrFeedCandidateNotFound
	}
	if r.pool == nil {
		return CandidateProfileRecord{}, ErrFeedCandidateNotFound
	}

	var (
		record   CandidateProfileRecord
		bio      sql.NullString
		heightCM int16
		distance sql.NullFloat64
	)
	err := r.pool.QueryRow(ctx, `
SELECT
	p.user_id,
	COALESCE(NULLIF(BTRIM(p.display_name), ''), ''),
	COALESCE(DATE_PART('year', AGE($3::timestamptz, p.birthdate::timestamp))::int, 0),
	COALESCE(p.zodiac, ''),
	COALESCE(p.city_id, ''),
	COALESCE(p.city, ''),
	CASE
		WHEN vp.last_lat IS NOT NULL AND vp.last_lon IS NOT NULL
			AND p.last_lat IS NOT NULL AND p.last_lon IS NOT NULL
		THEN 6371.0 * ACOS(LEAST(1.0, GREATEST(-1.0,
			COS(RADIANS(vp.last_lat)) * COS(RADIANS(p.last_lat)) * COS(RADIANS(p.last_lon) - RADIANS(vp.last_lon))
			+ SIN(RADIANS(vp.last_lat)) * SIN(RADIANS(p.last_lat))
		)))
		ELSE NULL
	END AS distance_km,
	NULLIF(BTRIM(p.bio), '') AS bio,
	COALESCE(p.occupation, ''),
	COALESCE(p.education, ''),
	COALESCE(p.height_cm, 0)::smallint,
	COALESCE(p.eye_color, ''),
	COALESCE(p.languages, '{}'::text[]),
	COALESCE(p.goals, '{}'::text[]),
	COALESCE(e.plus_expires_at > $3::timestamptz, FALSE) AS is_plus
FROM profiles p
LEFT JOIN profiles vp ON vp.user_id = $1
LEFT JOIN entitlements e ON e.user_id = p.user_id
LEFT JOIN user_bans ub ON ub.user_id = (
	'00000000-0000-0000-0000-' ||
	LPAD(TO_HEX((p.user_id & 281474976710655)::bigint), 12, '0')
)::uuid
WHERE
	p.user_id = $2
	AND p.approved = TRUE
	AND p.birthdate IS NOT NULL
	AND COALESCE(ub.banned, FALSE) = FALSE
	AND NOT EXISTS (
		SELECT 1
		FROM blocks b
		WHERE
			(b.actor_user_id = $1 AND b.target_user_id = p.user_id)
			OR (b.actor_user_id = p.user_id AND b.target_user_id = $1)
	)
	AND NOT EXISTS (
		SELECT 1
		FROM dislikes_state ds
		WHERE
			ds.actor_user_id = $1
			AND ds.target_user_id = p.user_id
			AND (
				COALESCE(ds.never_show, FALSE) = TRUE
				OR COALESCE(ds.hide_until, ds.until_at) > $3::timestamptz
			)
	)
LIMIT 1
`, q.ViewerUserID, q.CandidateUserID, q.Now.UTC()).Scan(
		&record.UserID,
		&record.DisplayName,
		&record.Age,
		&record.Zodiac,
		&record.CityID,
		&record.City,
		&distance,
		&bio,
		&record.Occupation,
		&record.Education,
		&heightCM,
		&record.EyeColor,
		&record.Languages,
		&record.Goals,
		&record.IsPlus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CandidateProfileRecord{}, ErrFeedCandidateNotFound
		}
		return CandidateProfileRecord{}, fmt.Errorf("get candidate profile: %w", err)
	}

	record.HeightCM = int(heightCM)
	if bio.Valid {
		value := strings.TrimSpace(bio.String)
		if value != "" {
			record.Bio = &value
		}
	}
	if distance.Valid {
		value := distance.Float64
		record.DistanceKM = &value
	}

	return record, nil
}

func normalizeGoals(goals []string) []string {
	if len(goals) == 0 {
		return nil
	}

	out := make([]string, 0, len(goals))
	seen := make(map[string]struct{}, len(goals))
	for _, goal := range goals {
		value := strings.ToLower(strings.TrimSpace(goal))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func floatOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
