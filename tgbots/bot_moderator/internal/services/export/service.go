package export

import (
	"context"
	"encoding/json"
	"time"
)

const (
	exportKindSheets    = "SHEETS"
	exportStatusPending = "PENDING"
)

type Repo interface {
	Enqueue(context.Context, string, json.RawMessage, string, time.Time) error
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnqueueSheetsRow(ctx context.Context, payload map[string]interface{}) error {
	if s.repo == nil {
		return nil
	}

	if payload == nil {
		payload = map[string]interface{}{}
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.repo.Enqueue(ctx, exportKindSheets, rawPayload, exportStatusPending, time.Now().UTC())
}
