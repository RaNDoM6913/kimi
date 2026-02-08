package dto

type EventBatchItemRequest struct {
	Name  string                 `json:"name"`
	TS    int64                  `json:"ts"`
	Props map[string]interface{} `json:"props,omitempty"`
}

type EventsBatchRequest []EventBatchItemRequest

type EventsBatchResponse struct {
	OK       bool `json:"ok"`
	Accepted int  `json:"accepted"`
}

// Backward aliases for prior skeleton naming.
type EventRequest = EventBatchItemRequest
