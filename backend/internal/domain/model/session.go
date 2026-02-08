package model

import "time"

type Session struct {
	ID            string    `json:"id"`
	UserID        int64     `json:"user_id"`
	RefreshToken  string    `json:"refresh_token"`
	UserAgent     string    `json:"user_agent"`
	IP            string    `json:"ip"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastRotatedAt time.Time `json:"last_rotated_at"`
	CreatedAt     time.Time `json:"created_at"`
}
