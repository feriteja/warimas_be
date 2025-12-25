package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

type Service interface {
	GetProductsByGroup(ctx context.Context, opts ProductQueryOptions) ([]model.ProductByCategory, error)
	GetList(ctx context.Context, opts ProductQueryOptions) (*ProductListResult, error)
	Create(ctx context.Context, input model.NewProduct) (model.Product, error)
	Update(ctx context.Context, input model.UpdateProduct) (model.Product, error)
	CreateVariants(ctx context.Context, input []*model.NewVariant) ([]*model.Variant, error)
	UpdateVariants(ctx context.Context, input []*model.UpdateVariant) ([]*model.Variant, error)
	GetPackages(ctx context.Context, filter *model.PackageFilterInput, sort *model.PackageSortInput, limit, page int32) ([]*model.Package, error)
	GetProductByID(ctx context.Context, productID string) (*model.Product, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetProductsByGroup(ctx context.Context, opts ProductQueryOptions) ([]model.ProductByCategory, error) {
	return s.repo.GetProductsByGroup(ctx, opts)
}

func (s *service) GetList(
	ctx context.Context,
	opts ProductQueryOptions,
) (*ProductListResult, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetProductList"),
	)

	start := time.Now()

	/* ---------- INPUT NORMALIZATION ---------- */

	if opts.Page <= 0 {
		opts.Page = 1
	}

	if opts.Limit <= 0 {
		opts.Limit = 20
	} else if opts.Limit > 100 {
		opts.Limit = 100
	}

	/* ---------- DEBUG INPUT LOG ---------- */

	log.Debug("get product list requested",
		zap.Int32("page", opts.Page),
		zap.Int32("limit", opts.Limit),
		zap.Bool("include_count", opts.IncludeCount),
		zap.Bool("include_disabled", opts.IncludeDisabled),
		zap.Any("filters", map[string]any{
			"category_id": opts.CategoryID,
			"seller_id":   opts.SellerID,
			"seller_name": opts.SellerName,
			"status":      opts.Status,
			"search":      opts.Search,
			"min_price":   opts.MinPrice,
			"max_price":   opts.MaxPrice,
			"in_stock":    opts.InStock,
		}),
	)

	/* ---------- FETCH DATA ---------- */

	products, total, err := s.repo.GetList(ctx, opts)
	if err != nil {
		log.Error("failed to fetch product list",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, err
	}

	/* ---------- SUCCESS LOG ---------- */

	fields := []zap.Field{
		zap.Int("count", len(products)),
		zap.Int32("page", opts.Page),
		zap.Int32("limit", opts.Limit),
		zap.Duration("duration", time.Since(start)),
	}

	if total != nil {
		fields = append(fields, zap.Int("total", *total))
	}

	log.Info("get product list success", fields...)

	/* ---------- RESPONSE ---------- */

	return &ProductListResult{
		Items:      products,
		TotalCount: total,
	}, nil
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

	// ---------- PAGINATION ----------
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	// ---------- AUTH ----------
	role := utils.GetUserRoleFromContext(ctx)
	includeDisabled := role == "ADMIN"

	return s.repo.GetPackages(
		ctx,
		filter,
		sort,
		limit,
		offset,
		includeDisabled,
	)
}

func (s *service) GetProductByID(ctx context.Context, productID string) (*model.Product, error) {

	role := utils.GetUserRoleFromContext(ctx)
	IncludeDisabled := false
	if role == "ADMIN" {
		IncludeDisabled = true
	}

	return s.repo.GetProductByID(ctx, GetProductOptions{
		ProductID:  productID,
		OnlyActive: !IncludeDisabled,
	})
}
