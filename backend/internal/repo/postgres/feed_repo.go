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

var ErrFeedViewerNotFound = errors.New("feed viewer profile not found")

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

type FeedCandidate struct {
	UserID        int64
	DisplayName   string
	CityID        string
	City          string
	Age           int
	GoalsPriority int
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
	DATE_PART('year', AGE($2::timestamptz, p.birthdate::timestamp))::int AS age,
	CASE
		WHEN $16::boolean = TRUE AND COALESCE(array_length(p.goals, 1), 0) > 0 AND p.goals && $15::text[]
		THEN 1 ELSE 0
	END AS goals_priority,
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
		if err := rows.Scan(
			&item.UserID,
			&item.DisplayName,
			&item.CityID,
			&item.City,
			&item.Age,
			&item.GoalsPriority,
			&item.DistanceKM,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan feed candidate: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate feed candidates: %w", rows.Err())
	}

	return items, nil
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
