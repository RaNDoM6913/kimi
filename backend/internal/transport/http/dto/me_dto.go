package dto

import "time"

type MeResponse struct {
	User             MeUserPublicResponse    `json:"user"`
	ModerationStatus string                  `json:"moderation_status"`
	Entitlements     MeEntitlementsResponse  `json:"entitlements"`
	Quota            MeQuotaSnapshotResponse `json:"quota"`
	AntiAbuseState   MeAntiAbuseState        `json:"antiabuse_state"`
}

type MeUserPublicResponse struct {
	ID        int64      `json:"id"`
	TGID      int64      `json:"tg_id"`
	Username  string     `json:"username"`
	Role      string     `json:"role"`
	IsPlus    bool       `json:"is_plus"`
	PlusUntil *time.Time `json:"plus_until"`
	CityID    string     `json:"city_id"`
}

type MeEntitlementsResponse struct {
	SuperLikeCredits      int        `json:"superlike_credits"`
	BoostCredits          int        `json:"boost_credits"`
	RevealCredits         int        `json:"reveal_credits"`
	MessageWoMatchCredits int        `json:"message_wo_match_credits"`
	IncognitoUntil        *time.Time `json:"incognito_until"`
}

type MeQuotaSnapshotResponse struct {
	LikesLeft         int       `json:"likes_left"`
	ResetAt           time.Time `json:"reset_at"`
	TooFastRetryAfter *int64    `json:"too_fast_retry_after"`
}

type MeAntiAbuseState struct {
	RiskScore     float64    `json:"risk_score"`
	CooldownUntil *time.Time `json:"cooldown_until"`
	ShadowEnabled *bool      `json:"shadow_enabled"`
}
