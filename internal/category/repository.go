package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

type Repository interface {
	GetCategories(ctx context.Context, filter *string, limit, page *int32) ([]*model.Category, error)
	AddCategory(ctx context.Context, name string) (*model.Category, error)
	GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, page *int32) ([]*model.Subcategory, error)
	AddSubcategory(ctx context.Context, categoryID string, name string) (*model.Subcategory, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetCategories(
	ctx context.Context,
	filter *string,
	limit *int32,
	page *int32,
) ([]*model.Category, error) {

	// ---------- DEFAULTS ----------
	finalLimit := int32(20)
	finalPage := int32(1)

	if limit != nil && *limit > 0 {
		finalLimit = *limit
	}
	if page != nil && *page > 0 {
		finalPage = *page
	}

	finalOffset := (finalPage - 1) * finalLimit

	log := logger.FromCtx(ctx).With(
		zap.String("filter", utils.PtrString(filter)),
		zap.Int32("limit", finalLimit),
		zap.Int32("page", finalPage),
		zap.Int32("offset", finalOffset),
	)
	log.Info("GetCategories started")

	// ---------- BASE QUERY ----------
	query := `
		SELECT
			c.id,
			c.name
		FROM category c
	`

	where := []string{}
	args := []interface{}{}

	// ---------- FILTER ----------
	if filter != nil && *filter != "" {
		where = append(where, fmt.Sprintf("c.name ILIKE $%d", len(args)+1))
		args = append(args, "%"+*filter+"%")
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	// ---------- ORDER ----------
	query += " ORDER BY c.name ASC"

	// ---------- PAGINATION ----------
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, finalLimit, finalOffset)

	log.Debug("Executing GetCategories query",
		zap.String("query", query),
		zap.Any("args", args),
	)

	// ---------- EXECUTE ----------
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("DB query failed GetCategories", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var categories []*model.Category

	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Error("Row scan failed", zap.Error(err))
			return nil, err
		}
		categories = append(categories, &c)
	}

	if err := rows.Err(); err != nil {
		log.Error("Rows iteration failed", zap.Error(err))
		return nil, err
	}

	return categories, nil
}

func (r *repository) GetSubcategories(
	ctx context.Context,
	categoryID string,
	filter *string,
	limit *int32,
	page *int32,
) ([]*model.Subcategory, error) {

	if categoryID == "" {
		return nil, errors.New("categoryID is required")
	}

	// Pagination Defaults
	finalLimit := int32(20)
	if limit != nil && *limit > 0 {
		finalLimit = *limit
	}

	finalPage := int32(1)
	if page != nil && *page > 0 {
		finalPage = *page
	}
	finalOffset := (finalPage - 1) * finalLimit

	// Base Query & Args
	query := `SELECT s.id, s.category_id, s.name FROM subcategories s`
	args := []interface{}{categoryID}
	where := []string{"s.category_id = $1"}

	// Optional Filter
	if filter != nil && *filter != "" {
		where = append(where, fmt.Sprintf("s.name ILIKE $%d", len(args)+1))
		args = append(args, "%"+*filter+"%")
	}

	// Construct Final Query
	query += " WHERE " + strings.Join(where, " AND ")
	query += " ORDER BY s.name ASC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, finalLimit, finalOffset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice
	subcategories := make([]*model.Subcategory, 0, finalLimit)

	for rows.Next() {
		var s model.Subcategory
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		subcategories = append(subcategories, &s)
	}

	return subcategories, nil
}

func (r *repository) AddCategory(
	ctx context.Context,
	name string,
) (*model.Category, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("category_name", name),
	)
	log.Info("AddCategory started")

	if name == "" {
		log.Warn("AddCategory validation failed: empty name")
		return nil, errors.New("category name cannot be empty")
	}

	query := `
		INSERT INTO category (name)
		VALUES ($1)
		RETURNING id, name
	`

	log.Debug("Executing AddCategory query",
		zap.String("query", query),
	)

	var c model.Category

	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&c.ID, &c.Name)
	if err != nil {
		log.Error("AddCategory DB query failed", zap.Error(err))
		return nil, fmt.Errorf("add category failed: %w", err)
	}

	log.Info("AddCategory success",
		zap.String("category_id", c.ID),
	)

	return &c, nil
}

func (r *repository) AddSubcategory(
	ctx context.Context,
	categoryID string,
	name string,
) (*model.Subcategory, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("category_id", categoryID),
		zap.String("subcategory_name", name),
	)
	log.Info("AddSubCategory started")

	if categoryID == "" {
		log.Warn("AddSubCategory validation failed: empty categoryID")
		return nil, errors.New("categoryID cannot be empty")
	}

	if name == "" {
		log.Warn("AddSubCategory validation failed: empty name")
		return nil, errors.New("subcategory name cannot be empty")
	}

	query := `
		INSERT INTO subcategories (category_id, name)
		VALUES ($1, $2)
		RETURNING id, category_id, name
	`

	log.Debug("Executing AddSubCategory query",
		zap.String("query", query),
	)

	var sc model.Subcategory

	err := r.db.QueryRowContext(ctx, query, categoryID, name).
		Scan(&sc.ID, &sc.CategoryID, &sc.Name)
	if err != nil {
		log.Error("AddSubCategory DB query failed", zap.Error(err))
		return nil, fmt.Errorf("add subcategory failed: %w", err)
	}

	log.Info("AddSubCategory success",
		zap.String("subcategory_id", sc.ID),
	)

	return &sc, nil
}
