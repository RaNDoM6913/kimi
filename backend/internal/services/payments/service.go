package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
)

const (
	statusPending   = "pending"
	statusConfirmed = "confirmed"
)

var (
	ErrValidation                 = errors.New("validation error")
	ErrUnsupportedSKU             = errors.New("unsupported sku")
	ErrUnsupportedProductSKU      = errors.New("unsupported product sku")
	ErrUnsupportedProvider        = errors.New("unsupported provider")
	ErrPurchaseNotFound           = errors.New("purchase not found")
	ErrPaymentTransactionNotFound = errors.New("payment transaction not found")
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

type PaymentTransactionStore interface {
	BeginPurchase(
		ctx context.Context,
		userID int64,
		provider, productSKU string,
		amount int,
		currency, idempotencyKey string,
	) (pgrepo.PaymentTransactionRecord, bool, error)
	ConfirmPayment(
		ctx context.Context,
		provider, providerEventID string,
		payload map[string]any,
		now time.Time,
	) (pgrepo.PaymentTransactionRecord, bool, error)
}

type TelemetryService interface {
	IngestBatch(ctx context.Context, userID *int64, events []analyticsvc.BatchEvent) error
}

type Service struct {
	purchases    PurchaseStore
	entitlements EntitlementStore
	paymentTxs   PaymentTransactionStore
	telemetry    TelemetryService
	now          func() time.Time
}

type Dependencies struct {
	Purchases           PurchaseStore
	Entitlements        EntitlementStore
	PaymentTransactions PaymentTransactionStore
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

type BeginPurchaseResult struct {
	TransactionID string
	UserID        int64
	Provider      string
	ProductSKU    string
	Amount        int
	Currency      string
	Status        string
	Idempotent    bool
}

type ConfirmPaymentResult struct {
	TransactionID   string
	UserID          int64
	Provider        string
	ProviderEventID string
	ProductSKU      string
	Amount          int
	Currency        string
	Status          string
	Idempotent      bool
}

func NewService(deps Dependencies) *Service {
	return &Service{
		purchases:    deps.Purchases,
		entitlements: deps.Entitlements,
		paymentTxs:   deps.PaymentTransactions,
		now:          time.Now,
	}
}

func (s *Service) AttachTelemetry(telemetry TelemetryService) {
	s.telemetry = telemetry
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

func (s *Service) BeginPurchase(
	ctx context.Context,
	userID int64,
	provider, productSKU string,
	amount int,
	currency, idempotencyKey string,
) (BeginPurchaseResult, error) {
	if userID <= 0 {
		return BeginPurchaseResult{}, ErrValidation
	}
	if s.paymentTxs == nil {
		return BeginPurchaseResult{}, fmt.Errorf("payment transaction store is nil")
	}

	normalizedProvider, err := normalizeDevProvider(provider)
	if err != nil {
		return BeginPurchaseResult{}, err
	}
	normalizedSKU, err := normalizeDevProductSKU(productSKU)
	if err != nil {
		return BeginPurchaseResult{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return BeginPurchaseResult{}, ErrValidation
	}
	if amount <= 0 {
		amount = defaultAmountForSKU(normalizedSKU)
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "BYN"
	}

	record, created, err := s.paymentTxs.BeginPurchase(ctx, userID, normalizedProvider, normalizedSKU, amount, currency, idempotencyKey)
	if err != nil {
		return BeginPurchaseResult{}, err
	}
	if created {
		s.logPaymentAudit(ctx, record.UserID, "PAYMENT_BEGIN", record.ProviderEventID, record.ProductSKU, record.Amount)
	}

	return BeginPurchaseResult{
		TransactionID: record.ID,
		UserID:        record.UserID,
		Provider:      record.Provider,
		ProductSKU:    record.ProductSKU,
		Amount:        record.Amount,
		Currency:      record.Currency,
		Status:        record.Status,
		Idempotent:    !created,
	}, nil
}

func (s *Service) ConfirmPayment(
	ctx context.Context,
	provider, providerEventID string,
	payload map[string]any,
) (ConfirmPaymentResult, error) {
	if s.paymentTxs == nil {
		return ConfirmPaymentResult{}, fmt.Errorf("payment transaction store is nil")
	}

	normalizedProvider, err := normalizeDevProvider(provider)
	if err != nil {
		return ConfirmPaymentResult{}, err
	}
	providerEventID = strings.TrimSpace(providerEventID)
	if providerEventID == "" {
		return ConfirmPaymentResult{}, ErrValidation
	}

	record, idempotent, err := s.paymentTxs.ConfirmPayment(ctx, normalizedProvider, providerEventID, payload, s.now().UTC())
	if err != nil {
		if errors.Is(err, pgrepo.ErrPaymentTransactionNotFound) {
			return ConfirmPaymentResult{}, ErrPaymentTransactionNotFound
		}
		return ConfirmPaymentResult{}, err
	}

	if !idempotent {
		s.logPaymentAudit(ctx, record.UserID, "PAYMENT_SUCCEEDED", record.ProviderEventID, record.ProductSKU, record.Amount)
	}

	return ConfirmPaymentResult{
		TransactionID:   record.ID,
		UserID:          record.UserID,
		Provider:        record.Provider,
		ProviderEventID: derefString(record.ProviderEventID),
		ProductSKU:      record.ProductSKU,
		Amount:          record.Amount,
		Currency:        record.Currency,
		Status:          record.Status,
		Idempotent:      idempotent,
	}, nil
}

func (s *Service) logPaymentAudit(
	ctx context.Context,
	userID int64,
	action string,
	providerEventID *string,
	productSKU string,
	amount int,
) {
	if s.telemetry == nil || userID <= 0 {
		return
	}

	uid := userID
	_ = s.telemetry.IngestBatch(ctx, &uid, []analyticsvc.BatchEvent{
		{
			Name: "audit_log",
			TS:   s.now().UTC().UnixMilli(),
			Props: map[string]any{
				"action":            action,
				"provider_event_id": derefString(providerEventID),
				"product_sku":       strings.ToLower(strings.TrimSpace(productSKU)),
				"amount":            amount,
			},
		},
	})
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

func normalizeDevProductSKU(raw string) (string, error) {
	sku := strings.ToLower(strings.TrimSpace(raw))
	switch sku {
	case "boost_60m",
		"superlike_3",
		"incognito_24h",
		"msg_nomatch_1",
		"plus_month":
		return sku, nil
	default:
		return "", ErrUnsupportedProductSKU
	}
}

func normalizeProvider(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeDevProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	switch provider {
	case "tg_stars", "external":
		return provider, nil
	default:
		return "", ErrUnsupportedProvider
	}
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

func defaultAmountForSKU(sku string) int {
	switch strings.ToLower(strings.TrimSpace(sku)) {
	case "boost_60m":
		return 100
	case "superlike_3":
		return 99
	case "incognito_24h":
		return 79
	case "msg_nomatch_1":
		return 59
	case "plus_month":
		return 1299
	default:
		return 1
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
