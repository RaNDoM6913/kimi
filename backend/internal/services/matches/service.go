package matches

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

var (
	ErrValidation          = errors.New("validation error")
	ErrInvalidReportReason = errors.New("invalid report reason")
)

type MatchStore interface {
	ListActiveForUser(ctx context.Context, userID int64, limit int) ([]pgrepo.ActiveMatchRecord, error)
	DeleteByUsers(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error)
}

type BlockStore interface {
	Upsert(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, reason string) error
}

type ReportStore interface {
	Create(ctx context.Context, tx pgx.Tx, reporterUserID, targetUserID int64, reason, details string) error
}

type Service struct {
	pool        *pgxpool.Pool
	matchStore  MatchStore
	blockStore  BlockStore
	reportStore ReportStore
}

type Dependencies struct {
	Pool        *pgxpool.Pool
	MatchStore  MatchStore
	BlockStore  BlockStore
	ReportStore ReportStore
}

type MatchItem struct {
	ID           int64
	TargetUserID int64
	DisplayName  string
	Age          int
	CityID       string
	City         string
	CreatedAt    time.Time
}

func NewService(deps Dependencies) *Service {
	return &Service{
		pool:        deps.Pool,
		matchStore:  deps.MatchStore,
		blockStore:  deps.BlockStore,
		reportStore: deps.ReportStore,
	}
}

func (s *Service) List(ctx context.Context, userID int64, limit int) ([]MatchItem, error) {
	if userID <= 0 {
		return nil, ErrValidation
	}
	if s.matchStore == nil {
		return nil, fmt.Errorf("match store is nil")
	}

	rows, err := s.matchStore.ListActiveForUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	items := make([]MatchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, MatchItem{
			ID:           row.ID,
			TargetUserID: row.TargetUserID,
			DisplayName:  row.DisplayName,
			Age:          row.Age,
			CityID:       row.CityID,
			City:         row.City,
			CreatedAt:    row.CreatedAt,
		})
	}
	return items, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, targetID int64) (bool, error) {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return false, ErrValidation
	}
	if s.pool == nil || s.matchStore == nil {
		return false, fmt.Errorf("unmatch dependencies are not configured")
	}

	var deleted bool
	if err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		ok, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, targetID)
		if err != nil {
			return err
		}
		deleted = ok
		return nil
	}); err != nil {
		return false, err
	}

	return deleted, nil
}

func (s *Service) Block(ctx context.Context, userID, targetID int64, reason string) error {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return ErrValidation
	}
	if s.pool == nil || s.blockStore == nil || s.matchStore == nil {
		return fmt.Errorf("block dependencies are not configured")
	}

	return pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		if err := s.blockStore.Upsert(txCtx, tx, userID, targetID, reason); err != nil {
			return err
		}
		_, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, targetID)
		return err
	})
}

func (s *Service) Report(ctx context.Context, userID, targetID int64, reason, details string) error {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return ErrValidation
	}
	if !isAllowedReason(reason) {
		return ErrInvalidReportReason
	}
	if s.pool == nil || s.reportStore == nil {
		return fmt.Errorf("report dependencies are not configured")
	}

	return pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		return s.reportStore.Create(txCtx, tx, userID, targetID, strings.ToLower(strings.TrimSpace(reason)), details)
	})
}

func isAllowedReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case string(enums.ReportReasonSpam),
		string(enums.ReportReasonFake),
		string(enums.ReportReasonAbusive),
		string(enums.ReportReasonOther):
		return true
	default:
		return false
	}
}
