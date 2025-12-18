package graph

import (
	"context"
	"errors"
	"fmt"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/service"
	"warimas-be/internal/utils"
)

// CreateProduct is the resolver for the createProduct field.
func (r *mutationResolver) CreateProduct(ctx context.Context, input model.NewProduct) (*model.Product, error) {

	_, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthorized: please login first")
	}

	p, err := r.ProductSvc.Create(input.Name, input.Price, int(input.Stock))
	if err != nil {
		return nil, err
	}

	return &model.Product{
		ID:    fmt.Sprint(p.ID),
		Name:  p.Name,
		Price: p.Price,
		Stock: int32(p.Stock),
	}, nil
}

func (r *mutationResolver) CreateVariants(ctx context.Context, input []*model.NewVariant) ([]*model.Variant, error) {

	_, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthorized: please login first")
	}

	v, err := r.ProductSvc.CreateVariants(ctx, input)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Products is the resolver for the products field.
func (r *queryResolver) ProductsHome(
	ctx context.Context,
	filter *model.ProductFilterInput,
	sort *model.ProductSortInput,
	limit, offset *int32,
) ([]*model.CategoryProduct, error) {

	// 1. Prepare service options
	opts := service.ProductQueryOptions{
		Filter: filter,
		Sort:   sort,
		Limit:  limit,
		Offset: offset,
	}

	// 2. Fetch grouped products
	grouped, err := r.ProductSvc.GetAll(ctx, opts)
	if err != nil {
		return nil, err
	}

	// 3. Convert service output -> GraphQL response
	result := make([]*model.CategoryProduct, 0, len(grouped))

	for _, g := range grouped {
		if len(g.Products) == 0 {
			continue // skip empty categories
		}

		result = append(result, &model.CategoryProduct{
			CategoryName:  g.CategoryName,
			TotalProducts: g.TotalProducts,
			Products:      g.Products,
		})
	}

	return result, nil
}

func (r *queryResolver) PackageRecomamendation(
	ctx context.Context,
	filter *model.PackageFilterInput,
	sort *model.PackageSortInput,
	limit *int32,
	page *int32,
) (*model.PackageResponse, error) {

	limitVal := int32(20)
	pageVal := int32(0)

	if limit != nil {
		limitVal = *limit
	}
	if page != nil {
		pageVal = *page
	}

	data, err := r.ProductSvc.GetPackages(ctx, filter, sort, limitVal, pageVal)
	if err != nil {
		msg := err.Error()
		return &model.PackageResponse{
			Success: false,
			Message: &msg,
			Data:    nil,
		}, nil
	}

	return &model.PackageResponse{
		Success: true,
		Message: nil,
		Data:    data,
	}, nil
}
