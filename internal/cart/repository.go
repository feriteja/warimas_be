package cart

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

type Repository interface {
	UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error
	RemoveFromCart(ctx context.Context, params DeleteFromCartParams) error
	ClearCart(ctx context.Context, userId uint) error
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

func (r *repository) UpdateCartQuantity(
	ctx context.Context,
	updateParams UpdateToCartParams,
) error {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "UpdateCartQuantity"),
		zap.Uint32("user_id", updateParams.UserID),
		zap.String("variant_id", updateParams.VariantID),
		zap.Uint32("quantity", updateParams.Quantity),
	)

	// Validate quantity
	if updateParams.Quantity <= 0 {
		log.Warn("invalid quantity provided")
		return ErrInvalidQuantity
	}

	// Execute update
	res, err := r.db.ExecContext(ctx, `
		UPDATE carts
		SET quantity = $1, updated_at = NOW()
		WHERE user_id = $2 AND variant_id = $3
	`,
		updateParams.Quantity,
		updateParams.UserID,
		updateParams.VariantID,
	)
	if err != nil {
		log.Error("failed to execute update cart query", zap.Error(err))
		return ErrFailedUpdateCart
	}

	// Check affected rows
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to read rows affected", zap.Error(err))
		return ErrFailedUpdateCart
	}

	if rowsAffected == 0 {
		log.Info("no cart item found to update")
		return ErrCartItemNotFound
	}

	log.Info("cart quantity updated successfully")
	return nil
}
func (r *repository) RemoveFromCart(
	ctx context.Context,
	deleteParams DeleteFromCartParams,
) error {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "RemoveFromCart"),
		zap.Uint32("user_id", deleteParams.UserID),
		zap.String("variant_id", deleteParams.VariantID),
	)

	res, err := r.db.ExecContext(ctx, `
		DELETE FROM carts
		WHERE user_id = $1 AND variant_id = $2
	`,
		deleteParams.UserID,
		deleteParams.VariantID,
	)
	if err != nil {
		log.Error("failed to execute delete cart query", zap.Error(err))
		return ErrFailedRemoveCart
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to read rows affected", zap.Error(err))
		return ErrFailedRemoveCart
	}

	if rowsAffected == 0 {
		log.Info("no cart item found to delete")
		return ErrCartItemNotFound
	}

	log.Info("cart item removed successfully")
	return nil
}

func (r *repository) ClearCart(
	ctx context.Context,
	userID uint,
) error {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "ClearCart"),
		zap.Uint("user_id", userID),
	)

	res, err := r.db.ExecContext(ctx, `
		DELETE FROM carts
		WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Error("failed to execute clear cart query", zap.Error(err))
		return ErrFailedClearCart
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to read rows affected", zap.Error(err))
		return ErrFailedClearCart
	}

	if rowsAffected == 0 {
		log.Info("cart already empty")
		return ErrCartEmpty
	}

	log.Info("cart cleared successfully", zap.Int64("items_removed", rowsAffected))
	return nil
}

func (r *repository) GetCartItemByUserAndVariant(
	ctx context.Context,
	userID uint,
	variantID string,
) (*CartItem, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetCartItemByUserAndVariant"),
		zap.Uint("user_id", userID),
		zap.String("variant_id", variantID),
	)

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
		log.Info("cart item not found")
		return nil, nil
	}

	if err != nil {
		log.Error("failed to scan cart item", zap.Error(err))
		return nil, ErrFailedGetCartItem
	}

	log.Debug("cart item fetched successfully")
	return item, nil
}
func (r *repository) UpdateCartItemQuantity(
	ctx context.Context,
	cartItemID string,
	quantity uint32,
) (*CartItem, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "UpdateCartItemQuantity"),
		zap.String("cart_item_id", cartItemID),
		zap.Uint32("quantity", quantity),
	)

	// Validate quantity
	if quantity == 0 {
		log.Warn("invalid quantity provided")
		return nil, ErrInvalidQuantity
	}

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

	if err == sql.ErrNoRows {
		log.Info("cart item not found for update")
		return nil, ErrCartItemNotFound
	}

	if err != nil {
		log.Error("failed to update cart item quantity", zap.Error(err))
		return nil, ErrFailedUpdateCart
	}

	log.Info("cart item quantity updated successfully")
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
		zap.Uint32("quantity", params.Quantity),
	)

	if params.Quantity == 0 {
		log.Warn("invalid quantity provided")
		return nil, ErrInvalidQuantity
	}

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
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == pq.ErrorCode(PgUniqueViolation) {
			log.Info("cart item already exists",
				zap.String("constraint", pqErr.Constraint),
			)
			return nil, ErrCartItemAlreadyExist
		}

		log.Error("failed to create cart item", zap.Error(err))
		return nil, ErrFailedCreateCartItem
	}

	log.Info("cart item created successfully",
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

	// Validate input
	if userID == 0 {
		log.Warn("invalid user_id provided")
		return nil, ErrFailedGetCartRows
	}

	start := time.Now()
	log.Debug("cart query started")

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
			where = append(
				where,
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

	log.Debug("executing cart query",
		zap.Int("args_count", len(args)),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("cart query execution failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, ErrFailedGetCartRows
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
			log.Error("cart row scan failed",
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
			return nil, ErrFailedGetCartRows
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		log.Error("cart rows iteration failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(start)),
		)
		return nil, ErrFailedGetCartRows
	}

	log.Info("cart query success",
		zap.Int("rows", len(result)),
		zap.Duration("duration", time.Since(start)),
	)

	return result, nil
}
