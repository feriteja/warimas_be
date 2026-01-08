package payment

import (
	"database/sql"
)

type Repository interface {
	SavePayment(p *Payment) error
	UpdatePaymentStatus(externalID, status string) error
	GetPaymentByOrder(orderID uint) (*Payment, error)
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
		currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		p.OrderID, p.ExternalReference, p.InvoiceURL, p.Amount, p.Status, p.PaymentMethod, p.ChannelCode, p.PaymentCode,
		"XENDIT", "IDR",
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
		SELECT id, order_id, external_id, invoice_url, amount, status, payment_method, created_at, updated_at
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
