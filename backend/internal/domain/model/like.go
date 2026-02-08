package model

import "time"

type Like struct {
	ID          int64     `json:"id"`
	FromUserID  int64     `json:"from_user_id"`
	ToUserID    int64     `json:"to_user_id"`
	IsSuperLike bool      `json:"is_super_like"`
	CreatedAt   time.Time `json:"created_at"`
}
