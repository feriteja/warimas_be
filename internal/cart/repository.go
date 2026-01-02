package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type Repository interface {
	UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error
	RemoveFromCart(ctx context.Context, params DeleteFromCartParams) error
	ClearCart(userId uint) error
	GetCartItemByUserAndVariant(
		ctx context.Context,
		userID uint,
		variantID string,
	) (*CartItem, error)
	UpdateCartItemQuantity(
		ctx context.Context,
		cartItemID string,
		quantity uint32,
	) (*CartItem, error)
	CreateCartItem(
		ctx context.Context,
		params CreateCartItemParams,
	) (*CartItem, error)
	GetCartRows(
		ctx context.Context,
		userID uint,
		filter *model.CartFilterInput,
		sort *model.CartSortInput,
		limit, page *uint16,
	) ([]cartRow, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) UpdateCartQuantity(ctx context.Context, updateParams UpdateToCartParams) error {
	// Validate that quantity is positive
	if updateParams.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Update the cart item’s quantity
	res, err := r.db.Exec(`
		UPDATE carts
		SET quantity = $1, updated_at = NOW()
		WHERE user_id = $2 AND variant_id = $3
	`, updateParams.Quantity, updateParams.UserID, updateParams.VariantID)
	if err != nil {
		return err
	}

	// Check if any rows were actually updated
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no matching cart item found to update")
	}

	return nil
}

func (r *repository) RemoveFromCart(ctx context.Context, deleteParams DeleteFromCartParams) error {
	res, err := r.db.Exec(`
		DELETE FROM carts
		WHERE user_id = $1 AND variant_id = $2
	`, deleteParams.UserID, deleteParams.VariantID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no matching cart item found to delete")
	}

	return nil
}

func (r *repository) ClearCart(userID uint) error {
	res, err := r.db.Exec(`DELETE FROM carts
	 WHERE user_id=$1`, userID)
	if err != nil {
		return nil
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("no matching cart item found to delete")
	}

	return nil
}

func (r *repository) GetCartItemByUserAndVariant(
	ctx context.Context,
	userID uint,
	variantID string,
) (*CartItem, error) {

	query := `
	SELECT
		id,
		user_id,
		variant_id,
		quantity,
		created_at,
		updated_at
	FROM carts
	WHERE user_id = $1 AND variant_id = $2
	`

	item := &CartItem{
		Product: &ProductCart{
			Variant: VariantCart{},
		},
	}

	row := r.db.QueryRowContext(ctx, query, userID, variantID)
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Product.Variant.ID,
		&item.Quantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (r *repository) UpdateCartItemQuantity(
	ctx context.Context,
	cartItemID string,
	quantity uint32,
) (*CartItem, error) {

	query := `
	UPDATE carts
	SET quantity = $1,
	    updated_at = NOW()
	WHERE id = $2
	RETURNING
		id,
		user_id,
		variant_id,
		quantity,
		created_at,
		updated_at
	`

	item := &CartItem{
		Product: &ProductCart{
			Variant: VariantCart{},
		},
	}
	row := r.db.QueryRowContext(ctx, query, quantity, cartItemID)
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Product.Variant.ID,
		&item.Quantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (r *repository) CreateCartItem(
	ctx context.Context,
	params CreateCartItemParams,
) (*CartItem, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "CreateCartItem"),
		zap.Uint("user_id", params.UserID),
		zap.String("variant_id", params.VariantID),
	)

	log.Debug("start create cart item")

	query := `
	INSERT INTO carts (
		user_id,
		variant_id,
		quantity
	)
	VALUES ($1, $2, $3)
	RETURNING
		id,
		user_id,
		variant_id,
		quantity,
		created_at,
		updated_at
	`

	item := &CartItem{
		Product: &ProductCart{
			Variant: VariantCart{},
		},
	}

	row := r.db.QueryRowContext(
		ctx,
		query,
		params.UserID,
		params.VariantID,
		params.Quantity,
	)

	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Product.Variant.ID,
		&item.Quantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		log.Error("failed to create cart item", zap.Error(err))
		return nil, err
	}

	log.Info("success create cart item",
		zap.String("cart_item_id", item.ID),
	)

	return item, nil
}

