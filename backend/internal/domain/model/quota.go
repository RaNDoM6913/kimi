package model

import "time"

type Quota struct {
	UserID            int64     `json:"user_id"`
	LocalDate         string    `json:"local_date"`
	FreeLikesLimit    int       `json:"free_likes_limit"`
	FreeLikesUsed     int       `json:"free_likes_used"`
	LikeTokens        int       `json:"like_tokens"`
	TooFastLastAction time.Time `json:"too_fast_last_action"`
}
