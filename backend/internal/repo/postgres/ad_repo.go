package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdRepo struct {
	pool *pgxpool.Pool
}

type AdCardRecord struct {
	ID       int64
	Kind     string
	Title    string
	AssetURL string
	ClickURL string
	Priority int
}

func NewAdRepo(pool *pgxpool.Pool) *AdRepo {
	return &AdRepo{pool: pool}
}

func (r *AdRepo) ListActive(ctx context.Context, cityID string, limit int, at time.Time) ([]AdCardRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if r.pool == nil {
		return []AdCardRecord{}, nil
	}

	rows, err := r.pool.Query(ctx, `
SELECT
	id,
	COALESCE(kind, 'IMAGE'),
	COALESCE(title, ''),
	asset_url,
	click_url,
	priority
FROM ads
WHERE
	is_active = TRUE
	AND (starts_at IS NULL OR starts_at <= $1::timestamptz)
	AND (ends_at IS NULL OR ends_at > $1::timestamptz)
	AND (
		$2::text = ''
		OR COALESCE(city_id, '') = ''
		OR city_id = $2::text
	)
ORDER BY priority DESC, id DESC
LIMIT $3
`, at.UTC(), strings.TrimSpace(cityID), limit)
	if err != nil {
		return nil, fmt.Errorf("list active ads: %w", err)
	}
	defer rows.Close()

	items := make([]AdCardRecord, 0, limit)
	for rows.Next() {
		var item AdCardRecord
		if err := rows.Scan(
			&item.ID,
			&item.Kind,
			&item.Title,
			&item.AssetURL,
			&item.ClickURL,
			&item.Priority,
		); err != nil {
			return nil, fmt.Errorf("scan active ad: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate active ads: %w", rows.Err())
	}

	return items, nil
}

func (r *AdRepo) ExistsActive(ctx context.Context, adID int64, at time.Time) (bool, error) {
	if adID <= 0 {
		return false, fmt.Errorf("invalid ad id")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if r.pool == nil {
		return false, fmt.Errorf("postgres pool is nil")
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM ads
	WHERE
		id = $1
		AND is_active = TRUE
		AND (starts_at IS NULL OR starts_at <= $2::timestamptz)
		AND (ends_at IS NULL OR ends_at > $2::timestamptz)
)
`, adID, at.UTC()).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active ad exists: %w", err)
	}
	return exists, nil
}

func (r *AdRepo) InsertEvent(ctx context.Context, adID, userID int64, eventType string, meta map[string]any) error {
	if adID <= 0 {
		return fmt.Errorf("invalid ad id")
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if strings.TrimSpace(eventType) == "" {
		return fmt.Errorf("event type is required")
	}
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}

	payload := "{}"
	if len(meta) > 0 {
		raw, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("marshal ad event meta: %w", err)
		}
		payload = string(raw)
	}

	if _, err := r.pool.Exec(ctx, `
INSERT INTO ad_events (
	ad_id,
	user_id,
	event_type,
	meta,
	created_at
) VALUES (
	$1,
	$2,
	$3,
	$4::jsonb,
	NOW()
)
`, adID, userID, strings.ToUpper(strings.TrimSpace(eventType)), payload); err != nil {
		return fmt.Errorf("insert ad event: %w", err)
	}
	return nil
}
