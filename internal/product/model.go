package product

type Variant struct {
	ID            string  `json:"id"`
	ProductID     string  `json:"product_id"`
	Name          string  `json:"name"`
	QuantityType  string  `json:"quantity_type"`
	Price         float64 `json:"price"`
	Stock         int     `json:"stock"`
	ImageUrl      *string `json:"imageurl,omitempty"`
	SubcategoryId *string `json:"subcategory_id,omitempty"`
}

type Product struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	SellerID    string     `json:"seller_id"`
	CategoryID  string     `json:"category_id"`
	Slug        string     `json:"slug"`
	Price       float64    `json:"price"`
	Variants    []*Variant `json:"variants"`
	Stock       int        `json:"stock"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	ImageUrl    *string    `json:"imageurl,omitempty"`
}

// type CategoryProduct struct {
// 	CategoryName string    `json:"categoryName"`
// 	Products     []Product `json:"product"`
// }
