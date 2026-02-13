package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPaymentTransactionNotFound = errors.New("payment transaction not found")

type PaymentTransactionRepo struct {
	pool *pgxpool.Pool
}

type PaymentTransactionRecord struct {
	ID              string
	UserID          int64
	Provider        string
	ProviderEventID *string
	IdempotencyKey  string
	Amount          int
	Currency        string
	ProductSKU      string
	Status          string
	ResultPayload   map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewPaymentTransactionRepo(pool *pgxpool.Pool) *PaymentTransactionRepo {
	return &PaymentTransactionRepo{pool: pool}
}

func (r *PaymentTransactionRepo) BeginPurchase(
	ctx context.Context,
	userID int64,
	provider, productSKU string,
	amount int,
	currency, idempotencyKey string,
) (PaymentTransactionRecord, bool, error) {
	if r.pool == nil {
		return PaymentTransactionRecord{}, false, fmt.Errorf("postgres pool is nil")
	}
	if userID <= 0 || amount <= 0 {
		return PaymentTransactionRecord{}, false, fmt.Errorf("invalid begin purchase payload")
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	productSKU = strings.ToLower(strings.TrimSpace(productSKU))
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "BYN"
	}
	if provider == "" || productSKU == "" || idempotencyKey == "" {
		return PaymentTransactionRecord{}, false, fmt.Errorf("invalid begin purchase payload")
	}

	txID := uuid.NewString()
	record, err := scanPaymentTransactionRow(r.pool.QueryRow(ctx, `
INSERT INTO payment_transactions (
	id,
	user_id,
	provider,
	idempotency_key,
	amount,
	currency,
	product_sku,
	status,
	created_at,
	updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', NOW(), NOW())
ON CONFLICT (idempotency_key) DO UPDATE
SET updated_at = payment_transactions.updated_at
RETURNING
	id,
	user_id,
	provider,
	provider_event_id,
	idempotency_key,
	amount,
	currency,
	product_sku,
	status,
	result_payload,
	created_at,
	updated_at
`, txID, userID, provider, idempotencyKey, amount, currency, productSKU))
	if err != nil {
		return PaymentTransactionRecord{}, false, fmt.Errorf("begin purchase transaction: %w", err)
	}

	created := strings.EqualFold(record.ID, txID)
	return record, created, nil
}

func (r *PaymentTransactionRepo) ConfirmPayment(
	ctx context.Context,
	provider, providerEventID string,
	payload map[string]any,
	now time.Time,
) (PaymentTransactionRecord, bool, error) {
	if r.pool == nil {
		return PaymentTransactionRecord{}, false, fmt.Errorf("postgres pool is nil")
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	providerEventID = strings.TrimSpace(providerEventID)
	if provider == "" || providerEventID == "" {
		return PaymentTransactionRecord{}, false, fmt.Errorf("invalid confirm payload")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var out PaymentTransactionRecord
	idempotent := false
	err := WithTx(ctx, r.pool, func(txCtx context.Context, tx pgx.Tx) error {
		rec, err := r.lockForConfirm(txCtx, tx, provider, providerEventID)
		if err != nil {
			return err
		}

		if strings.EqualFold(rec.Status, "SUCCEEDED") {
			idempotent = true
			out = rec
			return nil
		}

		if err := r.grantEntitlementTx(txCtx, tx, rec.UserID, rec.ProductSKU, now.UTC()); err != nil {
			return err
		}

		updated, err := r.markSucceededTx(txCtx, tx, rec.ID, providerEventID, payload)
		if err != nil {
			return err
		}
		out = updated
		idempotent = false
		return nil
	})
	if err != nil {
		return PaymentTransactionRecord{}, false, err
	}

	return out, idempotent, nil
}

func (r *PaymentTransactionRepo) lockForConfirm(ctx context.Context, tx pgx.Tx, provider, providerEventID string) (PaymentTransactionRecord, error) {
	if tx == nil {
		return PaymentTransactionRecord{}, fmt.Errorf("transaction is required")
	}

	rec, err := scanPaymentTransactionRow(tx.QueryRow(ctx, `
SELECT
	id,
	user_id,
	provider,
	provider_event_id,
	idempotency_key,
	amount,
	currency,
	product_sku,
	status,
	result_payload,
	created_at,
	updated_at
FROM payment_transactions
WHERE provider = $1
  AND provider_event_id = $2
FOR UPDATE
`, provider, providerEventID))
	if err == nil {
		return rec, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PaymentTransactionRecord{}, fmt.Errorf("lock payment transaction by provider_event_id: %w", err)
	}

	rec, err = scanPaymentTransactionRow(tx.QueryRow(ctx, `
SELECT
	id,
	user_id,
	provider,
	provider_event_id,
	idempotency_key,
	amount,
	currency,
	product_sku,
	status,
	result_payload,
	created_at,
	updated_at
FROM payment_transactions
WHERE provider = $1
  AND idempotency_key = $2
FOR UPDATE
`, provider, providerEventID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentTransactionRecord{}, ErrPaymentTransactionNotFound
		}
		return PaymentTransactionRecord{}, fmt.Errorf("lock payment transaction by idempotency_key: %w", err)
	}

	if rec.ProviderEventID != nil && strings.TrimSpace(*rec.ProviderEventID) != "" {
		return rec, nil
	}

	if _, err := tx.Exec(ctx, `
UPDATE payment_transactions
SET
	provider_event_id = $2,
	updated_at = NOW()
WHERE id = $1
`, rec.ID, providerEventID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return PaymentTransactionRecord{}, ErrProviderTxConflict
		}
		return PaymentTransactionRecord{}, fmt.Errorf("bind provider event id: %w", err)
	}

	rec.ProviderEventID = &providerEventID
	return rec, nil
}

func (r *PaymentTransactionRepo) markSucceededTx(
	ctx context.Context,
	tx pgx.Tx,
	transactionID string,
	providerEventID string,
	payload map[string]any,
) (PaymentTransactionRecord, error) {
	if tx == nil {
		return PaymentTransactionRecord{}, fmt.Errorf("transaction is required")
	}

	payloadJSON, err := marshalAnyPayload(payload)
	if err != nil {
		return PaymentTransactionRecord{}, err
	}

	row := tx.QueryRow(ctx, `
UPDATE payment_transactions
SET
	provider_event_id = $2,
	status = 'SUCCEEDED',
	result_payload = $3::jsonb,
	updated_at = NOW()
WHERE id = $1
RETURNING
	id,
	user_id,
	provider,
	provider_event_id,
	idempotency_key,
	amount,
	currency,
	product_sku,
	status,
	result_payload,
	created_at,
	updated_at
`, transactionID, providerEventID, payloadJSON)

	rec, err := scanPaymentTransactionRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentTransactionRecord{}, ErrPaymentTransactionNotFound
		}
		return PaymentTransactionRecord{}, fmt.Errorf("mark payment transaction succeeded: %w", err)
	}
	return rec, nil
}

