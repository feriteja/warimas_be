package graph

import (
	"context"
	"errors"
	"fmt"
	"warimas-be/internal/graph/model"
)

// CreateProduct is the resolver for the createProduct field.
func (r *mutationResolver) CreateProduct(ctx context.Context, input model.NewProduct) (*model.Product, error) {

	_, ok := GetUserIDFromContext(ctx)
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

// Products is the resolver for the products field.
func (r *queryResolver) Products(ctx context.Context) ([]*model.Product, error) {
	products, err := r.ProductSvc.GetAll()
	if err != nil {
		return nil, err
	}

	var result []*model.Product
	for _, p := range products {
		result = append(result, &model.Product{
			ID:    fmt.Sprint(p.ID),
			Name:  p.Name,
			Price: p.Price,
			Stock: int32(p.Stock),
		})
	}

	return result, nil
}
