package order

import (
	"time"
	"warimas-be/internal/graph/model"
)

type OrderStatus string

const (
	StatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	StatusPaid           OrderStatus = "PAID"
	StatusFulFilling     OrderStatus = "FULFILLING"
	StatusCompleted      OrderStatus = "COMPLETED"
	StatusCanceled       OrderStatus = "CANCELLED"
)

type Order struct {
	ID        uint
	UserID    *uint
	Total     uint
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Items     []OrderItem
	string
	ExternalID string
	Currency   string
}

type OrderItem struct {
	ID        uint
	OrderID   uint
	ProductID uint
	Quantity  int
	Price     int
	Product   model.Product
}
