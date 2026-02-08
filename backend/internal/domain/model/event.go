package model

import "time"

type Event struct {
	ID         int64                  `json:"id"`
	UserID     int64                  `json:"user_id"`
	Name       string                 `json:"name"`
	OccurredAt time.Time              `json:"occurred_at"`
	Payload    map[string]interface{} `json:"payload"`
}
