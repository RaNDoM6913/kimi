package adminhttp

import (
	"context"

	"bot_moderator/internal/repo/postgres"
)

type SystemRepo struct {
	client *Client
	db     *postgres.SystemRepo
	dual   bool
}

func NewSystemRepo(client *Client, db *postgres.SystemRepo, dual bool) *SystemRepo {
	return &SystemRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *SystemRepo) GetRegistrationEnabled(ctx context.Context) (bool, error) {
	response := struct {
		Enabled             *bool `json:"enabled"`
		ValueBool           *bool `json:"value_bool"`
		RegistrationEnabled *bool `json:"registration_enabled"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/system/registration", nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GetRegistrationEnabled(ctx)
	}
	if err != nil {
		return false, err
	}

	switch {
	case response.Enabled != nil:
		return *response.Enabled, nil
	case response.ValueBool != nil:
		return *response.ValueBool, nil
	case response.RegistrationEnabled != nil:
		return *response.RegistrationEnabled, nil
	default:
		return true, nil
	}
}

func (r *SystemRepo) ToggleRegistration(ctx context.Context, updatedByTGID int64) (bool, error) {
	request := map[string]interface{}{
		"updated_by_tg_id": updatedByTGID,
	}
	response := struct {
		Enabled             *bool `json:"enabled"`
		ValueBool           *bool `json:"value_bool"`
		RegistrationEnabled *bool `json:"registration_enabled"`
	}{}
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/system/registration/toggle", request, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.ToggleRegistration(ctx, updatedByTGID)
	}
	if err != nil {
		return false, err
	}

	switch {
	case response.Enabled != nil:
		return *response.Enabled, nil
	case response.ValueBool != nil:
		return *response.ValueBool, nil
	case response.RegistrationEnabled != nil:
		return *response.RegistrationEnabled, nil
	default:
		return true, nil
	}
}

func (r *SystemRepo) GetUsersCount(ctx context.Context) (int64, int64, error) {
	response := struct {
		Total        int64 `json:"total"`
		Approved     int64 `json:"approved"`
		UsersTotal   int64 `json:"users_total"`
		ApprovedOnly int64 `json:"approved_only"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/system/users-count", nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GetUsersCount(ctx)
	}
	if err != nil {
		return 0, 0, err
	}

	total := response.Total
	if total == 0 && response.UsersTotal != 0 {
		total = response.UsersTotal
	}
	approved := response.Approved
	if approved == 0 && response.ApprovedOnly != 0 {
		approved = response.ApprovedOnly
	}
	return total, approved, nil
}
