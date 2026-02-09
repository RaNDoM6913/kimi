package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
)

type AuditRepo struct {
	db *sql.DB
}

func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Save(ctx context.Context, entry model.Audit) error {
	if r.db == nil {
		return nil
	}

	payload := entry.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_audit (actor_tg_id, action, payload, created_at)
		VALUES ($1, $2, $3, $4)
	`, entry.ActorTGID, string(entry.Action), string(payload), entry.CreatedAt)
	return err
}

func (r *AuditRepo) ListRecent(ctx context.Context, limit int) ([]model.Audit, error) {
	if r.db == nil {
		return []model.Audit{}, nil
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, actor_tg_id, action, payload, created_at
		FROM bot_audit
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent bot audit: %w", err)
	}
	defer rows.Close()

	result := make([]model.Audit, 0, limit)
	for rows.Next() {
		var entry model.Audit
		var action string
		var payload []byte
		if err := rows.Scan(&entry.ID, &entry.ActorTGID, &action, &payload, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bot audit row: %w", err)
		}
		entry.Action = enums.AuditAction(action)
		entry.Payload = json.RawMessage(payload)
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bot audit rows: %w", err)
	}

	return result, nil
}
