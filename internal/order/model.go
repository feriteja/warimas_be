package order

import (
	"time"
	"warimas-be/internal/graph/model"
)

type OrderStatus string

const (
	StatusPending  OrderStatus = "PENDING"
	StatusAccepted OrderStatus = "ACCEPTED"
	StatusRejected OrderStatus = "REJECTED"
	StatusCanceled OrderStatus = "CANCELED"
)

type Order struct {
	ID        uint
	UserID    uint
	Total     float64
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Items     []OrderItem
}

type OrderItem struct {
	ID        uint
	OrderID   uint
	ProductID uint
	Quantity  int
	Price     float64
	Product   model.Product
}
