package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

type EventWriteRecord struct {
	Name       string
	OccurredAt time.Time
	Props      map[string]any
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) InsertBatch(ctx context.Context, userID *int64, events []EventWriteRecord) error {
	if len(events) == 0 {
		return nil
	}
	if r.pool == nil {
		return nil
	}

	const query = `
INSERT INTO events (
	user_id,
	name,
	payload,
	occurred_at,
	created_at
) VALUES (
	$1,
	$2,
	$3::jsonb,
	$4,
	NOW()
)
`

	batch := &pgx.Batch{}
	for _, event := range events {
		payload, err := json.Marshal(event.Props)
		if err != nil {
			return fmt.Errorf("marshal event props: %w", err)
		}

		var uid any
		if userID != nil && *userID > 0 {
			uid = *userID
		}

		occurredAt := event.OccurredAt.UTC()
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		batch.Queue(query, uid, event.Name, string(payload), occurredAt)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(events); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert event batch item #%d: %w", i, err)
		}
	}

	return nil
}
