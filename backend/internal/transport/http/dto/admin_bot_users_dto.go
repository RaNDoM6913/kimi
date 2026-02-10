package dto

import "time"

type AdminBotLookupUserResponse struct {
	User AdminBotLookupUser `json:"user"`
}

type AdminBotLookupUser struct {
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

type AdminBotBanRequest struct {
	Reason string `json:"reason"`
}
