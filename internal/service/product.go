package service

import "warimas-be/internal/graph/model"

type ProductQueryOptions struct {
	Filter          *model.ProductFilterInput
	Sort            *model.ProductSortInput
	Limit           *int32
	Page            *int32
	IncludeDisabled bool
}
