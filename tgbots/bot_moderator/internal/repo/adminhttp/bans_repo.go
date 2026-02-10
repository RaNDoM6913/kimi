package adminhttp

import (
	"context"
	"net/http"

	"bot_moderator/internal/repo/postgres"
)

type BansRepo struct {
	client *Client
	db     *postgres.BansRepo
	dual   bool
}

func NewBansRepo(client *Client, db *postgres.BansRepo, dual bool) *BansRepo {
	return &BansRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *BansRepo) Ban(ctx context.Context, userID int64, reason string, updatedByTGID int64) error {
	request := map[string]interface{}{
		"user_id":          userID,
		"reason":           reason,
		"updated_by_tg_id": updatedByTGID,
	}
	err := r.client.DoJSON(ctx, http.MethodPost, "/admin/bot/users/"+int64ToString(userID)+"/ban", request, nil)
	if shouldFallbackWithNotFound(r.dual, err) && r.db != nil {
		return r.db.Ban(ctx, userID, reason, updatedByTGID)
	}
	return err
}

func (r *BansRepo) Unban(ctx context.Context, userID int64, updatedByTGID int64) error {
	request := map[string]interface{}{
		"user_id":          userID,
		"updated_by_tg_id": updatedByTGID,
	}
	err := r.client.DoJSON(ctx, http.MethodPost, "/admin/bot/users/"+int64ToString(userID)+"/unban", request, nil)
	if shouldFallbackWithNotFound(r.dual, err) && r.db != nil {
		return r.db.Unban(ctx, userID, updatedByTGID)
	}
	return err
}

func (r *BansRepo) GetBanState(ctx context.Context, userID int64) (bool, string, error) {
	response := struct {
		Banned bool   `json:"banned"`
		Reason string `json:"reason"`
	}{}
	err := r.client.DoJSON(ctx, http.MethodGet, "/admin/bot/bans/state?user_id="+int64ToString(userID), nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GetBanState(ctx, userID)
	}
	if err != nil {
		return false, "", err
	}
	return response.Banned, response.Reason, nil
}
