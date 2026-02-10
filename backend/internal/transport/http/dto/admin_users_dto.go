package dto

import "time"

type AdminUserPrivateResponse struct {
	UserID    int64      `json:"user_id"`
	PhoneE164 *string    `json:"phone_e164"`
	Lat       *float64   `json:"lat"`
	Lon       *float64   `json:"lon"`
	LastGeoAt *time.Time `json:"last_geo_at"`
}

type AdminDailyMetricsItem struct {
	DayKey     string `json:"day_key"`
	CityID     string `json:"city_id"`
	Gender     string `json:"gender"`
	LookingFor string `json:"looking_for"`
	Likes      int    `json:"likes"`
	Dislikes   int    `json:"dislikes"`
	SuperLikes int    `json:"superlikes"`
	Matches    int    `json:"matches"`
	Reports    int    `json:"reports"`
	Approved   int    `json:"approved"`
}

type AdminDailyMetricsResponse struct {
	Items []AdminDailyMetricsItem `json:"items"`
}

type AdminAntiAbuseSummaryResponse struct {
	TooFast1h         int64 `json:"too_fast_1h"`
	CooldownApplied1h int64 `json:"cooldown_applied_1h"`
	ShadowEnabled24h  int64 `json:"shadow_enabled_24h"`
}

type AdminAntiAbuseTopItem struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

type AdminAntiAbuseTopResponse struct {
	Kind  string                  `json:"kind"`
	Limit int64                   `json:"limit"`
	Items []AdminAntiAbuseTopItem `json:"items"`
}
