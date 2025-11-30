package cart

import (
	"time"
	"warimas-be/internal/product"
)

type CartItem struct {
	ID        uint   `json:"id"`
	UserID    string `json:"user_id"`
	Quantity  int    `json:"quantity"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Product product.Product
}
