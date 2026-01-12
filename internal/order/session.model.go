package order

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusPaid    PaymentStatus = "PAID"
	PaymentStatusFailed  PaymentStatus = "FAILED"
	PaymentStatusExpired PaymentStatus = "EXPIRED"
)

type CheckoutSessionStatus string

const (
	CheckoutSessionStatusPending  CheckoutSessionStatus = "PENDING"
	CheckoutSessionStatusPaid     CheckoutSessionStatus = "PAID"
	CheckoutSessionStatusExpired  CheckoutSessionStatus = "EXPIRED"
	CheckoutSessionStatusCanceled CheckoutSessionStatus = "CANCELLED"
)

type CheckoutSession struct {
	ID          uuid.UUID
	ExternalID  string
	Status      CheckoutSessionStatus
	ExpiresAt   time.Time
	CreatedAt   time.Time
	ConfirmedAt *time.Time

	// Optional / lifecycle-dependent
	UserID    *uint
	GuestID   *uuid.UUID
	AddressID *uuid.UUID

	Items []CheckoutSessionItem

	// Pricing (server-calculated only)
	Subtotal    int
	Tax         int
	ShippingFee int
	Discount    int
	TotalPrice  int
	Currency    string
}

type CheckoutSessionItem struct {
	ID        uuid.UUID
	SessionID uuid.UUID

	VariantID   string
	VariantName string
	ProductName string
	ImageURL    *string

	Quantity     int
	QuantityType string

	Price    int
	Subtotal int
}

type PaymentOrderInfoResponse struct {
	OrderExternalID string          `json:"orderExternalId"`
	Status          PaymentStatus   `json:"status"`
	ExpiresAt       time.Time       `json:"expiresAt"`
	TotalAmount     int             `json:"totalAmount"`
	Currency        string          `json:"currency"`
	ShippingAddress ShippingAddress `json:"shippingAddress"`
	Payment         PaymentDetail   `json:"payment"`
}

type ShippingAddress struct {
	Name         string  `json:"name"`
	ReceiverName string  `json:"receiver_name"`
	Phone        string  `json:"phone"`
	Address1     string  `json:"address2"`
	Address2     *string `json:"address1"`
	City         string  `json:"city"`
	Province     string  `json:"province"`
	PostalCode   string  `json:"postalCode"`
}

type PaymentDetail struct {
	Method       string   `json:"method"`
	Bank         *string  `json:"bank,omitempty"`        // Pointer because it might be null for some methods
	PaymentCode  *string  `json:"paymentCode,omitempty"` // Pointer because it might be null
	ReferenceID  string   `json:"referenceId"`
	Instructions []string `json:"instructions"`
}
