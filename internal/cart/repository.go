package cart

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/product"

	"go.uber.org/zap"
)

type Repository interface {
	AddToCart(ctx context.Context, userID uint, variantId string, quantity uint) (*CartItem, error)
	GetCart(ctx context.Context, userID uint,
		filter *model.CartFilterInput,
		sort *model.CartSortInput,
		limit, offset *int32) ([]*model.CartItem, error)
	UpdateCartQuantity(userID uint, productID string, quantity int) error
	RemoveFromCart(userID uint, productID string) error
	ClearCart(userId uint) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AddToCart(ctx context.Context, userID uint, variantId string, quantity uint) (*CartItem, error) {
	log := logger.FromCtx(ctx).With(
		zap.Uint("user_id", userID),
		zap.String("variant_id", variantId),
		zap.Uint("qty", quantity),
	)

	log.Info("AddToCart started")

	// 1️⃣ Load variant
	var v struct {
		ID        string
		ProductID string
		Price     float64
		Stock     int
	}

	err := r.db.QueryRow(`
		SELECT id, product_id, price, stock
		FROM variants
		WHERE id = $1
	`, variantId).Scan(&v.ID, &v.ProductID, &v.Price, &v.Stock)

	if err == sql.ErrNoRows {
		log.Warn("variant not found")
		return nil, errors.New("variant not found")
	}
	if err != nil {
		log.Error("database error loading variant", zap.Error(err))
		return nil, err
	}

	log.Info("variant loaded",
		zap.String("product_id", v.ProductID),
		zap.Float64("price", v.Price),
		zap.Int("stock", v.Stock),
	)

	// 2️⃣ Check if item already in cart
	var existingQty int
	err = r.db.QueryRow(`
		SELECT quantity FROM carts
		WHERE user_id = $1 AND variant_id = $2
	`, userID, variantId).Scan(&existingQty)

	if err == sql.ErrNoRows {
		// Insert new
		log.Info("inserting new cart item")

		_, err = r.db.Exec(`
			INSERT INTO carts (user_id, variant_id, quantity, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, userID, variantId, quantity)

		if err != nil {
			log.Error("failed to insert cart item", zap.Error(err))
			return nil, err
		}

	} else if err == nil {
		// Update existing
		newQty := existingQty + int(quantity)

		if newQty > v.Stock {
			log.Warn("quantity exceeds stock",
				zap.Int("requested_qty", newQty),
				zap.Int("stock", v.Stock),
			)
			return nil, errors.New("quantity exceeds stock")
		}

		log.Info("updating cart quantity",
			zap.Int("existing_qty", existingQty),
			zap.Int("new_qty", newQty),
		)

		_, err = r.db.Exec(`
			UPDATE carts
			SET quantity = $1, updated_at = NOW()
			WHERE user_id = $2 AND variant_id = $3
		`, newQty, userID, variantId)

		if err != nil {
			log.Error("failed to update cart item", zap.Error(err))
			return nil, err
		}

	} else {
		log.Error("database error checking existing cart item", zap.Error(err))
		return nil, err
	}

	// 3️⃣ Fetch final state
	var (
		cartID       uint
		userIDStr    string
		variantIDStr string
		qty          int
		createdAt    time.Time
		updatedAt    time.Time
		pID, pName   string
		vID          string
		vPrice       float64
		vStock       int
	)

	err = r.db.QueryRow(`
		SELECT 
			c.id, c.user_id, c.variant_id, c.quantity, c.created_at, c.updated_at,
			v.id, v.price, v.stock,
			p.id, p.name
		FROM carts c
		JOIN variants v ON c.variant_id = v.id
		JOIN products p ON v.product_id = p.id
		WHERE c.user_id = $1 AND c.variant_id = $2
	`, userID, variantId).Scan(
		&cartID, &userIDStr, &variantIDStr, &qty,
		&createdAt, &updatedAt,
		&vID, &vPrice, &vStock,
		&pID, &pName,
	)

	if err != nil {
		log.Error("failed to load final cart item", zap.Error(err))
		return nil, err
	}

	ci := &CartItem{
		ID:        cartID,
		UserID:    userIDStr,
		Quantity:  qty,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Product: product.Product{
			ID:   pID,
			Name: pName,
			Variants: []*product.Variant{
				{
					ID:    vID,
					Price: vPrice,
					Stock: vStock,
				},
			},
		},
	}

	log.Info("AddToCart success",
		zap.Uint("cart_item_id", cartID),
		zap.Int("final_qty", qty),
	)

	return ci, nil
}

func (r *repository) GetCart(
	ctx context.Context,
	userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, offset *int32,
) ([]*model.CartItem, error) {

	log := logger.FromCtx(ctx)

	log.Info("GetCart started",
		zap.Uint("user_id", userID),
	)

	query := `
        SELECT
            c.id AS cart_id,
            c.user_id AS cart_user_id,
            c.quantity AS cart_quantity,
            c.created_at AS cart_created_at,
            c.updated_at AS cart_updated_at,

            p.id AS product_id,
            p.name AS product_name,
            p.seller_id AS product_seller_id,
            p.category_id AS product_category_id,
            p.slug AS product_slug,
            p.imageurl AS product_imageurl,

            v.id AS variant_id,
            v.name AS variant_name,
            v.product_id AS variant_product_id,
            v.quantity_type AS variant_quantity_type,
            v.price AS variant_price,
            v.stock AS variant_stock,
            v.imageurl AS variant_imageurl,
            v.subcategory_id AS variant_subcategory_id
        FROM carts c
        JOIN variants v ON c.variant_id = v.id
        JOIN products p ON v.product_id = p.id
        WHERE c.user_id = $1
    `

	log.Debug("Executing GetCart query")

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Error("DB query failed in GetCart",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var items []*model.CartItem
	rowCount := 0

	for rows.Next() {

		cart := &model.CartItem{
			Product: &model.ProductCart{},
		}
		variant := &model.VariantCart{}

		var categoryID *string
		var subcategoryID *string
		var productImageUrl sql.NullString
		var variantImageUrl sql.NullString

		err := rows.Scan(
			&cart.ID, &cart.UserID, &variant.Qty, &cart.CreatedAt, &cart.UpdatedAt,

			&cart.Product.ID,
			&cart.Product.Name,
			&cart.Product.SellerID,
			&categoryID,
			&cart.Product.Slug,
			&productImageUrl,

			&variant.ID,
			&variant.Name,
			&variant.ProductID,
			&variant.QuantityType,
			&variant.Price,
			&variant.Stock,
			&variantImageUrl,
			&subcategoryID,
		)

		if err != nil {
			log.Error("Failed to scan row in GetCart",
				zap.Uint("user_id", userID),
				zap.Error(err),
			)
			return nil, err
		}

		// Assign nullable fields
		cart.Product.ImageURL = productImageUrl.String
		cart.Product.CategoryID = categoryID
		variant.SubcategoryID = subcategoryID
		variant.ImageURL = variantImageUrl.String

		cart.Product.Variants = []*model.VariantCart{variant}

		items = append(items, cart)
		rowCount++
	}

	log.Info("GetCart completed",
		zap.Uint("user_id", userID),
		zap.Int("row_count", rowCount),
	)

	return items, nil
}

func (r *repository) UpdateCartQuantity(userID uint, productID string, quantity int) error {
	// Validate that quantity is positive
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Update the cart item’s quantity
	res, err := r.db.Exec(`
		UPDATE carts
		SET quantity = $1, updated_at = NOW()
		WHERE user_id = $2 AND variant_id = $3
	`, quantity, userID, productID)
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

func (r *repository) RemoveFromCart(userID uint, productID string) error {
	res, err := r.db.Exec(`
		DELETE FROM carts
		WHERE user_id = $1 AND variant_id = $2
	`, userID, productID)
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
