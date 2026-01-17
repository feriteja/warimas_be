package graph

import (
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/product"
	"warimas-be/internal/utils"
)

func MapSortField(f *model.ProductSortField) product.ProductSortField {
	if f == nil {
		return product.ProductSortFieldCreatedAt
	}

	switch *f {
	case model.ProductSortFieldPrice:
		return product.ProductSortFieldPrice
	case model.ProductSortFieldName:
		return product.ProductSortFieldName
	default:
		return product.ProductSortFieldCreatedAt
	}
}

func MapProductByCategoryToGraphQL(
	e product.ProductByCategory,
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

func MapSortDirection(d *model.SortDirection) product.SortDirection {
	if d == nil {
		return product.SortDirectionDesc
	}

	if *d == model.SortDirectionAsc {
		return product.SortDirectionAsc
	}
	return product.SortDirectionDesc
}

func MapProductToGraphQL(p *product.Product) *model.Product {
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

func MapVariantToGraphQL(v *product.Variant) *model.Variant {
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

func MapNewProductInput(input model.NewProduct) product.NewProductInput {
	return product.NewProductInput{
		Name:          input.Name,
		ImageURL:      input.ImageURL,
		Description:   input.Description,
		CategoryID:    input.CategoryID,
		SubcategoryID: input.SubcategoryID,
	}
}

func MapUpdateProductInput(input model.UpdateProduct) product.UpdateProductInput {
	return product.UpdateProductInput{
		ID:            input.ID,
		Name:          input.Name,
		ImageURL:      input.ImageURL,
		Description:   input.Description,
		CategoryID:    input.CategoryID,
		SubcategoryID: input.SubcategoryID,
		Status:        input.Status,
	}
}
