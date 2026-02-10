package adminhttp

import (
	"context"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/postgres"
)

type AuditRepo struct {
	client *Client
	db     *postgres.AuditRepo
	dual   bool
}

func NewAuditRepo(client *Client, db *postgres.AuditRepo, dual bool) *AuditRepo {
	return &AuditRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *AuditRepo) Save(ctx context.Context, entry model.Audit) error {
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/audit", entry, nil)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.Save(ctx, entry)
	}
	return err
}

func (r *AuditRepo) ListRecent(ctx context.Context, limit int) ([]model.Audit, error) {
	response := struct {
		Items []model.Audit `json:"items"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/audit/recent?limit="+intToString(limit), nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.ListRecent(ctx, limit)
	}
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}