// repository/cart_repo.go
func (r *repository) GetCartRows(
	ctx context.Context,
	userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, page *uint16,
) ([]cartRow, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetCartRows"),
		zap.Uint("user_id", userID),
	)

	start := time.Now()
	log.Info("query started")

	// ---------- pagination ----------
	finalLimit := uint16(20)
	if limit != nil && *limit > 0 {
		finalLimit = *limit
	}
	if finalLimit > 100 {
		finalLimit = 100
	}

	finalPage := uint16(1)
	if page != nil && *page > 0 {
		finalPage = *page
	}

	offset := int((finalPage - 1) * finalLimit)

	log = log.With(
		zap.Uint16("limit", finalLimit),
		zap.Uint16("page", finalPage),
		zap.Int("offset", offset),
	)

	// ---------- where ----------
	where := []string{"c.user_id = $1"}
	args := []any{userID}

	if filter != nil {

		if filter.InStock != nil {
			log = log.With(zap.Bool("filter_in_stock", *filter.InStock))
			if *filter.InStock {
				where = append(where, "v.stock > 0")
			} else {
				where = append(where, "v.stock = 0")
			}
		}

		if filter.Search != nil && *filter.Search != "" {
			log = log.With(zap.String("filter_search", *filter.Search))
			where = append(where,
				fmt.Sprintf(
					"(p.name ILIKE $%d OR v.name ILIKE $%d)",
					len(args)+1,
					len(args)+1,
				),
			)
			args = append(args, "%"+*filter.Search+"%")
		}
	}

	// ---------- sort ----------
	orderBy := "c.created_at DESC"
	if sort != nil {
		field := "c.created_at"
		switch sort.Field {
		case "price":
			field = "v.price"
		case "name":
			field = "p.name"
		case "stock":
			field = "v.stock"
		}

		dir := "DESC"
		if strings.EqualFold(string(sort.Direction), "asc") {
			dir = "ASC"
		}

		orderBy = field + " " + dir

		log = log.With(
			zap.String("sort_field", field),
			zap.String("sort_dir", dir),
		)
	}

	// ---------- query ----------
	query := `
	SELECT
		c.id,
		c.user_id,
		c.quantity,
		c.created_at,
		c.updated_at,

		p.id,
		p.name,
		p.seller_id,
		COALESCE(s.name, 'UNKNOWN'),
		p.category_id,
		p.subcategory_id,
		p.slug,
		p.status,
		p.imageurl,

		v.id,
		v.name,
		v.product_id,
		v.quantity_type,
		v.price,
		v.stock,
		v.imageurl
	FROM carts c
	JOIN variants v ON c.variant_id = v.id
	JOIN products p ON v.product_id = p.id
	LEFT JOIN sellers s ON p.seller_id = s.id
	WHERE ` + strings.Join(where, " AND ") + `
	ORDER BY ` + orderBy + `
	LIMIT $` + fmt.Sprint(len(args)+1) + `
	OFFSET $` + fmt.Sprint(len(args)+2)

	args = append(args, finalLimit, offset)

	log.Debug("executing query",
		zap.Int("args_count", len(args)),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("query failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, err
	}
	defer rows.Close()

	result := make([]cartRow, 0, finalLimit)

	for rows.Next() {
		var row cartRow
		if err := rows.Scan(
			&row.CartID,
			&row.UserID,
			&row.Quantity,
			&row.CreatedAt,
			&row.UpdatedAt,

			&row.ProductID,
			&row.ProductName,
			&row.SellerID,
			&row.SellerName,
			&row.CategoryID,
			&row.SubcategoryID,
			&row.Slug,
			&row.Status,
			&row.ProductImageURL,

			&row.VariantID,
			&row.VariantName,
			&row.VariantProductID,
			&row.QuantityType,
			&row.Price,
			&row.Stock,
			&row.VariantImageURL,
		); err != nil {
			log.Error("row scan failed",
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
			return nil, err
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, err
	}

	log.Info("query success",
		zap.Int("rows", len(result)),
		zap.Duration("duration", time.Since(start)),
	)

	return result, nil
}
