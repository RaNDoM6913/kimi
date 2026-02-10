package adminhttp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/postgres"
)

type UsersLookupRepo struct {
	client *Client
	db     *postgres.UsersLookupRepo
	dual   bool
}

func NewUsersLookupRepo(client *Client, db *postgres.UsersLookupRepo, dual bool) *UsersLookupRepo {
	return &UsersLookupRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *UsersLookupRepo) FindUser(ctx context.Context, query string) (model.LookupUser, error) {
	response := lookupUserEnvelope{}
	err := r.client.DoJSON(ctx, http.MethodGet, "/admin/bot/lookup/user?query="+urlQueryEscape(query), nil, &response)
	if shouldFallbackWithNotFound(r.dual, err) && r.db != nil {
		return r.db.FindUser(ctx, query)
	}
	if err != nil {
		return model.LookupUser{}, err
	}

	user := response.toModel()
	if user.UserID == 0 && r.db != nil && r.dual {
		return r.db.FindUser(ctx, query)
	}
	return user, nil
}

func (r *UsersLookupRepo) FindByUserID(ctx context.Context, userID int64) (model.LookupUser, error) {
	query := int64ToString(userID)
	response := lookupUserEnvelope{}
	err := r.client.DoJSON(ctx, http.MethodGet, "/admin/bot/lookup/user?query="+urlQueryEscape(query), nil, &response)
	if shouldFallbackWithNotFound(r.dual, err) && r.db != nil {
		return r.db.FindByUserID(ctx, userID)
	}
	if err != nil {
		return model.LookupUser{}, err
	}

	user := response.toModel()
	if user.UserID == 0 && r.db != nil && r.dual {
		return r.db.FindByUserID(ctx, userID)
	}
	return user, nil
}

func (r *UsersLookupRepo) InsertAction(ctx context.Context, action model.BotLookupAction) error {
	err := r.client.DoJSON(ctx, http.MethodPost, "/admin/bot/lookup/action", action, nil)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.InsertAction(ctx, action)
	}
	return err
}

func (r *UsersLookupRepo) ForceReview(ctx context.Context, userID int64) error {
	request := map[string]interface{}{
		"user_id": userID,
	}
	err := r.client.DoJSON(ctx, http.MethodPost, "/admin/bot/users/"+int64ToString(userID)+"/force-review", request, nil)
	if shouldFallbackWithNotFound(r.dual, err) && r.db != nil {
		return r.db.ForceReview(ctx, userID)
	}
	return err
}

type lookupUserEnvelope struct {
	User lookupUserDTO `json:"user"`
	Item lookupUserDTO `json:"item"`

	UserID           int64      `json:"user_id"`
	TGID             int64      `json:"tg_id"`
	Username         string     `json:"username"`
	CityID           string     `json:"city_id"`
	Birthdate        *time.Time `json:"birthdate"`
	Age              int        `json:"age"`
	Gender           string     `json:"gender"`
	LookingFor       string     `json:"looking_for"`
	Goals            []string   `json:"goals"`
	Languages        []string   `json:"languages"`
	Occupation       string     `json:"occupation"`
	Education        string     `json:"education"`
	ModerationStatus string     `json:"moderation_status"`
	Approved         bool       `json:"approved"`
	PhotoKeys        []string   `json:"photo_keys"`
	CircleKey        string     `json:"circle_key"`
	PhotoURLs        []string   `json:"photo_urls"`
	CircleURL        string     `json:"circle_url"`
	PlusExpiresAt    *time.Time `json:"plus_expires_at"`
	BoostUntil       *time.Time `json:"boost_until"`
	SuperlikeCredits int        `json:"superlike_credits"`
	RevealCredits    int        `json:"reveal_credits"`
	LikeTokens       int        `json:"like_tokens"`
	IsBanned         bool       `json:"is_banned"`
	BanReason        string     `json:"ban_reason"`
}

