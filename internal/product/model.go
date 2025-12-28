package product

import "time"

type ProductSortField int

const (
	ProductSortFieldCreatedAt ProductSortField = iota
	ProductSortFieldName
	ProductSortFieldPrice
)

type SortDirection int

const (
	SortDirectionAsc SortDirection = iota
	SortDirectionDesc
)

type Variant struct {
	ID           string
	Name         string
	ProductID    string
	QuantityType string
	Price        float64
	Stock        int32
	ImageURL     string
	CategoryID   *string
	SellerID     string
	CreatedAt    string
	Description  *string
}

type Product struct {
	ID              string
	Name            string
	SellerID        string
	SellerName      string
	CategoryID      string
	CategoryName    string
	SubcategoryID   string
	SubcategoryName string
	Slug            string
	Variants        []*Variant
	Description     *string
	Status          string
	ImageURL        *string
	CreatedAt       time.Time
	UpdatedAt       *time.Time
}
type ProductByCategory struct {
	CategoryName  string
	TotalProducts int
	Products      []*Product
}

type GetProductOptions struct {
	ProductID  string
	OnlyActive bool
}

type GetVariantOptions struct {
	VariantID  string
	OnlyActive bool
}

type ProductListResult struct {
	Items      []*Product
	TotalCount *int
}

type ProductQueryOptions struct {
	// filters (plain values, no GraphQL)
	CategoryID *string
	SellerName *string
	Status     *string
	Search     *string
	MinPrice   *float64
	MaxPrice   *float64
	InStock    *bool

	// sorting
	SortField     ProductSortField
	SortDirection SortDirection

	Limit int32
	Page  int32

	// visibility
	IncludeDisabled bool
	IncludeCount    bool
	SellerID        *string
}
