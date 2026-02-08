package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

const (
	statusPending   = "pending"
	statusConfirmed = "confirmed"
)

var (
	ErrValidation       = errors.New("validation error")
	ErrUnsupportedSKU   = errors.New("unsupported sku")
	ErrPurchaseNotFound = errors.New("purchase not found")
)

type PurchaseStore interface {
	CreatePending(ctx context.Context, userID int64, sku, provider string, payload map[string]any) (pgrepo.PurchaseRecord, error)
	FindByID(ctx context.Context, purchaseID int64) (pgrepo.PurchaseRecord, error)
	FindByProviderTx(ctx context.Context, provider, providerTxID string) (pgrepo.PurchaseRecord, error)
	MarkConfirmed(ctx context.Context, purchaseID int64, provider, providerTxID string, payload map[string]any) (pgrepo.PurchaseRecord, bool, error)
}

type EntitlementStore interface {
	ApplyPurchaseSKU(ctx context.Context, userID int64, sku string, now time.Time) error
}

type Service struct {
	purchases    PurchaseStore
	entitlements EntitlementStore
	now          func() time.Time
}

type Dependencies struct {
	Purchases    PurchaseStore
	Entitlements EntitlementStore
}

type CreateInput struct {
	SKU      string
	Provider string
}

type CreateResult struct {
	PurchaseID int64
	SKU        string
	Provider   string
	Status     string
}

type WebhookInput struct {
	PurchaseID   int64
	Provider     string
	ProviderTxID string
	Status       string
	Payload      map[string]any
}

type WebhookResult struct {
	PurchaseID       int64
	UserID           int64
	SKU              string
	Status           string
	AlreadyProcessed bool
}

func NewService(deps Dependencies) *Service {
	return &Service{
		purchases:    deps.Purchases,
		entitlements: deps.Entitlements,
		now:          time.Now,
	}
}

func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (CreateResult, error) {
	if userID <= 0 {
		return CreateResult{}, ErrValidation
	}
	if s.purchases == nil {
		return CreateResult{}, fmt.Errorf("purchase store is nil")
	}

	sku, err := normalizeSKU(in.SKU)
	if err != nil {
		return CreateResult{}, err
	}
	provider := normalizeProvider(in.Provider)
	if provider == "" {
		return CreateResult{}, ErrValidation
	}

	record, err := s.purchases.CreatePending(ctx, userID, sku, provider, map[string]any{
		"source": "api",
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{
		PurchaseID: record.ID,
		SKU:        record.SKU,
		Provider:   record.Provider,
		Status:     record.Status,
	}, nil
}

func (s *Service) ConfirmWebhook(ctx context.Context, in WebhookInput) (WebhookResult, error) {
	if s.purchases == nil || s.entitlements == nil {
		return WebhookResult{}, fmt.Errorf("payments dependencies are not configured")
	}

	provider := normalizeProvider(in.Provider)
	providerTxID := strings.TrimSpace(in.ProviderTxID)
	if provider == "" || providerTxID == "" {
		return WebhookResult{}, ErrValidation
	}

	if !isConfirmationStatus(in.Status) {
		return WebhookResult{}, ErrValidation
	}

	existing, err := s.purchases.FindByProviderTx(ctx, provider, providerTxID)
	if err == nil {
		return WebhookResult{
			PurchaseID:       existing.ID,
			UserID:           existing.UserID,
			SKU:              existing.SKU,
			Status:           existing.Status,
			AlreadyProcessed: strings.EqualFold(existing.Status, statusConfirmed),
		}, nil
	}
	if err != nil && !errors.Is(err, pgrepo.ErrPurchaseNotFound) {
		return WebhookResult{}, err
	}

	if in.PurchaseID <= 0 {
		return WebhookResult{}, ErrValidation
	}

	purchase, err := s.purchases.FindByID(ctx, in.PurchaseID)
	if err != nil {
		if errors.Is(err, pgrepo.ErrPurchaseNotFound) {
			return WebhookResult{}, ErrPurchaseNotFound
		}
		return WebhookResult{}, err
	}

	updated, changed, err := s.purchases.MarkConfirmed(ctx, purchase.ID, provider, providerTxID, in.Payload)
	if err != nil {
		if errors.Is(err, pgrepo.ErrProviderTxConflict) {
			conflictRecord, lookupErr := s.purchases.FindByProviderTx(ctx, provider, providerTxID)
			if lookupErr == nil {
				return WebhookResult{
					PurchaseID:       conflictRecord.ID,
					UserID:           conflictRecord.UserID,
					SKU:              conflictRecord.SKU,
					Status:           conflictRecord.Status,
					AlreadyProcessed: strings.EqualFold(conflictRecord.Status, statusConfirmed),
				}, nil
			}
		}
		return WebhookResult{}, err
	}

	if !changed {
		if !strings.EqualFold(updated.Status, statusConfirmed) {
			return WebhookResult{}, fmt.Errorf("purchase did not transition to confirmed status")
		}
		return WebhookResult{
			PurchaseID:       updated.ID,
			UserID:           updated.UserID,
			SKU:              updated.SKU,
			Status:           updated.Status,
			AlreadyProcessed: true,
		}, nil
	}

	sku, err := normalizeSKU(updated.SKU)
	if err != nil {
		return WebhookResult{}, err
	}

	if err := s.entitlements.ApplyPurchaseSKU(ctx, updated.UserID, sku, s.now().UTC()); err != nil {
		return WebhookResult{}, err
	}

	return WebhookResult{
		PurchaseID:       updated.ID,
		UserID:           updated.UserID,
		SKU:              sku,
		Status:           updated.Status,
		AlreadyProcessed: false,
	}, nil
}

func normalizeSKU(raw string) (string, error) {
	sku := strings.ToLower(strings.TrimSpace(raw))
	switch sku {
	case "boost_30m",
		"superlike_pack_3",
		"reveal_1",
		"incognito_24h",
		"message_wo_match_1",
		"plus_1m":
		return sku, nil
	default:
		return "", ErrUnsupportedSKU
	}
}

func normalizeProvider(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isConfirmationStatus(raw string) bool {
	status := strings.ToLower(strings.TrimSpace(raw))
	if status == "" {
		return true
	}
	switch status {
	case "confirmed", "success", "paid":
		return true
	default:
		return false
	}
}
