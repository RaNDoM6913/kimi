package dto

type SwipeRequest struct {
	TargetID int64              `json:"target_id"`
	Action   string             `json:"action"`
	Client   *SwipeClientReport `json:"client,omitempty"`
}

type SwipeClientReport struct {
	CardViewMS    int      `json:"card_view_ms"`
	SwipeVelocity *float64 `json:"swipe_velocity,omitempty"`
	Screen        string   `json:"screen,omitempty"`
}
