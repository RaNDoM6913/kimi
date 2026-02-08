package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPurchaseNotFound   = errors.New("purchase not found")
	ErrProviderTxConflict = errors.New("provider tx already attached to another purchase")
)

type PurchaseRepo struct {
	pool *pgxpool.Pool
}

type PurchaseRecord struct {
	ID           int64
	UserID       int64
	SKU          string
	Provider     string
	ExternalTxID *string
	Status       string
	Payload      map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewPurchaseRepo(pool *pgxpool.Pool) *PurchaseRepo {
	return &PurchaseRepo{pool: pool}
}

func (r *PurchaseRepo) CreatePending(ctx context.Context, userID int64, sku, provider string, payload map[string]any) (PurchaseRecord, error) {
	if r.pool == nil {
		return PurchaseRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 || strings.TrimSpace(sku) == "" || strings.TrimSpace(provider) == "" {
		return PurchaseRecord{}, fmt.Errorf("invalid purchase create payload")
	}

	payloadJSON, err := marshalPayload(payload)
	if err != nil {
		return PurchaseRecord{}, err
	}

	var (
		record     PurchaseRecord
		rawPayload []byte
	)
	err = r.pool.QueryRow(ctx, `
INSERT INTO purchases (
	user_id,
	sku,
	provider,
	status,
	payload,
	created_at,
	updated_at
) VALUES ($1, $2, $3, 'pending', $4::jsonb, NOW(), NOW())
RETURNING id, user_id, sku, provider, external_txn_id, status, payload, created_at, updated_at
`, userID, strings.ToLower(strings.TrimSpace(sku)), strings.ToLower(strings.TrimSpace(provider)), payloadJSON).Scan(
		&record.ID,
		&record.UserID,
		&record.SKU,
		&record.Provider,
		&record.ExternalTxID,
		&record.Status,
		&rawPayload,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return PurchaseRecord{}, fmt.Errorf("create pending purchase: %w", err)
	}

	record.Payload = decodePayload(rawPayload)
	return record, nil
}

func (r *PurchaseRepo) FindByID(ctx context.Context, purchaseID int64) (PurchaseRecord, error) {
	if r.pool == nil {
		return PurchaseRecord{}, fmt.Errorf("postgres pool is nil")
	}
	if purchaseID <= 0 {
		return PurchaseRecord{}, fmt.Errorf("invalid purchase id")
	}

	record, err := scanPurchase(r.pool.QueryRow(ctx, `
SELECT id, user_id, sku, provider, external_txn_id, status, payload, created_at, updated_at
FROM purchases
WHERE id = $1
LIMIT 1
`, purchaseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PurchaseRecord{}, ErrPurchaseNotFound
		}
		return PurchaseRecord{}, fmt.Errorf("find purchase by id: %w", err)
	}

	return record, nil
}

func (r *PurchaseRepo) FindByProviderTx(ctx context.Context, provider, providerTxID string) (PurchaseRecord, error) {
	if r.pool == nil {
		return PurchaseRecord{}, fmt.Errorf("postgres pool is nil")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	providerTxID = strings.TrimSpace(providerTxID)
	if provider == "" || providerTxID == "" {
		return PurchaseRecord{}, fmt.Errorf("invalid provider tx payload")
	}

	record, err := scanPurchase(r.pool.QueryRow(ctx, `
SELECT id, user_id, sku, provider, external_txn_id, status, payload, created_at, updated_at
FROM purchases
WHERE provider = $1
  AND external_txn_id = $2
LIMIT 1
`, provider, providerTxID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PurchaseRecord{}, ErrPurchaseNotFound
		}
		return PurchaseRecord{}, fmt.Errorf("find purchase by provider tx: %w", err)
	}

	return record, nil
}

func (r *PurchaseRepo) MarkConfirmed(ctx context.Context, purchaseID int64, provider, providerTxID string, payload map[string]any) (PurchaseRecord, bool, error) {
	if r.pool == nil {
		return PurchaseRecord{}, false, fmt.Errorf("postgres pool is nil")
	}
	if purchaseID <= 0 {
		return PurchaseRecord{}, false, fmt.Errorf("invalid purchase id")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	providerTxID = strings.TrimSpace(providerTxID)
	if provider == "" || providerTxID == "" {
		return PurchaseRecord{}, false, fmt.Errorf("invalid provider tx payload")
	}

	payloadJSON, err := marshalPayload(payload)
	if err != nil {
		return PurchaseRecord{}, false, err
	}

	var rawPayload []byte
	row := r.pool.QueryRow(ctx, `
UPDATE purchases
SET
	external_txn_id = $2,
	status = 'confirmed',
	payload = $3::jsonb,
	updated_at = NOW()
WHERE id = $1
  AND provider = $4
  AND status <> 'confirmed'
RETURNING id, user_id, sku, provider, external_txn_id, status, payload, created_at, updated_at
`, purchaseID, providerTxID, payloadJSON, provider)

	var updated PurchaseRecord
	if err := row.Scan(
		&updated.ID,
		&updated.UserID,
		&updated.SKU,
		&updated.Provider,
		&updated.ExternalTxID,
		&updated.Status,
		&rawPayload,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	); err == nil {
		updated.Payload = decodePayload(rawPayload)
		return updated, true, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return PurchaseRecord{}, false, ErrProviderTxConflict
		}
		return PurchaseRecord{}, false, fmt.Errorf("mark purchase confirmed: %w", err)
	}

	existing, err := r.FindByID(ctx, purchaseID)
	if err != nil {
		return PurchaseRecord{}, false, err
	}
	if strings.ToLower(existing.Provider) != provider {
		return PurchaseRecord{}, false, fmt.Errorf("provider mismatch for purchase")
	}
	return existing, false, nil
}

func scanPurchase(row pgx.Row) (PurchaseRecord, error) {
	var (
		record     PurchaseRecord
		rawPayload []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.UserID,
		&record.SKU,
		&record.Provider,
		&record.ExternalTxID,
		&record.Status,
		&rawPayload,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return PurchaseRecord{}, err
	}
	record.Payload = decodePayload(rawPayload)
	return record, nil
}

func marshalPayload(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "{}", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal purchase payload: %w", err)
	}
	return string(raw), nil
}

func decodePayload(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{}
	}
	if payload == nil {
		return map[string]any{}
	}
	return payload
}
