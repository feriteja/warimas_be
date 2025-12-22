package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/graph/model"
	servicepkg "warimas-be/internal/service"
	"warimas-be/internal/utils"
)

type Service interface {
	GetProductsByGroup(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.ProductByCategory, error)
	GetList(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]*model.Product, error)
	Create(ctx context.Context, input model.NewProduct) (model.Product, error)
	Update(ctx context.Context, input model.UpdateProduct) (model.Product, error)
	CreateVariants(ctx context.Context, input []*model.NewVariant) ([]*model.Variant, error)
	UpdateVariants(ctx context.Context, input []*model.UpdateVariant) ([]*model.Variant, error)
	GetPackages(ctx context.Context, filter *model.PackageFilterInput, sort *model.PackageSortInput, limit, page int32) ([]*model.Package, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetProductsByGroup(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]model.ProductByCategory, error) {
	return s.repo.GetProductsByGroup(ctx, opts)
}

func (s *service) GetList(ctx context.Context, opts servicepkg.ProductQueryOptions) ([]*model.Product, error) {
	return s.repo.GetList(ctx, opts)
}

func (s *service) Create(ctx context.Context, input model.NewProduct) (model.Product, error) {
	if input.Name == "" {
		return model.Product{}, errors.New("name cannot be empty")
	}

	sellerID, ok := ctx.Value(utils.SellerIDKey).(string)
	if !ok || sellerID == "" {
		return model.Product{}, errors.New("unauthorized: seller ID not found in context")
	}

	return s.repo.Create(ctx, input, sellerID)
}

func (s *service) Update(
	ctx context.Context,
	input model.UpdateProduct,
) (model.Product, error) {

	if input.ID == "" {
		return model.Product{}, errors.New("product id is required")
	}

	// Validate only provided fields
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return model.Product{}, errors.New("name cannot be empty")
	}

	sellerID, ok := ctx.Value(utils.SellerIDKey).(string)
	if !ok || sellerID == "" {
		return model.Product{}, errors.New("unauthorized")
	}

	// Ensure at least one field is updated
	if !utils.HasAnyUpdateProductField(input) {
		return model.Product{}, errors.New("no fields to update")
	}

	return s.repo.Update(ctx, input, sellerID)
}

func (s *service) CreateVariants(
	ctx context.Context,
	input []*model.NewVariant,
) ([]*model.Variant, error) {

	if len(input) == 0 {
		return nil, errors.New("variant input cannot be empty")
	}

	sellerID, ok := ctx.Value(utils.SellerIDKey).(string)
	if !ok || sellerID == "" {
		return nil, errors.New("unauthorized: seller ID not found in context")
	}

	return s.repo.BulkCreateVariants(ctx, input, sellerID)
}

func (s *service) UpdateVariants(
	ctx context.Context,
	input []*model.UpdateVariant,
) ([]*model.Variant, error) {

	if len(input) == 0 {
		return nil, errors.New("variant input cannot be empty")
	}

	sellerID, ok := ctx.Value(utils.SellerIDKey).(string)
	if !ok || sellerID == "" {
		return nil, errors.New("unauthorized")
	}

	for i, v := range input {
		if v == nil {
			return nil, fmt.Errorf("variant at index %d is nil", i)
		}

		if v.ID == "" {
			return nil, fmt.Errorf("variant id is required at index %d", i)
		}

		if v.ProductID == "" {
			return nil, fmt.Errorf("product id is required at index %d", i)
		}

		// Validate partial fields
		if v.Name != nil && strings.TrimSpace(*v.Name) == "" {
			return nil, fmt.Errorf("variant name cannot be empty at index %d", i)
		}

		if v.Price != nil && *v.Price <= 0 {
			return nil, fmt.Errorf("price must be positive at index %d", i)
		}

		if v.Stock != nil && *v.Stock < 0 {
			return nil, fmt.Errorf("stock cannot be negative at index %d", i)
		}

		if !utils.HasAnyVariantUpdateField(v) {
			return nil, fmt.Errorf("no fields to update at index %d", i)
		}
	}

	return s.repo.BulkUpdateVariants(ctx, input, sellerID)
}

func (s *service) GetPackages(
	ctx context.Context,
	filter *model.PackageFilterInput,
	sort *model.PackageSortInput,
	limit, page int32,
) ([]*model.Package, error) {

	offset := page * limit

	return s.repo.GetPackages(ctx, filter, sort, limit, offset)
}
