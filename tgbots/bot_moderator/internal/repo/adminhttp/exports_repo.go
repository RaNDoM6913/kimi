package adminhttp

import (
	"context"
	"encoding/json"
	"time"

	"bot_moderator/internal/repo/postgres"
)

type ExportsQueueRepo struct {
	client *Client
	db     *postgres.ExportsQueueRepo
	dual   bool
}

func NewExportsQueueRepo(client *Client, db *postgres.ExportsQueueRepo, dual bool) *ExportsQueueRepo {
	return &ExportsQueueRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *ExportsQueueRepo) Enqueue(ctx context.Context, kind string, payload json.RawMessage, status string, createdAt time.Time) error {
	request := map[string]interface{}{
		"kind":       kind,
		"payload":    payload,
		"status":     status,
		"created_at": createdAt,
	}
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/exports/queue", request, nil)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.Enqueue(ctx, kind, payload, status, createdAt)
	}
	return err
}
