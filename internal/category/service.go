package category

import (
	"context"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

// Service defines the business logic for carts.
type Service interface {
	GetCategories(ctx context.Context, filter *string, limit, offset *int32) ([]*Category, int64, error)
	AddCategory(ctx context.Context, name string) (*Category, error)
	GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, offset *int32) ([]*Subcategory, int64, error)
	AddSubcategory(ctx context.Context, categoryID string, name string) (*Subcategory, error)
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
func (s *service) GetCategories(ctx context.Context, filter *string, limit, offset *int32) ([]*Category, int64, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetCategories"),
	)
	log.Info("GetCategories started")

	// 1. Get parent categories
	categories, total, err := s.repo.GetCategories(ctx, filter, limit, offset)
	if err != nil {
		log.Error("failed to get categories", zap.Error(err))
		return nil, 0, err
	}

	if len(categories) == 0 {
		log.Info("no categories found")
		return []*Category{}, 0, nil
	}

	// 2. Collect category IDs
	categoryIDs := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryIDs = append(categoryIDs, c.ID)
	}

	// 3. Fetch all subcategories for the collected IDs in one query
	subcategoriesMap, err := s.repo.GetSubcategoriesByIds(ctx, categoryIDs)
	if err != nil {
		log.Error("failed to get subcategories by ids", zap.Error(err))
		return nil, 0, err
	}

	// 4. Attach subcategories to their parent categories
	for _, c := range categories {
		c.Subcategories = subcategoriesMap[c.ID]
	}

	log.Info("GetCategories success", zap.Int("count", len(categories)))
	return categories, total, nil
}

// GetSubCategories retrieves all sub-categories
func (s *service) GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, offset *int32) ([]*Subcategory, int64, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetSubcategories"),
		zap.String("category_id", categoryID),
	)
	log.Info("GetSubcategories started")

	subcategories, total, err := s.repo.GetSubcategories(ctx, categoryID, filter, limit, offset)
	if err != nil {
		log.Error("failed to get subcategories", zap.Error(err))
		return nil, 0, err
	}

	log.Info("GetSubcategories success", zap.Int("count", len(subcategories)))
	return subcategories, total, nil
}

func (s *service) AddCategory(ctx context.Context, name string) (*Category, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "AddCategory"),
		zap.String("name", name),
	)
	log.Info("AddCategory started")

	category, err := s.repo.AddCategory(ctx, name)
	if err != nil {
		log.Error("failed to add category", zap.Error(err))
		return nil, err
	}

	log.Info("AddCategory success", zap.String("category_id", category.ID))
	return category, nil
}

func (s *service) AddSubcategory(ctx context.Context, categoryID string, name string) (*Subcategory, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "AddSubcategory"),
		zap.String("category_id", categoryID),
		zap.String("name", name),
	)
	log.Info("AddSubcategory started")

	subcategory, err := s.repo.AddSubcategory(ctx, categoryID, name)
	if err != nil {
		log.Error("failed to add subcategory", zap.Error(err))
		return nil, err
	}

	log.Info("AddSubcategory success", zap.String("subcategory_id", subcategory.ID))
	return subcategory, nil
}
