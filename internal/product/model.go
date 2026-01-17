package product

import "time"

type ProductSortField int

const (
	ProductSortFieldCreatedAt ProductSortField = iota
	ProductSortFieldName
	ProductSortFieldPrice
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "ASC"
	SortDirectionDesc SortDirection = "DESC"
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
	CategoryID   *string
	CategorySlug *string
	SellerName   *string
	Status       *string
	Search       *string
	MinPrice     *float64
	MaxPrice     *float64
	InStock      *bool

	// sorting
	SortField     ProductSortField
	SortDirection SortDirection

	Limit int32
	Page  int32

	// visibility
	OnlyActive   bool
	IncludeCount bool
	SellerID     *string
}

type NewProductInput struct {
	Name          string
	ImageURL      *string
	Description   *string
	CategoryID    string
	SubcategoryID string
}

type UpdateProductInput struct {
	ID            string
	Name          *string
	ImageURL      *string
	Description   *string
	CategoryID    *string
	SubcategoryID *string
	Status        *string
}

type NewVariantInput struct {
	ProductID    string
	QuantityType string
	Name         string
	Price        float64
	Stock        int32
	ImageURL     *string
	Description  *string
}

type UpdateVariantInput struct {
	ID           string
	ProductID    string
	QuantityType *string
	Name         *string
	Price        *float64
	Stock        *int32
	ImageURL     *string
	Description  *string
}

type Package struct {
	ID       string
	Name     string
	ImageURL *string
	UserID   *string
	Items    []*PackageItem
}

type PackageItem struct {
	ID        string
	PackageID string
	VariantID string
	ImageURL  string
	Name      string
	Price     float64
	Quantity  int32
	CreatedAt string
	UpdatedAt string
}

type PackageFilterInput struct {
	ID   *string
	Name *string
}

type PackageSortField string

const (
	PackageSortFieldName      PackageSortField = "NAME"
	PackageSortFieldCreatedAt PackageSortField = "CREATED_AT"
)

type PackageSortInput struct {
	Field     PackageSortField
	Direction SortDirection
}
