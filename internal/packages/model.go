package packages

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "ASC"
	SortDirectionDesc SortDirection = "DESC"
)

type PackageFilterInput struct {
	ID   *string
	Name *string
	Type *string
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

type Package struct {
	ID        string
	Name      string
	Type      string
	ImageURL  *string
	UserID    *uint
	Items     []*PackageItem
	CreatedAt string
	UpdatedAt string
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

type CreatePackageInput struct {
	Name  string
	Type  string
	Items []CreatePackageItemInput
}

type CreatePackageItemInput struct {
	VariantID string
	Quantity  int32
}
