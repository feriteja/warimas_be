package product

import (
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"
)

func MapSortField(f *model.ProductSortField) ProductSortField {
	if f == nil {
		return ProductSortFieldCreatedAt
	}

	switch *f {
	case model.ProductSortFieldPrice:
		return ProductSortFieldPrice
	case model.ProductSortFieldName:
		return ProductSortFieldName
	default:
		return ProductSortFieldCreatedAt
	}
}

func MapProductByCategoryToGraphQL(
	e ProductByCategory,
) *model.ProductByCategory {

	products := make([]*model.Product, 0, len(e.Products))
	for _, p := range e.Products {
		products = append(products, MapProductToGraphQL(p))
	}

	return &model.ProductByCategory{
		CategoryName:  &e.CategoryName,
		TotalProducts: int32(e.TotalProducts),
		Products:      products,
	}
}

func MapSortDirection(d *model.SortDirection) SortDirection {
	if d == nil {
		return SortDirectionDesc
	}

	if *d == model.SortDirectionAsc {
		return SortDirectionAsc
	}
	return SortDirectionDesc
}

func MapProductToGraphQL(p *Product) *model.Product {
	status := p.Status

	variants := make([]*model.Variant, 0, len(p.Variants))
	for _, v := range p.Variants {
		variants = append(variants, MapVariantToGraphQL(v))

	}

	return &model.Product{
		ID:              p.ID,
		Name:            p.Name,
		SellerID:        p.SellerID,
		SellerName:      p.SellerName,
		CategoryID:      p.CategoryID,
		CategoryName:    p.CategoryName,
		SubcategoryID:   p.SubcategoryID,
		SubcategoryName: p.SubcategoryName,
		Slug:            p.Slug,
		ImageURL:        p.ImageURL,
		Description:     p.Description,
		Status:          &status,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       utils.FormatTimePtr(p.UpdatedAt),
		Variants:        variants,
	}
}

func MapVariantToGraphQL(v *Variant) *model.Variant {
	if v == nil {
		return nil
	}

	imageURL := ""
	if v.ImageURL != "" {
		imageURL = v.ImageURL
	}

	return &model.Variant{
		ID:           v.ID,
		Name:         v.Name,
		ProductID:    v.ProductID,
		QuantityType: v.QuantityType,
		Price:        v.Price,
		Stock:        int32(v.Stock),
		ImageURL:     imageURL,
		Description:  v.Description,
		CategoryID:   nil,
		CreatedAt:    v.CreatedAt,
	}
}
