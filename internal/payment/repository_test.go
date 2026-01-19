package payment

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_SavePayment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	p := &Payment{
		OrderID:           101,
		ExternalReference: "ref-payment-001",
		InvoiceURL:        "https://invoice.url/123",
		Amount:            150000,
		Status:            "PENDING",
		PaymentMethod:     "BCA_VA",
		ChannelCode:       "700700",
		PaymentCode:       "1234567890",
		ExpireAt:          time.Now().Add(24 * time.Hour),
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO payments`).
			WithArgs(
				p.OrderID, p.ExternalReference, p.InvoiceURL, p.Amount,
				p.Status, p.PaymentMethod, p.ChannelCode, p.PaymentCode, "XENDIT", "IDR", p.ExpireAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.SavePayment(context.Background(), p)
		assert.NoError(t, err)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO payments`).
			WillReturnError(errors.New("database error"))

		err := repo.SavePayment(context.Background(), p)
		assert.Error(t, err)
	})
}

func TestRepository_UpdatePaymentStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	extID := "pay-123"
	status := "PAID"

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE payments SET status = \$1 WHERE external_id = \$2`).
			WithArgs(status, extID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdatePaymentStatus(context.Background(), extID, status)
		assert.NoError(t, err)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectExec(`UPDATE payments SET status = \$1 WHERE external_id = \$2`).
			WithArgs(status, extID).
			WillReturnError(errors.New("db error"))

		err := repo.UpdatePaymentStatus(context.Background(), extID, status)
		assert.Error(t, err)
	})
}

func TestRepository_SavePaymentWebhook(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	provider := "XENDIT"
	eventID := "evt-1"
	eventType := "invoice.paid"
	extID := "ord-1"
	payload := []byte(`{}`)
	valid := true

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO payment_webhooks`).
			WithArgs(provider, eventType, eventID, extID, valid, payload).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

		id, isDup, err := repo.SavePaymentWebhook(ctx, provider, eventID, eventType, extID, payload, valid)
		assert.NoError(t, err)
		assert.False(t, isDup)
		assert.Equal(t, int64(10), id)
	})

	t.Run("Duplicate", func(t *testing.T) {
		// Simulate ON CONFLICT DO NOTHING returning no rows
		mock.ExpectQuery(`INSERT INTO payment_webhooks`).
			WithArgs(provider, eventType, eventID, extID, valid, payload).
			WillReturnError(sql.ErrNoRows)

		id, isDup, err := repo.SavePaymentWebhook(ctx, provider, eventID, eventType, extID, payload, valid)
		assert.NoError(t, err)
		assert.True(t, isDup)
		assert.Equal(t, int64(0), id)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO payment_webhooks`).
			WillReturnError(errors.New("db error"))

		_, _, err := repo.SavePaymentWebhook(ctx, provider, eventID, eventType, extID, payload, valid)
		assert.Error(t, err)
	})
}

func TestRepository_WebhookUpdates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	id := int64(1)

	t.Run("MarkProcessed", func(t *testing.T) {
		mock.ExpectExec(`UPDATE payment_webhooks SET processed_at = now\(\) WHERE id = \$1`).
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.MarkWebhookProcessed(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("MarkProcessed_Error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE payment_webhooks SET processed_at = now\(\) WHERE id = \$1`).
			WithArgs(id).
			WillReturnError(errors.New("db error"))

		err := repo.MarkWebhookProcessed(ctx, id)
		assert.Error(t, err)
	})

	t.Run("MarkFailed", func(t *testing.T) {
		reason := "error"
		mock.ExpectExec(`UPDATE payment_webhooks SET process_error = \$2 WHERE id = \$1`).
			WithArgs(id, reason).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.MarkWebhookFailed(ctx, id, reason)
		assert.NoError(t, err)
	})

	t.Run("MarkFailed_Error", func(t *testing.T) {
		reason := "error"
		mock.ExpectExec(`UPDATE payment_webhooks SET process_error = \$2 WHERE id = \$1`).
			WithArgs(id, reason).
			WillReturnError(errors.New("db error"))

		err := repo.MarkWebhookFailed(ctx, id, reason)
		assert.Error(t, err)
	})
}

func TestRepository_GetPaymentByOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	orderID := uint(101)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "order_id", "external_reference", "invoice_url", "amount",
			"status", "payment_method", "created_at", "updated_at", "payment_code", "expire_at",
		}).AddRow(
			1, orderID, "ref-payment-001", "https://invoice.url/123", 150000,
			"PENDING", "BCA_VA", time.Now(), time.Now(), "1234567890", time.Now().Add(time.Hour),
		)

		mock.ExpectQuery(`SELECT .* FROM payments WHERE order_id = \$1`).
			WithArgs(orderID).
			WillReturnRows(rows)

		payment, err := repo.GetPaymentByOrder(context.Background(), orderID)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, orderID, payment.OrderID)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM payments`).
			WillReturnError(errors.New("connection refused"))

		_, err := repo.GetPaymentByOrder(context.Background(), orderID)
		assert.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM payments`).
			WithArgs(orderID).
			WillReturnError(sql.ErrNoRows)

		p, err := repo.GetPaymentByOrder(context.Background(), orderID)
		assert.Error(t, err)
		assert.Nil(t, p)
		assert.Equal(t, sql.ErrNoRows, err)
	})
}
