package graph

import (
	"context"
	"warimas-be/internal/graph/model"
)

func (r *queryResolver) Category(ctx context.Context,
	filter *string,
	limit, offset *int32) ([]*model.Category, error) {

	c, err := r.CategorySvc.GetCategories(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *queryResolver) Subcategory(ctx context.Context,
	filter *string, categoryID string,
	limit, offset *int32) ([]*model.Subcategory, error) {

	s, err := r.CategorySvc.GetSubcategories(ctx, categoryID, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (r *mutationResolver) AddCategory(ctx context.Context,
	name string) (*model.Category, error) {

	c, err := r.CategorySvc.AddCategory(ctx, name)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *mutationResolver) AddSubcategory(ctx context.Context, categoryID string,
	name string) (*model.Subcategory, error) {

	c, err := r.CategorySvc.AddSubcategory(ctx, categoryID, name)
	if err != nil {
		return nil, err
	}

	return c, nil
}
