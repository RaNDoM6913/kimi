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
