package cart

import (
	"time"
)

type CartItem struct {
	ID        string       `json:"id"`
	UserID    int32        `json:"userId"`
	Quantity  int32        `json:"quantity"`
	Product   *ProductCart `json:"product,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt *time.Time   `json:"updatedAt,omitempty"`
}

type VariantCart struct {
	ID           string  `json:"id"`
	ProductID    string  `json:"product_id"`
	Name         string  `json:"name"`
	QuantityType string  `json:"quantity_type"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	Status       string  `json:"status"`
	ImageUrl     *string `json:"imageurl,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type ProductCart struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	SellerID      string      `json:"seller_id"`
	SellerName    string      `json:"seller_name"`
	CategoryID    string      `json:"category_id"`
	SubcategoryID string      `json:"subcategory_id"`
	Slug          string      `json:"slug"`
	Description   *string     `json:"description,omitempty"`
	Status        string      `json:"status"`
	ImageUrl      *string     `json:"imageurl,omitempty"`
	Variant       VariantCart `json:"variant"`
}

type AddToCartParams struct {
	UserID    uint
	VariantID string
	Quantity  uint32
}

type UpdateToCartParams struct {
	UserID    uint32
	VariantID string
	Quantity  uint32
}

type DeleteFromCartParams struct {
	UserID    uint32
	VariantID string
}

type CreateCartItemParams struct {
	UserID    uint
	VariantID string
	Quantity  uint32
}

type cartRow struct {
	CartID    string
	UserID    int32
	Quantity  int32
	CreatedAt time.Time
	UpdatedAt *time.Time

	ProductID       string
	ProductName     string
	SellerID        string
	SellerName      string
	CategoryID      string
	SubcategoryID   string
	Slug            string
	ProductImageURL *string

	VariantID        string
	VariantName      string
	VariantProductID string
	QuantityType     string
	Price            float64
	Stock            int
	VariantImageURL  *string
}