func (r *PaymentTransactionRepo) grantEntitlementTx(ctx context.Context, tx pgx.Tx, userID int64, sku string, now time.Time) error {
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO user_entitlements (user_id, updated_at)
VALUES ($1, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure user_entitlements row: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO user_credits (user_id, updated_at)
VALUES ($1, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure user_credits row: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO entitlements (
	user_id,
	superlike_credits,
	reveal_credits,
	like_tokens,
	message_wo_match_credits,
	updated_at
) VALUES ($1, 0, 0, 0, 0, NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
		return fmt.Errorf("ensure entitlements row: %w", err)
	}

	normalizedSKU := strings.ToLower(strings.TrimSpace(sku))
	switch normalizedSKU {
	case "boost_60m":
		if _, err := tx.Exec(ctx, `
UPDATE user_credits
SET boost_credits = boost_credits + 1, updated_at = NOW()
WHERE user_id = $1
`, userID); err != nil {
			return fmt.Errorf("grant boost credits: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE entitlements
SET
	boost_until = CASE
		WHEN boost_until IS NOT NULL AND boost_until > $2::timestamptz
			THEN boost_until + INTERVAL '60 minutes'
		ELSE $2::timestamptz + INTERVAL '60 minutes'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC()); err != nil {
			return fmt.Errorf("grant boost entitlement: %w", err)
		}
		return nil
	case "superlike_3":
		if _, err := tx.Exec(ctx, `
UPDATE user_credits
SET superlike_credits = superlike_credits + 3, updated_at = NOW()
WHERE user_id = $1
`, userID); err != nil {
			return fmt.Errorf("grant superlike credits: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE entitlements
SET superlike_credits = superlike_credits + 3, updated_at = NOW()
WHERE user_id = $1
`, userID); err != nil {
			return fmt.Errorf("grant superlike entitlement: %w", err)
		}
		return nil
	case "incognito_24h":
		if _, err := tx.Exec(ctx, `
UPDATE user_entitlements
SET
	incognito_until = CASE
		WHEN incognito_until IS NOT NULL AND incognito_until > $2::timestamptz
			THEN incognito_until + INTERVAL '24 hours'
		ELSE $2::timestamptz + INTERVAL '24 hours'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC()); err != nil {
			return fmt.Errorf("grant incognito in user_entitlements: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE entitlements
SET
	incognito_until = CASE
		WHEN incognito_until IS NOT NULL AND incognito_until > $2::timestamptz
			THEN incognito_until + INTERVAL '24 hours'
		ELSE $2::timestamptz + INTERVAL '24 hours'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC()); err != nil {
			return fmt.Errorf("grant incognito entitlement: %w", err)
		}
		return nil
	case "msg_nomatch_1":
		if _, err := tx.Exec(ctx, `
UPDATE user_credits
SET message_wo_match_credits = message_wo_match_credits + 1, updated_at = NOW()
WHERE user_id = $1
`, userID); err != nil {
			return fmt.Errorf("grant message credits in user_credits: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE entitlements
SET message_wo_match_credits = message_wo_match_credits + 1, updated_at = NOW()
WHERE user_id = $1
`, userID); err != nil {
			return fmt.Errorf("grant message entitlement: %w", err)
		}
		return nil
	case "plus_month":
		if _, err := tx.Exec(ctx, `
UPDATE user_entitlements
SET
	plus_active_until = CASE
		WHEN plus_active_until IS NOT NULL AND plus_active_until > $2::timestamptz
			THEN plus_active_until + INTERVAL '30 days'
		ELSE $2::timestamptz + INTERVAL '30 days'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC()); err != nil {
			return fmt.Errorf("grant plus in user_entitlements: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE entitlements
SET
	plus_expires_at = CASE
		WHEN plus_expires_at IS NOT NULL AND plus_expires_at > $2::timestamptz
			THEN plus_expires_at + INTERVAL '30 days'
		ELSE $2::timestamptz + INTERVAL '30 days'
	END,
	updated_at = NOW()
WHERE user_id = $1
`, userID, now.UTC()); err != nil {
			return fmt.Errorf("grant plus entitlement: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported product sku: %s", normalizedSKU)
	}
}

func scanPaymentTransactionRow(row pgx.Row) (PaymentTransactionRecord, error) {
	var rec PaymentTransactionRecord
	var payloadRaw []byte
	if err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Provider,
		&rec.ProviderEventID,
		&rec.IdempotencyKey,
		&rec.Amount,
		&rec.Currency,
		&rec.ProductSKU,
		&rec.Status,
		&payloadRaw,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		return PaymentTransactionRecord{}, err
	}
	rec.ResultPayload = decodeAnyPayload(payloadRaw)
	return rec, nil
}

func marshalAnyPayload(payload map[string]any) (string, error) {
	if payload == nil {
		return "null", nil
	}
	if len(payload) == 0 {
		return "{}", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	return string(raw), nil
}

func decodeAnyPayload(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	if string(raw) == "null" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return payload
}
