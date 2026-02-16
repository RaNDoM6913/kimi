package dto

import "time"

type ModerationStatusResponse struct {
	Status          string  `json:"status"`
	ReasonText      *string `json:"reason_text,omitempty"`
	RequiredFixStep *string `json:"required_fix_step,omitempty"`
	ETABucket       string  `json:"eta_bucket"`
}

type AdminBotModQueueAcquireResponse struct {
	ModerationItem AdminBotModerationItem `json:"moderation_item"`
	Profile        AdminBotProfileCard    `json:"profile"`
	Media          AdminBotProfileMedia   `json:"media"`
}

type AdminBotModerationItem struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Status       string     `json:"status"`
	ETABucket    string     `json:"eta_bucket"`
	CreatedAt    time.Time  `json:"created_at"`
	LockedByTGID *int64     `json:"locked_by_tg_id,omitempty"`
	LockedAt     *time.Time `json:"locked_at,omitempty"`
	LockedUntil  *time.Time `json:"locked_until,omitempty"`
}

type AdminBotProfileCard struct {
	UserID      int64    `json:"user_id"`
	DisplayName string   `json:"display_name"`
	CityID      string   `json:"city_id"`
	Gender      string   `json:"gender"`
	LookingFor  string   `json:"looking_for"`
	Goals       []string `json:"goals"`
	Occupation  string   `json:"occupation"`
	Education   string   `json:"education"`
	Birthdate   *string  `json:"birthdate,omitempty"`
}

type AdminBotProfileMedia struct {
	Photos []string `json:"photos"`
	Circle *string  `json:"circle,omitempty"`
}

type AdminBotModerationRejectRequest struct {
	ReasonCode      string `json:"reason_code"`
	ReasonText      string `json:"reason_text"`
	RequiredFixStep string `json:"required_fix_step"`
}

type AdminBotModerationRejectReasonItem struct {
	ReasonCode      string `json:"reason_code"`
	Label           string `json:"label"`
	ReasonText      string `json:"reason_text"`
	RequiredFixStep string `json:"required_fix_step"`
}

type AdminBotModerationRejectReasonsResponse struct {
	Items []AdminBotModerationRejectReasonItem `json:"items"`
}
