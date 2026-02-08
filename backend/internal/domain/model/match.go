package model

import "time"

type Match struct {
	ID          int64     `json:"id"`
	UserAID     int64     `json:"user_a_id"`
	UserBID     int64     `json:"user_b_id"`
	CreatedAt   time.Time `json:"created_at"`
	DMDeepLinkA string    `json:"dm_deep_link_a"`
	DMDeepLinkB string    `json:"dm_deep_link_b"`
}
