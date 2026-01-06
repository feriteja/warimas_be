package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/product"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Repository interface {
	CreateOrder(userID uint) (*Order, error)
	GetOrders(ctx context.Context, filter *model.OrderFilterInput, sort *model.OrderSortInput, limit, page *int32) ([]*model.Order, error)
	GetOrderDetail(orderID uint) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
	UpdateStatusByReferenceID(referenceID string, status string) error
	GetByReferenceID(referenceID string) (*Order, error)
	GetOrderBySessionID(
		ctx context.Context,
		sessionID uuid.UUID,
	) (*Order, error)

	CreateOrderTx(
		ctx context.Context,
		order *Order,
		session *CheckoutSession,
	) error

	GetVariantForCheckout(
		ctx context.Context,
		variantID string,
	) (*product.Variant, *product.Product, error)

	CreateCheckoutSession(
		ctx context.Context,
		session *CheckoutSession,
		items []CheckoutSessionItem,
	) error

	GetCheckoutSession(
		ctx context.Context,
		sessionID string,
	) (*CheckoutSession, error)

	GetUserAddress(
		ctx context.Context,
		addressID string,
		userID uint,
	) (*address.Address, error)

	UpdateSessionAddressAndPricing(
		ctx context.Context,
		session *CheckoutSession,
	) error

	ConfirmCheckoutSession(
		ctx context.Context,
		session *CheckoutSession,
	) error

	ValidateVariantStock(
		ctx context.Context,
		variantID string,
		qty int,
	) (bool, error)

	MarkSessionExpired(
		ctx context.Context,
		sessionID uuid.UUID,
	) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetOrderBySessionID(
	ctx context.Context,
	sessionID uuid.UUID,
) (*Order, error) {

	query := `
		SELECT id, status, total_price
		FROM orders
		WHERE checkout_session_id = $1
	`

	var o Order
	err := r.db.QueryRowContext(ctx, query, sessionID).
		Scan(&o.ID, &o.Status, &o.Total)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *repository) CreateOrderTx(
	ctx context.Context,
	order *Order,
	session *CheckoutSession,
) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert order
	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (
			id, user_id, checkout_session_id,
			status, total_price, created_at
		) VALUES ($1,$2,$3,$4,$5,$6)
	`,
		order.ID,
		order.UserID,
		session.ID,
		order.Status,
		order.Total,
		order.CreatedAt,
	)
	if err != nil {
		return err
	}

	// 2. Insert order items + deduct stock
	for _, item := range session.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (
				order_id, variant_id, variant_name,
				quantity, price, subtotal
			) VALUES ($1,$2,$3,$4,$5,$6)
		`,
			order.ID,
			item.VariantID,
			item.VariantName,
			item.Quantity,
			item.Price,
			item.Subtotal,
		)
		if err != nil {
			return err
		}

		// Deduct stock
		_, err = tx.ExecContext(ctx, `
			UPDATE variants
			SET stock = stock - $1
			WHERE id = $2 AND stock >= $1
		`, item.Quantity, item.VariantID)
		if err != nil {
			return err
		}
	}

	// 3. Mark session as completed
	_, err = tx.ExecContext(ctx, `
		UPDATE checkout_sessions
		SET status = 'PAID'
		WHERE id = $1
	`, session.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ✅ Create new order from user’s cart
func (r *repository) CreateOrder(userID uint) (*Order, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1️⃣ Get cart items
	rows, err := tx.Query(`
		SELECT c.product_id, c.quantity, p.price
		FROM carts c
		JOIN products p ON p.id = c.product_id
		WHERE c.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	var total int

	for rows.Next() {
		var item OrderItem
		err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		total += item.Quantity * item.Price
	}

	if len(items) == 0 {
		return nil, errors.New("cart is empty")
	}

	// 2️⃣ Create order
	var orderID uint
	err = tx.QueryRow(`
		INSERT INTO orders (user_id, total, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id
	`, userID, total).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	// 3️⃣ Insert order items
	for _, item := range items {
		_, err = tx.Exec(`
			INSERT INTO order_items (order_id, product_id, quantity, price)
			VALUES ($1, $2, $3, $4)
		`, orderID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return nil, err
		}
	}

	// 4️⃣ Clear user cart
	// _, err = tx.Exec("DELETE FROM carts WHERE user_id = $1", userID)
	// if err != nil {
	// 	return nil, err
	// }

	// 5️⃣ Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Return order struct
	return &Order{
		ID:     orderID,
		UserID: &userID,
		Total:  uint(total),
		Status: StatusPending,
		Items:  items,
	}, nil
}

// ✅ Get all orders for a user or admin
func (r *repository) GetOrders(
	ctx context.Context,
	filter *model.OrderFilterInput,
	sort *model.OrderSortInput,
	limit, page *int32,
) ([]*model.Order, error) {

	// ---------- AUTH ----------
	userID, _ := utils.GetUserIDFromContext(ctx)
	role := utils.GetUserRoleFromContext(ctx)
	isAdmin := role == "ADMIN"

	// ---------- PAGINATION ----------
	finalLimit := int32(20)
	finalPage := int32(1)

	if limit != nil && *limit > 0 {
		finalLimit = *limit
	}
	if page != nil && *page > 0 {
		finalPage = *page
	}
	if finalLimit > 100 {
		finalLimit = 100
	}

	offset := (finalPage - 1) * finalLimit

	log := logger.FromCtx(ctx).With(
		zap.String("method", "GetOrders"),
		zap.String("role", role),
		zap.Int32("limit", finalLimit),
		zap.Int32("page", finalPage),
		zap.Int32("offset", offset),
	)

	log.Debug("start get orders")

	// ---------- BASE QUERY ----------
	query := `
		SELECT
			o.id,
			o.total,
			o.status,
			o.created_at,
			o.updated_at
		FROM orders o
		WHERE 1=1
	`

	args := []any{}
	argIndex := 1

	// ---------- ACCESS CONTROL ----------
	if !isAdmin {
		query += fmt.Sprintf(" AND o.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// ---------- FILTERING ----------
	if filter != nil {

		if filter.Search != nil && *filter.Search != "" {
			query += fmt.Sprintf(
				" AND (o.id::text ILIKE $%d OR o.status ILIKE $%d)",
				argIndex, argIndex,
			)
			args = append(args, "%"+*filter.Search+"%")
			argIndex++
		}

		if filter.Status != nil && *filter.Status != "" {
			query += fmt.Sprintf(" AND o.status = $%d", argIndex)
			args = append(args, *filter.Status)
			argIndex++
		}

		if filter.DateFrom != nil {
			query += fmt.Sprintf(" AND o.created_at >= $%d", argIndex)
			args = append(args, *filter.DateFrom)
			argIndex++
		}

		if filter.DateTo != nil {
			query += fmt.Sprintf(" AND o.created_at <= $%d", argIndex)
			args = append(args, *filter.DateTo)
			argIndex++
		}
	}

	// ---------- SORTING ----------
	orderBy := "o.created_at DESC"

	if sort != nil {
		dir := strings.ToUpper(string(sort.Direction))
		if dir != "ASC" && dir != "DESC" {
			dir = "DESC"
		}

		switch sort.Field {
		case model.OrderSortFieldTotal:
			orderBy = "o.total " + dir
		case model.OrderSortFieldCreatedAt:
			orderBy = "o.created_at " + dir
		}
	}

	query += " ORDER BY " + orderBy

	// ---------- PAGINATION ----------
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, finalLimit, offset)

	log.Debug("executing get orders query",
		zap.String("query", query),
		zap.Any("args", args),
	)

	// ---------- EXECUTE ----------
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to query orders", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order

	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID,
			&o.TotalPrice,
			&o.Status,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			log.Error("failed to scan order row", zap.Error(err))
			return nil, err
		}
		orders = append(orders, &o)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Info("get orders success",
		zap.Int("count", len(orders)),
	)

	return orders, nil
}

// ✅ Get detailed order with items
func (r *repository) GetOrderDetail(orderID uint) (*Order, error) {
	var o Order
	err := r.db.QueryRow(`
		SELECT id, user_id, total, status, created_at, updated_at
		FROM orders WHERE id = $1
	`, orderID).Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("order not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
		SELECT oi.id, oi.product_id, oi.quantity, oi.price, p.name
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		WHERE oi.order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price, &item.Product.Name); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, item)
	}

	return &o, nil
}

// ✅ Admin: Update order status
func (r *repository) UpdateOrderStatus(orderID uint, status OrderStatus) error {
	res, err := r.db.Exec(`UPDATE orders SET status = $1 WHERE id = $2`, status, orderID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

func (r *repository) UpdateStatusByReferenceID(referenceID string, status string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	queryOrder := `UPDATE orders SET status = $1 WHERE id = $2`
	res, err := tx.Exec(queryOrder, status, referenceID)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		tx.Rollback()
		return fmt.Errorf("no order found with id: %s", referenceID)
	}

	queryPayment := `UPDATE payments SET status = $1 WHERE order_id = $2`
	if _, err := tx.Exec(queryPayment, status, referenceID); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *repository) GetByReferenceID(referenceID string) (*Order, error) {
	query := `SELECT id, total, status FROM orders WHERE id = ? LIMIT 1`
	row := r.db.QueryRow(query, referenceID)
	var o Order
	err := row.Scan(&o.ID, &o.Total, &o.Status)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *repository) GetVariantForCheckout(
	ctx context.Context,
	variantID string,
) (*product.Variant, *product.Product, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetVariantForCheckout"),
		zap.String("variant_id", variantID),
	)

	log.Debug("fetching variant for checkout")

	query := `
		SELECT
			v.id,
			v.name,
			v.price,
			v.quantity_type,
			v.imageurl,
			p.name
		FROM variants v
		LEFT JOIN products p ON p.id = v.product_id
		WHERE v.id = $1
	`

	var v product.Variant
	var p product.Product

	err := r.db.QueryRowContext(ctx, query, variantID).
		Scan(&v.ID, &v.Name, &v.Price, &v.QuantityType, &v.ImageURL, &p.Name)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("variant not found")
			return nil, nil, err
		}

		log.Error(
			"failed to query variant for checkout",
			zap.Error(err),
		)
		return nil, nil, err
	}

	log.Debug(
		"variant fetched successfully",
		zap.String("variant_name", v.Name),
		zap.Int("price", int(v.Price)),
		zap.String("product_name", p.Name),
	)

	return &v, &p, nil
}
func (r *repository) CreateCheckoutSession(
	ctx context.Context,
	session *CheckoutSession,
	items []CheckoutSessionItem,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "CreateCheckoutSession"),
		zap.String("session_id", session.ID.String()),
		zap.Int("item_count", len(items)),
	)

	log.Debug("starting checkout session transaction")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error(
			"failed to begin transaction",
			zap.Error(err),
		)
		return err
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error(
					"failed to rollback transaction",
					zap.Error(rbErr),
				)
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	// Insert checkout session
	_, err = tx.ExecContext(ctx, `
		INSERT INTO checkout_sessions (
			id, user_id, status, subtotal, tax, shipping_fee,
			discount, total_price, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		session.ID,
		session.UserID,
		session.Status,
		session.Subtotal,
		session.Tax,
		session.ShippingFee,
		session.Discount,
		session.TotalPrice,
		session.ExpiresAt,
	)
	if err != nil {
		log.Error(
			"failed to insert checkout session",
			zap.Error(err),
		)
		return err
	}

	log.Debug("checkout session inserted")

	// Insert session items
	for i, item := range items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO checkout_session_items (
				id, checkout_session_id, variant_id, variant_name, product_name,
				quantity, quantity_type, imageurl, unit_price, subtotal
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`,
			item.ID,
			session.ID,
			item.VariantID,
			item.VariantName,
			item.ProductName,
			item.Quantity,
			item.QuantityType,
			item.ImageURL,
			item.Price,
			item.Subtotal,
		)
		if err != nil {
			log.Error(
				"failed to insert checkout session item",
				zap.Int("item_index", i),
				zap.String("variant_id", item.VariantID),
				zap.Error(err),
			)
			return err
		}
	}

	log.Debug("all checkout session items inserted")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error(
			"failed to commit checkout session transaction",
			zap.Error(err),
		)
		return err
	}

	committed = true
	log.Info("checkout session transaction committed successfully")

	return nil
}

func (r *repository) GetCheckoutSession(
	ctx context.Context,
	sessionID string,
) (*CheckoutSession, error) {

	var s CheckoutSession

	query := `
		SELECT
			id, status, expires_at, created_at,
			user_id, address_id,
			subtotal, tax, shipping_fee, discount, total_price,
			confirmed_at, payment_ref
		FROM checkout_sessions
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, sessionID).
		Scan(
			&s.ID,
			&s.Status,
			&s.ExpiresAt,
			&s.CreatedAt,
			&s.UserID,
			&s.AddressID,
			&s.Subtotal,
			&s.Tax,
			&s.ShippingFee,
			&s.Discount,
			&s.TotalPrice,
			&s.ConfirmedAt,
			&s.PaymentRef,
		)
	if err != nil {
		return nil, err
	}

	// Load items
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, variant_id, variant_name, image_url,
			quantity, quantity_type, price, subtotal
		FROM checkout_session_items
		WHERE checkout_session_id = $1
	`, s.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item CheckoutSessionItem
		err := rows.Scan(
			&item.ID,
			&item.VariantID,
			&item.VariantName,
			&item.ImageURL,
			&item.Quantity,
			&item.QuantityType,
			&item.Price,
			&item.Subtotal,
		)
		if err != nil {
			return nil, err
		}
		s.Items = append(s.Items, item)
	}

	return &s, nil
}

func (r *repository) GetUserAddress(
	ctx context.Context,
	addressID string,
	userID uint,
) (*address.Address, error) {

	query := `
		SELECT id, city
		FROM addresses
		WHERE id = $1 AND user_id = $2
	`

	var a address.Address
	err := r.db.QueryRowContext(ctx, query, addressID, userID).
		Scan(&a.ID, &a.City)

	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *repository) UpdateSessionAddressAndPricing(
	ctx context.Context,
	session *CheckoutSession,
) error {

	query := `
		UPDATE checkout_sessions
		SET
			address_id = $1,
			shipping_fee = $2,
			tax = $3,
			total_price = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		session.AddressID,
		session.ShippingFee,
		session.Tax,
		session.TotalPrice,
		session.ID,
	)

	return err
}

func (r *repository) ValidateVariantStock(
	ctx context.Context,
	variantID string,
	qty int,
) (bool, error) {

	query := `
		SELECT stock >= $1
		FROM variants
		WHERE id = $2
	`

	var ok bool
	err := r.db.QueryRowContext(ctx, query, qty, variantID).
		Scan(&ok)

	return ok, err
}

func (r *repository) ConfirmCheckoutSession(
	ctx context.Context,
	session *CheckoutSession,
) error {

	query := `
		UPDATE checkout_sessions
		SET
			status = $1,
			payment_ref = $2,
			confirmed_at = NOW()
		WHERE id = $3
		  AND status = 'PENDING'
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		session.Status,
		session.PaymentRef,
		session.ID,
	)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("checkout session already confirmed")
	}

	return nil
}

func (r *repository) MarkSessionExpired(
	ctx context.Context,
	sessionID uuid.UUID,
) error {

	_, err := r.db.ExecContext(ctx, `
		UPDATE checkout_sessions
		SET status = 'EXPIRED'
		WHERE id = $1
		  AND status = 'PENDING'
	`, sessionID)

	return err
}
