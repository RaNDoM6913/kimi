package dto

type AdEventRequest struct {
	AdID int64                  `json:"ad_id"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}
