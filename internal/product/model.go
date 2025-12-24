package product

type Variant struct {
	ID           string  `json:"id"`
	ProductID    string  `json:"product_id"`
	Name         string  `json:"name"`
	QuantityType string  `json:"quantity_type"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	ImageUrl     *string `json:"imageurl,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type Product struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	SellerID        string     `json:"seller_id"`
	SellerName      string     `json:"seller_name"`
	CategoryID      string     `json:"category_id"`
	CategoryName    string     `json:"category_name"`
	SubcategoryID   string     `json:"subcategory_id"`
	SubcategoryName string     `json:"subcategory_name"`
	Slug            string     `json:"slug"`
	Variants        []*Variant `json:"variants"`
	Description     *string    `json:"description,omitempty"`
	Status          string     `json:"status"`
	ImageUrl        *string    `json:"imageurl,omitempty"`
}

type GetProductOptions struct {
	ProductID  string
	OnlyActive bool
}

type GetVariantOptions struct {
	VariantID  string
	OnlyActive bool
}
