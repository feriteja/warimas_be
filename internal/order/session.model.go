package order

import (
	"time"

	"github.com/google/uuid"
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
