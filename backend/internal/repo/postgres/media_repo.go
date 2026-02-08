package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
)

type MediaRepo struct {
	pool *pgxpool.Pool
}

type CircleRecord struct {
	ID        int64
	ObjectKey string
	CreatedAt time.Time
}

type MediaAssetRecord struct {
	ID        int64
	Position  int
	ObjectKey string
	CreatedAt time.Time
}

func NewMediaRepo(pool *pgxpool.Pool) *MediaRepo {
	return &MediaRepo{pool: pool}
}

func (r *MediaRepo) CreatePhoto(ctx context.Context, userID int64, objectKey string) (mediasvc.PhotoRecord, error) {
	if r.pool == nil {
		return mediasvc.PhotoRecord{}, fmt.Errorf("postgres pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mediasvc.PhotoRecord{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
SELECT position
FROM media
WHERE user_id = $1 AND kind = 'photo' AND status = 'active'
ORDER BY position
FOR UPDATE
`, userID)
	if err != nil {
		return mediasvc.PhotoRecord{}, fmt.Errorf("query photo positions: %w", err)
	}

	positions := map[int]struct{}{}
	for rows.Next() {
		var position int
		if err := rows.Scan(&position); err != nil {
			rows.Close()
			return mediasvc.PhotoRecord{}, fmt.Errorf("scan photo position: %w", err)
		}
		positions[position] = struct{}{}
	}
	rows.Close()

	if len(positions) >= mediasvc.MaxActivePhotos() {
		return mediasvc.PhotoRecord{}, mediasvc.ErrPhotoLimitReached
	}

	position := nextPosition(positions)
	if position == 0 {
		return mediasvc.PhotoRecord{}, mediasvc.ErrPhotoLimitReached
	}

	var record mediasvc.PhotoRecord
	err = tx.QueryRow(ctx, `
INSERT INTO media (user_id, kind, s3_key, position, status, created_at, updated_at)
VALUES ($1, 'photo', $2, $3, 'active', NOW(), NOW())
RETURNING id, position, s3_key, created_at
`, userID, objectKey, position).Scan(&record.ID, &record.Position, &record.ObjectKey, &record.CreatedAt)
	if err != nil {
		return mediasvc.PhotoRecord{}, fmt.Errorf("insert media photo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return mediasvc.PhotoRecord{}, fmt.Errorf("commit transaction: %w", err)
	}

	return record, nil
}

func (r *MediaRepo) ListActivePhotos(ctx context.Context, userID int64) ([]mediasvc.PhotoRecord, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}

	rows, err := r.pool.Query(ctx, `
SELECT id, position, s3_key, created_at
FROM media
WHERE user_id = $1 AND kind = 'photo' AND status = 'active'
ORDER BY position ASC, created_at ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list active photos: %w", err)
	}
	defer rows.Close()

	photos := make([]mediasvc.PhotoRecord, 0)
	for rows.Next() {
		var record mediasvc.PhotoRecord
		if err := rows.Scan(&record.ID, &record.Position, &record.ObjectKey, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan active photo: %w", err)
		}
		photos = append(photos, record)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate active photos: %w", rows.Err())
	}

	return photos, nil
}

func (r *MediaRepo) CreateCircle(ctx context.Context, userID int64, objectKey string) (CircleRecord, error) {
	if r.pool == nil {
		return CircleRecord{}, fmt.Errorf("postgres pool is nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return CircleRecord{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
SELECT position
FROM media
WHERE user_id = $1 AND kind = 'circle'
ORDER BY position
FOR UPDATE
`, userID)
	if err != nil {
		return CircleRecord{}, fmt.Errorf("query circle positions: %w", err)
	}

	maxPosition := 0
	for rows.Next() {
		var position int
		if err := rows.Scan(&position); err != nil {
			rows.Close()
			return CircleRecord{}, fmt.Errorf("scan circle position: %w", err)
		}
		if position > maxPosition {
			maxPosition = position
		}
	}
	rows.Close()

	nextPos := maxPosition + 1
	if nextPos <= 0 {
		nextPos = 1
	}

	var record CircleRecord
	err = tx.QueryRow(ctx, `
INSERT INTO media (user_id, kind, s3_key, position, status, created_at, updated_at)
VALUES ($1, 'circle', $2, $3, 'pending', NOW(), NOW())
RETURNING id, s3_key, created_at
`, userID, objectKey, nextPos).Scan(&record.ID, &record.ObjectKey, &record.CreatedAt)
	if err != nil {
		return CircleRecord{}, fmt.Errorf("insert circle media: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CircleRecord{}, fmt.Errorf("commit transaction: %w", err)
	}

	return record, nil
}

func (r *MediaRepo) ListUserPhotos(ctx context.Context, userID int64, limit int) ([]MediaAssetRecord, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	if limit <= 0 {
		limit = 3
	}

	rows, err := r.pool.Query(ctx, `
SELECT id, position, s3_key, created_at
FROM media
WHERE user_id = $1 AND kind = 'photo' AND status = 'active'
ORDER BY position ASC, created_at ASC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user photos: %w", err)
	}
	defer rows.Close()

	items := make([]MediaAssetRecord, 0)
	for rows.Next() {
		var item MediaAssetRecord
		if err := rows.Scan(&item.ID, &item.Position, &item.ObjectKey, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user photo: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate user photos: %w", rows.Err())
	}

	return items, nil
}

func (r *MediaRepo) GetLatestCircle(ctx context.Context, userID int64) (*MediaAssetRecord, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}

	var item MediaAssetRecord
	err := r.pool.QueryRow(ctx, `
SELECT id, position, s3_key, created_at
FROM media
WHERE user_id = $1 AND kind = 'circle'
ORDER BY created_at DESC, id DESC
LIMIT 1
`, userID).Scan(&item.ID, &item.Position, &item.ObjectKey, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest circle: %w", err)
	}

	return &item, nil
}

func (r *MediaRepo) ListCirclesOlderThan(ctx context.Context, olderThan time.Time) ([]CircleRecord, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}

	rows, err := r.pool.Query(ctx, `
SELECT id, s3_key, created_at
FROM media
WHERE kind = 'circle' AND created_at < $1
ORDER BY created_at ASC
`, olderThan.UTC())
	if err != nil {
		return nil, fmt.Errorf("list old circles: %w", err)
	}
	defer rows.Close()

	circles := make([]CircleRecord, 0)
	for rows.Next() {
		var rec CircleRecord
		if err := rows.Scan(&rec.ID, &rec.ObjectKey, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan circle record: %w", err)
		}
		circles = append(circles, rec)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate circle records: %w", rows.Err())
	}

	return circles, nil
}

func (r *MediaRepo) DeleteCircle(ctx context.Context, mediaID int64) error {
	if r.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	if mediaID <= 0 {
		return nil
	}

	if _, err := r.pool.Exec(ctx, `
DELETE FROM media
WHERE id = $1 AND kind = 'circle'
`, mediaID); err != nil {
		return fmt.Errorf("delete circle media: %w", err)
	}

	return nil
}

func nextPosition(occupied map[int]struct{}) int {
	for i := 1; i <= mediasvc.MaxActivePhotos(); i++ {
		if _, ok := occupied[i]; !ok {
			return i
		}
	}
	return 0
}
