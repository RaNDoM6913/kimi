package dto

import "time"

type LikesIncomingPreviewResponse struct {
	UserID  int64     `json:"user_id"`
	LikedAt time.Time `json:"liked_at"`
}

type LikesIncomingProfileResponse struct {
	UserID      int64     `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Age         int       `json:"age"`
	CityID      string    `json:"city_id"`
	City        string    `json:"city"`
	LikedAt     time.Time `json:"liked_at"`
}

type LikesIncomingResponse struct {
	Blurred    bool                           `json:"blurred"`
	TotalCount int                            `json:"total_count"`
	Preview    []LikesIncomingPreviewResponse `json:"preview"`
	Profiles   []LikesIncomingProfileResponse `json:"profiles,omitempty"`
}

type LikesRevealOneResponse struct {
	Profile LikesIncomingProfileResponse `json:"profile"`
}
