package model

import "time"

type UserPrivate struct {
	UserID    int64     `json:"user_id"`
	PhoneE164 string    `json:"phone_e164"`
	PhoneHash string    `json:"phone_hash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
