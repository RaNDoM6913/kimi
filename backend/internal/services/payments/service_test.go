package payments

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

type purchaseStoreStub struct {
	nextID      int64
	purchases   map[int64]pgrepo.PurchaseRecord
	providerTxs map[string]int64
}

func newPurchaseStoreStub() *purchaseStoreStub {
	return &purchaseStoreStub{
		nextID:      1,
		purchases:   make(map[int64]pgrepo.PurchaseRecord),
		providerTxs: make(map[string]int64),
	}
}

func (s *purchaseStoreStub) CreatePending(_ context.Context, userID int64, sku, provider string, payload map[string]any) (pgrepo.PurchaseRecord, error) {
	id := s.nextID
	s.nextID++
	now := time.Now().UTC()
	rec := pgrepo.PurchaseRecord{
		ID:        id,
		UserID:    userID,
		SKU:       sku,
		Provider:  provider,
		Status:    statusPending,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.purchases[id] = rec
	return rec, nil
}

func (s *purchaseStoreStub) FindByID(_ context.Context, purchaseID int64) (pgrepo.PurchaseRecord, error) {
	rec, ok := s.purchases[purchaseID]
	if !ok {
		return pgrepo.PurchaseRecord{}, pgrepo.ErrPurchaseNotFound
	}
	return rec, nil
}

func (s *purchaseStoreStub) FindByProviderTx(_ context.Context, provider, providerTxID string) (pgrepo.PurchaseRecord, error) {
	key := provider + "|" + providerTxID
	id, ok := s.providerTxs[key]
	if !ok {
		return pgrepo.PurchaseRecord{}, pgrepo.ErrPurchaseNotFound
	}
	rec, ok := s.purchases[id]
	if !ok {
		return pgrepo.PurchaseRecord{}, pgrepo.ErrPurchaseNotFound
	}
	return rec, nil
}

func (s *purchaseStoreStub) MarkConfirmed(_ context.Context, purchaseID int64, provider, providerTxID string, payload map[string]any) (pgrepo.PurchaseRecord, bool, error) {
	rec, ok := s.purchases[purchaseID]
	if !ok {
		return pgrepo.PurchaseRecord{}, false, pgrepo.ErrPurchaseNotFound
	}
	if rec.Provider != provider {
		return pgrepo.PurchaseRecord{}, false, fmt.Errorf("provider mismatch")
	}
	if rec.Status == statusConfirmed {
		return rec, false, nil
	}

	key := provider + "|" + providerTxID
	if existingID, exists := s.providerTxs[key]; exists && existingID != purchaseID {
		return pgrepo.PurchaseRecord{}, false, pgrepo.ErrProviderTxConflict
	}

	rec.Status = statusConfirmed
	rec.Payload = payload
	rec.ExternalTxID = &providerTxID
	rec.UpdatedAt = time.Now().UTC()
	s.purchases[purchaseID] = rec
	s.providerTxs[key] = purchaseID
	return rec, true, nil
}

type entitlementStoreStub struct {
	applyCount int
	lastSKU    string
	lastUserID int64
}

func (s *entitlementStoreStub) ApplyPurchaseSKU(_ context.Context, userID int64, sku string, _ time.Time) error {
	if userID <= 0 || sku == "" {
		return errors.New("invalid apply payload")
	}
	s.applyCount++
	s.lastSKU = sku
	s.lastUserID = userID
	return nil
}

type paymentTxStoreStub struct {
	nextID       int
	byID         map[string]pgrepo.PaymentTransactionRecord
	byIdem       map[string]string
	byProviderEv map[string]string
	grantCount   int
}

func newPaymentTxStoreStub() *paymentTxStoreStub {
	return &paymentTxStoreStub{
		nextID:       1,
		byID:         make(map[string]pgrepo.PaymentTransactionRecord),
		byIdem:       make(map[string]string),
		byProviderEv: make(map[string]string),
	}
}

func (s *paymentTxStoreStub) BeginPurchase(
	_ context.Context,
	userID int64,
	provider, productSKU string,
	amount int,
	currency, idempotencyKey string,
) (pgrepo.PaymentTransactionRecord, bool, error) {
	if txID, ok := s.byIdem[idempotencyKey]; ok {
		return s.byID[txID], false, nil
	}
	id := fmt.Sprintf("tx-%d", s.nextID)
	s.nextID++
	now := time.Now().UTC()
	rec := pgrepo.PaymentTransactionRecord{
		ID:             id,
		UserID:         userID,
		Provider:       provider,
		IdempotencyKey: idempotencyKey,
		Amount:         amount,
		Currency:       currency,
		ProductSKU:     productSKU,
		Status:         "PENDING",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.byID[id] = rec
	s.byIdem[idempotencyKey] = id
	return rec, true, nil
}

func (s *paymentTxStoreStub) ConfirmPayment(
	_ context.Context,
	provider, providerEventID string,
	_ map[string]any,
	_ time.Time,
) (pgrepo.PaymentTransactionRecord, bool, error) {
	key := provider + "|" + providerEventID
	if txID, ok := s.byProviderEv[key]; ok {
		rec := s.byID[txID]
		if rec.Status == "SUCCEEDED" {
			return rec, true, nil
		}
	}

	txID, ok := s.byIdem[providerEventID]
	if !ok {
		return pgrepo.PaymentTransactionRecord{}, false, pgrepo.ErrPaymentTransactionNotFound
	}
	rec := s.byID[txID]
	if rec.Status == "SUCCEEDED" {
		return rec, true, nil
	}
	rec.Status = "SUCCEEDED"
	rec.ProviderEventID = &providerEventID
	rec.UpdatedAt = time.Now().UTC()
	s.byID[txID] = rec
	s.byProviderEv[key] = txID
	s.grantCount++
	return rec, false, nil
}

func TestConfirmWebhookIdempotentByProviderTxID(t *testing.T) {
	purchases := newPurchaseStoreStub()
	entitlements := &entitlementStoreStub{}

	svc := NewService(Dependencies{
		Purchases:    purchases,
		Entitlements: entitlements,
	})

	createResult, err := svc.Create(context.Background(), 42, CreateInput{
		SKU:      "reveal_1",
		Provider: "telegram_stars",
	})
	if err != nil {
		t.Fatalf("create purchase: %v", err)
	}

	first, err := svc.ConfirmWebhook(context.Background(), WebhookInput{
		PurchaseID:   createResult.PurchaseID,
		Provider:     "telegram_stars",
		ProviderTxID: "tx-1001",
		Status:       "confirmed",
		Payload: map[string]any{
			"amount_minor": 99,
		},
	})
	if err != nil {
		t.Fatalf("first confirm webhook: %v", err)
	}
	if first.AlreadyProcessed {
		t.Fatalf("first webhook must not be idempotent")
	}
	if entitlements.applyCount != 1 {
		t.Fatalf("expected 1 entitlement apply, got %d", entitlements.applyCount)
	}

	second, err := svc.ConfirmWebhook(context.Background(), WebhookInput{
		PurchaseID:   createResult.PurchaseID,
		Provider:     "telegram_stars",
		ProviderTxID: "tx-1001",
		Status:       "confirmed",
		Payload: map[string]any{
			"amount_minor": 99,
		},
	})
	if err != nil {
		t.Fatalf("second confirm webhook: %v", err)
	}
	if !second.AlreadyProcessed {
		t.Fatalf("second webhook must be idempotent")
	}
	if entitlements.applyCount != 1 {
		t.Fatalf("entitlements applied more than once: %d", entitlements.applyCount)
	}
	if entitlements.lastSKU != "reveal_1" {
		t.Fatalf("unexpected sku applied: %s", entitlements.lastSKU)
	}
}

func TestBeginPurchaseIdempotentByKey(t *testing.T) {
	txStore := newPaymentTxStoreStub()
	svc := NewService(Dependencies{
		PaymentTransactions: txStore,
	})

	first, err := svc.BeginPurchase(context.Background(), 101, "tg_stars", "superlike_3", 199, "BYN", "idem-1")
	if err != nil {
		t.Fatalf("first begin purchase: %v", err)
	}
	if first.TransactionID == "" || first.Idempotent {
		t.Fatalf("first begin should create tx and be non-idempotent: %+v", first)
	}

	second, err := svc.BeginPurchase(context.Background(), 101, "tg_stars", "superlike_3", 199, "BYN", "idem-1")
	if err != nil {
		t.Fatalf("second begin purchase: %v", err)
	}
	if second.TransactionID != first.TransactionID {
		t.Fatalf("expected same transaction id for idempotent begin, got %s vs %s", second.TransactionID, first.TransactionID)
	}
	if !second.Idempotent {
		t.Fatalf("second begin must be idempotent")
	}
}

func TestConfirmPaymentIdempotentGrantOnce(t *testing.T) {
	txStore := newPaymentTxStoreStub()
	svc := NewService(Dependencies{
		PaymentTransactions: txStore,
	})

	begin, err := svc.BeginPurchase(context.Background(), 101, "external", "plus_month", 1299, "BYN", "evt-42")
	if err != nil {
		t.Fatalf("begin purchase: %v", err)
	}
	if begin.TransactionID == "" {
		t.Fatalf("expected transaction id")
	}

	first, err := svc.ConfirmPayment(context.Background(), "external", "evt-42", map[string]any{"raw": true})
	if err != nil {
		t.Fatalf("first confirm payment: %v", err)
	}
	if first.Idempotent {
		t.Fatalf("first confirm must not be idempotent")
	}
	if txStore.grantCount != 1 {
		t.Fatalf("expected single grant on first confirm, got %d", txStore.grantCount)
	}

	second, err := svc.ConfirmPayment(context.Background(), "external", "evt-42", map[string]any{"raw": true})
	if err != nil {
		t.Fatalf("second confirm payment: %v", err)
	}
	if !second.Idempotent {
		t.Fatalf("second confirm must be idempotent")
	}
	if txStore.grantCount != 1 {
		t.Fatalf("grant should not repeat, got %d", txStore.grantCount)
	}
}
