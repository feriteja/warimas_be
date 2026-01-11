package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type Repository interface {
	SavePayment(p *Payment) error
	UpdatePaymentStatus(externalID, status string) error
	GetPaymentByOrder(orderID uint) (*Payment, error)
	SavePaymentWebhook(
		ctx context.Context,
		provider string,
		eventID string,
		eventType string,
		externalID string,
		payload json.RawMessage,
		signatureValid bool,
	) (webhookID int64, isDuplicate bool, err error)

	MarkWebhookProcessed(ctx context.Context, webhookID int64) error
	MarkWebhookFailed(ctx context.Context, webhookID int64, reason string) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) SavePayment(p *Payment) error {
	_, err := r.db.Exec(`
		INSERT INTO payments (order_id, 
		external_reference, 
		invoice_url, 
		amount, 
		status, 
		payment_method, 
		channel_code, 
		payment_code,
		provider,
		currency,
		expire_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		p.OrderID, p.ExternalReference, p.InvoiceURL, p.Amount, p.Status, p.PaymentMethod, p.ChannelCode, p.PaymentCode,
		"XENDIT", "IDR", p.ExpireAt,
	)
	return err
}

func (r *repository) UpdatePaymentStatus(externalID, status string) error {
	_, err := r.db.Exec(`
		UPDATE payments SET status = $1 WHERE external_id = $2
	`, status, externalID)
	return err
}

func (r *repository) GetPaymentByOrder(orderID uint) (*Payment, error) {
	row := r.db.QueryRow(`
		SELECT id, order_id, external_reference, invoice_url, amount, status, payment_method, created_at, updated_at
		FROM payments WHERE order_id = $1
	`, orderID)

	var p Payment
	err := row.Scan(
		&p.ID, &p.OrderID, &p.ExternalReference, &p.InvoiceURL,
		&p.Amount, &p.Status, &p.PaymentMethod, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) SavePaymentWebhook(
	ctx context.Context,
	provider string,
	eventID string,
	eventType string,
	externalID string,
	payload json.RawMessage,
	signatureValid bool,
) (int64, bool, error) {

	const q = `
	INSERT INTO payment_webhooks (
		provider,
		event_type,
		event_id,
		external_id,
		signature_valid,
		payload
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (provider, event_id)
	DO NOTHING
	RETURNING id;
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		q,
		provider,
		eventID,
		eventType,
		externalID,
		signatureValid,
		payload,
	).Scan(&id)

	if err != nil {
		// Duplicate webhook → idempotent success
		if errors.Is(err, sql.ErrNoRows) {
			return 0, true, nil
		}
		return 0, false, err
	}

	return id, false, nil
}

func (r *repository) MarkWebhookProcessed(
	ctx context.Context,
	webhookID int64,
) error {

	const q = `
	UPDATE payment_webhooks
	SET processed_at = now()
	WHERE id = $1;
	`

	_, err := r.db.ExecContext(ctx, q, webhookID)
	return err
}

func (r *repository) MarkWebhookFailed(
	ctx context.Context,
	webhookID int64,
	reason string,
) error {

	const q = `
	UPDATE payment_webhooks
	SET process_error = $2
	WHERE id = $1;
	`

	_, err := r.db.ExecContext(ctx, q, webhookID, reason)
	return err
}
