package cart

import (
	"time"
	"warimas-be/internal/graph/model"
)

type CartItem struct {
	ID        uint `json:"id"`
	UserID    uint `json:"user_id"`
	ProductID uint `json:"productId"`
	Quantity  int  `json:"quantity"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Product model.Product
}
