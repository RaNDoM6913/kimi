package model

import "time"

type Location struct {
	UserID      int64     `json:"user_id"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	City        string    `json:"city"`
	NearestCity string    `json:"nearest_city"`
	UpdatedAt   time.Time `json:"updated_at"`
}
