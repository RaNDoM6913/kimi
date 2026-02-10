package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DailyMetricsRepo struct {
	pool *pgxpool.Pool
}

type DailyMetricsDelta struct {
	Likes      int
	Dislikes   int
	SuperLikes int
	Matches    int
	Reports    int
	Approved   int
}

type DailyMetricRow struct {
	DayKey     time.Time
	CityID     string
	Gender     string
	LookingFor string
	Likes      int
	Dislikes   int
	SuperLikes int
	Matches    int
	Reports    int
	Approved   int
}

func NewDailyMetricsRepo(pool *pgxpool.Pool) *DailyMetricsRepo {
	return &DailyMetricsRepo{pool: pool}
}

func (r *DailyMetricsRepo) Increment(ctx context.Context, userID int64, at time.Time, delta DailyMetricsDelta) error {
	if r.pool == nil {
		return nil
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if delta.isZero() {
		return nil
	}

	_, err := r.pool.Exec(ctx, `
INSERT INTO daily_metrics (
	day_key,
	city_id,
	gender,
	looking_for,
	likes,
	dislikes,
	superlikes,
	matches,
	reports,
	approved,
	updated_at
)
SELECT
	$2::date,
	COALESCE(NULLIF(TRIM(p.city_id), ''), 'unknown'),
	COALESCE(NULLIF(LOWER(TRIM(p.gender)), ''), 'unknown'),
	COALESCE(NULLIF(LOWER(TRIM(p.looking_for)), ''), 'unknown'),
	$3::int,
	$4::int,
	$5::int,
	$6::int,
	$7::int,
	$8::int,
	NOW()
FROM profiles p
WHERE p.user_id = $1
UNION ALL
SELECT
	$2::date,
	'unknown',
	'unknown',
	'unknown',
	$3::int,
	$4::int,
	$5::int,
	$6::int,
	$7::int,
	$8::int,
	NOW()
WHERE NOT EXISTS (
	SELECT 1 FROM profiles p WHERE p.user_id = $1
)
ON CONFLICT (day_key, city_id, gender, looking_for) DO UPDATE SET
	likes = daily_metrics.likes + EXCLUDED.likes,
	dislikes = daily_metrics.dislikes + EXCLUDED.dislikes,
	superlikes = daily_metrics.superlikes + EXCLUDED.superlikes,
	matches = daily_metrics.matches + EXCLUDED.matches,
	reports = daily_metrics.reports + EXCLUDED.reports,
	approved = daily_metrics.approved + EXCLUDED.approved,
	updated_at = NOW()
`,
		userID,
		at.UTC().Format("2006-01-02"),
		delta.Likes,
		delta.Dislikes,
		delta.SuperLikes,
		delta.Matches,
		delta.Reports,
		delta.Approved,
	)
	if err != nil {
		return fmt.Errorf("increment daily metrics: %w", err)
	}

	return nil
}

func (r *DailyMetricsRepo) ListDaily(ctx context.Context, from, to time.Time) ([]DailyMetricRow, error) {
	if r.pool == nil {
		return []DailyMetricRow{}, nil
	}
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("from/to are required")
	}

	rows, err := r.pool.Query(ctx, `
SELECT
	day_key,
	city_id,
	gender,
	looking_for,
	likes,
	dislikes,
	superlikes,
	matches,
	reports,
	approved
FROM daily_metrics
WHERE day_key BETWEEN $1::date AND $2::date
ORDER BY day_key ASC, city_id ASC, gender ASC, looking_for ASC
`, from.UTC().Format("2006-01-02"), to.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("list daily metrics: %w", err)
	}
	defer rows.Close()

	items := make([]DailyMetricRow, 0)
	for rows.Next() {
		var item DailyMetricRow
		if err := rows.Scan(
			&item.DayKey,
			&item.CityID,
			&item.Gender,
			&item.LookingFor,
			&item.Likes,
			&item.Dislikes,
			&item.SuperLikes,
			&item.Matches,
			&item.Reports,
			&item.Approved,
		); err != nil {
			return nil, fmt.Errorf("scan daily metric row: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate daily metrics rows: %w", rows.Err())
	}

	return items, nil
}

func (d DailyMetricsDelta) isZero() bool {
	return d.Likes == 0 && d.Dislikes == 0 && d.SuperLikes == 0 && d.Matches == 0 && d.Reports == 0 && d.Approved == 0
}