func (e lookupUserEnvelope) toModel() model.LookupUser {
	dto := e.pickDTO()
	user := dto.toModel()
	if user.UserID != 0 {
		return user
	}

	return model.LookupUser{
		UserID:           e.UserID,
		TGID:             e.TGID,
		Username:         strings.TrimSpace(e.Username),
		CityID:           strings.TrimSpace(e.CityID),
		Birthdate:        e.Birthdate,
		Age:              e.Age,
		Gender:           strings.TrimSpace(e.Gender),
		LookingFor:       strings.TrimSpace(e.LookingFor),
		Goals:            cloneStrings(e.Goals),
		Languages:        cloneStrings(e.Languages),
		Occupation:       strings.TrimSpace(e.Occupation),
		Education:        strings.TrimSpace(e.Education),
		ModerationStatus: strings.TrimSpace(e.ModerationStatus),
		Approved:         e.Approved,
		PhotoKeys:        cloneStrings(e.PhotoKeys),
		CircleKey:        strings.TrimSpace(e.CircleKey),
		PhotoURLs:        cloneStrings(e.PhotoURLs),
		CircleURL:        strings.TrimSpace(e.CircleURL),
		PlusExpiresAt:    e.PlusExpiresAt,
		BoostUntil:       e.BoostUntil,
		SuperlikeCredits: e.SuperlikeCredits,
		RevealCredits:    e.RevealCredits,
		LikeTokens:       e.LikeTokens,
		IsBanned:         e.IsBanned,
		BanReason:        strings.TrimSpace(e.BanReason),
	}
}

func (e lookupUserEnvelope) pickDTO() lookupUserDTO {
	if e.User.UserID != 0 {
		return e.User
	}
	if e.Item.UserID != 0 {
		return e.Item
	}
	return lookupUserDTO{}
}

type lookupUserDTO struct {
	UserID           int64      `json:"user_id"`
	TGID             int64      `json:"tg_id"`
	Username         string     `json:"username"`
	CityID           string     `json:"city_id"`
	Birthdate        *time.Time `json:"birthdate"`
	Age              int        `json:"age"`
	Gender           string     `json:"gender"`
	LookingFor       string     `json:"looking_for"`
	Goals            []string   `json:"goals"`
	Languages        []string   `json:"languages"`
	Occupation       string     `json:"occupation"`
	Education        string     `json:"education"`
	ModerationStatus string     `json:"moderation_status"`
	Approved         bool       `json:"approved"`
	PhotoKeys        []string   `json:"photo_keys"`
	CircleKey        string     `json:"circle_key"`
	PhotoURLs        []string   `json:"photo_urls"`
	CircleURL        string     `json:"circle_url"`
	PlusExpiresAt    *time.Time `json:"plus_expires_at"`
	BoostUntil       *time.Time `json:"boost_until"`
	SuperlikeCredits int        `json:"superlike_credits"`
	RevealCredits    int        `json:"reveal_credits"`
	LikeTokens       int        `json:"like_tokens"`
	IsBanned         bool       `json:"is_banned"`
	BanReason        string     `json:"ban_reason"`
}

func (dto lookupUserDTO) toModel() model.LookupUser {
	return model.LookupUser{
		UserID:           dto.UserID,
		TGID:             dto.TGID,
		Username:         strings.TrimSpace(dto.Username),
		CityID:           strings.TrimSpace(dto.CityID),
		Birthdate:        dto.Birthdate,
		Age:              dto.Age,
		Gender:           strings.TrimSpace(dto.Gender),
		LookingFor:       strings.TrimSpace(dto.LookingFor),
		Goals:            cloneStrings(dto.Goals),
		Languages:        cloneStrings(dto.Languages),
		Occupation:       strings.TrimSpace(dto.Occupation),
		Education:        strings.TrimSpace(dto.Education),
		ModerationStatus: strings.TrimSpace(dto.ModerationStatus),
		Approved:         dto.Approved,
		PhotoKeys:        cloneStrings(dto.PhotoKeys),
		CircleKey:        strings.TrimSpace(dto.CircleKey),
		PhotoURLs:        cloneStrings(dto.PhotoURLs),
		CircleURL:        strings.TrimSpace(dto.CircleURL),
		PlusExpiresAt:    dto.PlusExpiresAt,
		BoostUntil:       dto.BoostUntil,
		SuperlikeCredits: dto.SuperlikeCredits,
		RevealCredits:    dto.RevealCredits,
		LikeTokens:       dto.LikeTokens,
		IsBanned:         dto.IsBanned,
		BanReason:        strings.TrimSpace(dto.BanReason),
	}
}
