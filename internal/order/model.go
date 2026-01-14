package order

import (
	"time"

	"github.com/google/uuid"
)

// --- Enums & Constants ---

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusAccepted       OrderStatus = "ACCEPTED"
	OrderStatusShipped        OrderStatus = "SHIPPED"
	OrderStatusCompleted      OrderStatus = "COMPLETED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
	OrderStatusFailed         OrderStatus = "FAILED"
)

type OrderSortField string

const (
	OrderSortFieldTotal     OrderSortField = "TOTAL"
	OrderSortFieldCreatedAt OrderSortField = "CREATED_AT"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "ASC"
	SortDirectionDesc SortDirection = "DESC"
)

const (
	defaultLimit = int32(20)
	maxLimit     = int32(100)
	defaultPage  = int32(1)
)

// --- Primary Model ---

type Order struct {
	ID            int32
	UserID        *int32
	AddressID     uuid.UUID
	TotalAmount   uint
	Status        OrderStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Items         []*OrderItem
	Subtotal      uint
	Tax           uint
	ShippingFee   uint
	Discount      uint
	ExternalID    string
	InvoiceNumber *string
	Currency      string
}

// --- Supporting Order Entities ---

type OrderItem struct {
	ID           uint
	OrderID      uint
	ProductID    string
	VariantID    string
	VariantName  string
	ProductName  string
	Quantity     int
	QuantityType string
	Price        float64
	Subtotal     float64
	ImageURL     *string
}

// --- Reference & Shared Types ---

type UserRef struct {
	ID int32 `json:"id"`
}

// --- Query & API Interaction Types ---

type OrderFilterInput struct {
	Search   *string      `json:"search,omitempty"`
	Status   *OrderStatus `json:"status,omitempty"`
	DateFrom *time.Time   `json:"dateFrom,omitempty"`
	DateTo   *time.Time   `json:"dateTo,omitempty"`
}

type OrderSortInput struct {
	Field     OrderSortField `json:"field"`
	Direction SortDirection  `json:"direction"`
}
