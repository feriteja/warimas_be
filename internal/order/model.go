package order

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	StatusPaid           OrderStatus = "PAID"
	StatusFulFilling     OrderStatus = "FULFILLING"
	StatusCompleted      OrderStatus = "COMPLETED"
	StatusCanceled       OrderStatus = "CANCELLED"
)

const (
	defaultLimit = int32(20)
	maxLimit     = int32(100)
	defaultPage  = int32(1)
)

type Order struct {
	ID          uint
	UserID      *uint
	AddressID   uuid.UUID
	TotalAmount uint
	Status      OrderStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []*OrderItem
	Subtotal    uint
	Tax         uint
	ShippingFee uint
	Discount    uint
	ExternalID  string
	Currency    string
}

type OrderItem struct {
	ID          uint
	OrderID     uint
	ProductID   string
	VariantID   string
	VariantName string
	ProductName string
	Quantity    int
	Price       float64
	Subtotal    float64
}

type OrderFilterInput struct {
	Search   *string      `json:"search,omitempty"`
	Status   *OrderStatus `json:"status,omitempty"`
	DateFrom *time.Time   `json:"dateFrom,omitempty"`
	DateTo   *time.Time   `json:"dateTo,omitempty"`
}

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

type OrderSortInput struct {
	Field     OrderSortField `json:"field"`
	Direction SortDirection  `json:"direction"`
}
