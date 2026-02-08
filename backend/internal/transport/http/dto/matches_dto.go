package dto

import "time"

type MatchItemResponse struct {
	ID           int64     `json:"id"`
	TargetUserID int64     `json:"target_user_id"`
	DisplayName  string    `json:"display_name"`
	Age          int       `json:"age"`
	CityID       string    `json:"city_id"`
	City         string    `json:"city"`
	CreatedAt    time.Time `json:"created_at"`
}

type MatchesResponse struct {
	Items []MatchItemResponse `json:"items"`
}

type UnmatchRequest struct {
	TargetID int64 `json:"target_id"`
}

type BlockRequest struct {
	TargetID int64  `json:"target_id"`
	Reason   string `json:"reason"`
}

type ReportRequest struct {
	TargetID int64  `json:"target_id"`
	Reason   string `json:"reason"`
	Details  string `json:"details"`
}
