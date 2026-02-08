package dto

type SwipeRequest struct {
	TargetID int64  `json:"target_id"`
	Action   string `json:"action"`
}
