package category

import (
	"context"
	"warimas-be/internal/graph/model"
)

// Service defines the business logic for carts.
type Service interface {
	GetCategories(ctx context.Context, filter *string, limit, offset *int32) ([]*model.Category, error)
	AddCategory(ctx context.Context, name string) (*model.Category, error)
	GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, offset *int32) ([]*model.Subcategory, error)
	AddSubcategory(ctx context.Context, categoryID string, name string) (*model.Subcategory, error)
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new cart service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetCategories retrieves all categories
func (s *service) GetCategories(ctx context.Context, filter *string, limit, offset *int32) ([]*model.Category, error) {
	return s.repo.GetCategories(ctx, filter, limit, offset)
}

// GetSubCategories retrieves all sub-categories
func (s *service) GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, offset *int32) ([]*model.Subcategory, error) {
	return s.repo.GetSubcategories(ctx, categoryID, filter, limit, offset)
}

func (s *service) AddCategory(ctx context.Context, name string) (*model.Category, error) {
	return s.repo.AddCategory(ctx, name)
}

func (s *service) AddSubcategory(ctx context.Context, categoryID string, name string) (*model.Subcategory, error) {
	return s.repo.AddSubcategory(ctx, categoryID, name)
}
