package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ExportsQueueRepo struct {
	db *sql.DB
}

func NewExportsQueueRepo(db *sql.DB) *ExportsQueueRepo {
	return &ExportsQueueRepo{db: db}
}

func (r *ExportsQueueRepo) Enqueue(ctx context.Context, kind string, payload json.RawMessage, status string, createdAt time.Time) error {
	if r.db == nil {
		return nil
	}

	kind = strings.TrimSpace(kind)
	status = strings.TrimSpace(status)
	if kind == "" {
		return fmt.Errorf("kind is required")
	}
	if status == "" {
		return fmt.Errorf("status is required")
	}
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_exports_queue (kind, payload, status, created_at)
		VALUES ($1, $2, $3, $4)
	`, kind, string(payload), status, createdAt)
	if err != nil {
		return fmt.Errorf("enqueue export item: %w", err)
	}
	return nil
}
