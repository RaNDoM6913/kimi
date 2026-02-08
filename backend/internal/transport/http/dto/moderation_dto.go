package dto

type ModerationStatusResponse struct {
	Status          string  `json:"status"`
	ReasonText      *string `json:"reason_text,omitempty"`
	RequiredFixStep *string `json:"required_fix_step,omitempty"`
	ETABucket       string  `json:"eta_bucket"`
}
