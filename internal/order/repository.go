package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"warimas-be/internal/address"
	"warimas-be/internal/logger"
	"warimas-be/internal/product"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type Repository interface {
	CreateOrder(userID uint) (*Order, error)
	FetchOrders(
		ctx context.Context,
		filter *OrderFilterInput,
		sort *OrderSortInput,
		limit int32,
		offset int32,
	) ([]*Order, error)
	FetchOrderItems(
		ctx context.Context,
		orderIDs []uint,
	) (map[uint][]*OrderItem, error)
	CountOrders(
		ctx context.Context,
		filter *OrderFilterInput,
	) (int64, error)
	GetOrderDetail(orderID uint) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
	UpdateStatusByReferenceID(ctx context.Context, referenceID, ExternalReference, paymentProviderID, status string) error
	GetByReferenceID(ctx context.Context, referenceID string) (*Order, error)
	GetOrderBySessionID(
		ctx context.Context,
		sessionID uuid.UUID,
	) (*Order, error)

	GetOrderByExternalID(
		ctx context.Context,
		externalID string,
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
		externalID string,
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
		SELECT id, status, total_amount
		FROM orders
		WHERE checkout_session_id = $1
	`

	var o Order
	err := r.db.QueryRowContext(ctx, query, sessionID).
		Scan(&o.ID, &o.Status, &o.TotalAmount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *repository) GetOrderByExternalID(
	ctx context.Context,
	externalID string,
) (*Order, error) {

	log := logger.FromCtx(ctx)

	log.Debug("fetching order by external_id",
		zap.String("external_id", externalID),
	)

	query := `
		SELECT id, user_id, status, total_amount, currency, address_id
		FROM orders
		WHERE external_id = $1
	`

	var o Order
	err := r.db.QueryRowContext(ctx, query, externalID).
		Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalAmount,
			&o.Currency,
			&o.AddressID,
		)

	if err == sql.ErrNoRows {
		log.Info("order not found",
			zap.String("external_id", externalID),
		)
		return nil, nil
	}

	if err != nil {
		log.Error("failed to fetch order by external_id",
			zap.String("external_id", externalID),
			zap.Error(err),
		)
		return nil, err
	}

	log.Info("order fetched successfully",
		zap.String("external_id", externalID),
		zap.Uint("order_id", o.ID),
	)

	return &o, nil
}

func (r *repository) CreateOrderTx(
	ctx context.Context,
	order *Order,
	session *CheckoutSession,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "CreateOrderTx"),
		zap.String("session_id", session.ID.String()),
	)

	log.Info("starting order transaction")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// 1. Insert order (RETURNING id)
	err = tx.QueryRowContext(ctx, `
		INSERT INTO orders (
			user_id,
			checkout_session_id,
			status,
			total_amount,
			currency,
			external_id,
			subtotal,
			tax,
			shipping_fee,
			discount,
			address_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`,
		order.UserID,
		session.ID,
		order.Status,
		order.TotalAmount,
		order.Currency,
		order.ExternalID,
		session.Subtotal,
		session.Tax,
		session.ShippingFee,
		session.Discount,
		session.AddressID,
	).Scan(&order.ID)
	if err != nil {
		log.Error("failed to insert order", zap.Error(err))
		return err
	}

	log.Info("order created",
		zap.Uint("order_id", order.ID),
		zap.Int("items_count", len(session.Items)),
	)

	// 2. Insert order items + deduct stock
	for _, item := range session.Items {

		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (
				order_id,
				quantity,
				unit_price,
				variant_id,
				variant_name,
				product_name,
				subtotal
			) VALUES ($1,$2,$3,$4,$5,$6,$7)
		`,
			order.ID,
			item.Quantity,
			item.Price,
			item.VariantID,
			item.VariantName,
			item.ProductName,
			item.Subtotal,
		)
		if err != nil {
			log.Error("failed to insert order item",
				zap.String("variant_id", item.VariantID),
				zap.Error(err),
			)
			return err
		}

		// Deduct stock (safe)
		res, err := tx.ExecContext(ctx, `
			UPDATE variants
			SET stock = stock - $1
			WHERE id = $2 AND stock >= $1
		`,
			item.Quantity,
			item.VariantID,
		)
		if err != nil {
			log.Error("failed to deduct stock",
				zap.String("variant_id", item.VariantID),
				zap.Error(err),
			)
			return err
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			log.Warn("insufficient stock during order creation",
				zap.String("variant_id", item.VariantID),
				zap.Int("quantity", item.Quantity),
			)
			return errors.New("insufficient stock")
		}
	}

	log.Info("all order items inserted and stock deducted")

	// 4. Commit
	if err := tx.Commit(); err != nil {
		log.Error("failed to commit order transaction", zap.Error(err))
		return err
	}

	log.Info("order transaction committed successfully",
		zap.Uint("order_id", order.ID),
	)

	return nil
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

	var items []*OrderItem
	var total int

	for rows.Next() {
		var item OrderItem
		err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
		total += item.Quantity * int(item.Price)
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
		ID:          orderID,
		UserID:      &userID,
		TotalAmount: uint(total),
		Status:      StatusPendingPayment,
		Items:       items,
	}, nil
}

// ✅ Get detailed order with items
func (r *repository) GetOrderDetail(orderID uint) (*Order, error) {
	var o Order
	err := r.db.QueryRow(`
		SELECT id, user_id, total, status, created_at, updated_at
		FROM orders WHERE id = $1
	`, orderID).Scan(&o.ID, &o.UserID, &o.TotalAmount, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("order not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
	SELECT id, order_id, quantity, unit_price, variant_id, variant_name, product_name, subtotal
		FROM order_items
		WHERE oi.order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.Quantity,
			&item.Price,
			&item.VariantID,
			&item.VariantName,
			&item.ProductName,
			&item.Subtotal); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, &item)
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

func (r *repository) UpdateStatusByReferenceID(
	ctx context.Context,
	referenceID string,
	paymentRequestID string,
	paymentProviderID string,
	status string,
) (err error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "UpdateStatusByReferenceID"),
		zap.String("reference_id", referenceID),
		zap.String("payment_request_id", paymentRequestID),
		zap.String("status", status),
	)

	log.Info("starting update payment/order/session status")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to start transaction", zap.Error(err))
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
			log.Warn("transaction rolled back due to error", zap.Error(err))
		}
	}()

	// --------------------------------------------------
	// 1. Update order (LOCK ROW) & get checkout_session_id
	// --------------------------------------------------
	var sessionID string
	queryOrder := `
		UPDATE orders
		SET status = $1
		WHERE external_id = $2
		RETURNING checkout_session_id
	`

	err = tx.QueryRowContext(ctx, queryOrder, status, referenceID).
		Scan(&sessionID)
	if err != nil {
		log.Error("failed to update order status", zap.Error(err))
		return fmt.Errorf("update orders by external_id: %w", err)
	}

	log.Info("order status updated",
		zap.String("checkout_session_id", sessionID),
	)

	// --------------------------------------------------
	// 2. Update checkout session
	// --------------------------------------------------
	querySession := `
		UPDATE checkout_sessions
		SET status = $1
		WHERE id = $2
	`

	res, err := tx.ExecContext(ctx, querySession, status, sessionID)
	if err != nil {
		log.Error("failed to update checkout session", zap.Error(err))
		return fmt.Errorf("update checkout session: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		err = fmt.Errorf("checkout session not found: %s", sessionID)
		log.Warn("checkout session not found", zap.String("session_id", sessionID))
		return err
	}

	log.Info("checkout session status updated")

	// --------------------------------------------------
	// 3. Update payment
	// --------------------------------------------------
	queryPayment := `
		UPDATE payments
		SET status = $1,
		    provider_payment_id = $2
	`

	args := []any{status, paymentProviderID}

	if status == string(StatusPaid) {
		queryPayment += `, paid_at = now()`
	}

	queryPayment += `
		WHERE external_reference = $3
	`

	args = append(args, paymentRequestID)

	res, err = tx.ExecContext(ctx, queryPayment, args...)
	if err != nil {
		log.Error("failed to update payment status", zap.Error(err))
		return fmt.Errorf("update payment: %w", err)
	}

	rows, _ = res.RowsAffected()
	if rows == 0 {
		log.Warn("payment not found",
			zap.String("external_reference", paymentRequestID),
		)
	}

	log.Info("payment status updated")

	// --------------------------------------------------
	// Commit
	// --------------------------------------------------
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("commit transaction: %w", err)
	}

	log.Info("transaction committed successfully")
	return nil
}

func (r *repository) GetByReferenceID(
	ctx context.Context,
	referenceID string,
) (*Order, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetByReferenceID"),
		zap.String("reference_id", referenceID),
	)

	log.Debug("fetching order by reference id")

	query := `
		SELECT
			id,
			total_amount,
			status
		FROM orders
		WHERE external_id = $1
		LIMIT 1
	`

	var o Order
	err := r.db.QueryRowContext(ctx, query, referenceID).
		Scan(&o.ID, &o.TotalAmount, &o.Status)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("order not found")
			return nil, fmt.Errorf("order not found with reference_id: %s", referenceID)
		}

		log.Error("failed to query order", zap.Error(err))
		return nil, err
	}

	log.Debug("order fetched successfully",
		zap.Uint("order_id", o.ID),
		zap.String("status", string(o.Status)),
	)

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
			v.stock,
			p.name
		FROM variants v
		LEFT JOIN products p ON p.id = v.product_id
		WHERE v.id = $1
	`

	var v product.Variant
	var p product.Product

	err := r.db.QueryRowContext(ctx, query, variantID).
		Scan(&v.ID, &v.Name, &v.Price, &v.QuantityType, &v.ImageURL, &v.Stock, &p.Name)

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
			discount, total_amount, expires_at, external_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, $10)
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
		session.ExternalID,
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
	externalID string,
) (*CheckoutSession, error) {

	var s CheckoutSession

	query := `
		SELECT
			id, status, expires_at, created_at,
			user_id, address_id,
			subtotal, tax, shipping_fee, discount, total_amount, currency, external_id,
			confirmed_at
		FROM checkout_sessions
		WHERE external_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, externalID).
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
			&s.Currency,
			&s.ExternalID,
			&s.ConfirmedAt,
		)
	if err != nil {
		return nil, err
	}

	// Load items
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, variant_id, variant_name, product_name, imageurl,
			quantity, quantity_type, unit_price, subtotal
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
			&item.ProductName,
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

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "GetUserAddress"),
		zap.String("address_id", addressID),
		zap.Uint("user_id", userID),
	)

	const query = `
		SELECT id, city
		FROM addresses
		WHERE id = $1
		  AND user_id = $2
		  AND is_active = true
	`

	var a address.Address
	err := r.db.QueryRowContext(ctx, query, addressID, userID).
		Scan(&a.ID, &a.City)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("address not found or not owned by user")
			return nil, ErrAddressNotFound
		}

		log.Error("failed to query user address", zap.Error(err))
		return nil, ErrAddressNotFound
	}

	log.Debug("user address fetched successfully")

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
			total_amount = $4
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
			confirmed_at = NOW()
		WHERE id = $1
		  AND status = 'PENDING'
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
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

func (r *repository) CountOrders(
	ctx context.Context,
	filter *OrderFilterInput,
) (int64, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "countOrders"),
	)

	var (
		total int64
		args  []any
		where []string
	)

	baseQuery := `
		SELECT COUNT(1)
		FROM orders
	`

	// Default condition
	// where = append(where, "deleted_at IS NULL")

	// -------------------------
	// Dynamic filters
	// -------------------------

	if filter != nil {

		// Search (example: order_id or external_id)
		if filter.Search != nil && *filter.Search != "" {
			args = append(args, "%"+*filter.Search+"%")
			where = append(where,
				fmt.Sprintf("(id::text ILIKE $%d OR external_id ILIKE $%d)", len(args), len(args)),
			)
		}

		// Status
		if filter.Status != nil {
			args = append(args, *filter.Status)
			where = append(where,
				fmt.Sprintf("status = $%d", len(args)),
			)
		}

		// Date From
		if filter.DateFrom != nil {
			args = append(args, *filter.DateFrom)
			where = append(where,
				fmt.Sprintf("created_at >= $%d", len(args)),
			)
		}

		// Date To
		if filter.DateTo != nil {
			args = append(args, *filter.DateTo)
			where = append(where,
				fmt.Sprintf("created_at <= $%d", len(args)),
			)
		}
	}

	// -------------------------
	// Final query
	// -------------------------

	query := baseQuery
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	log.Debug("count orders query built",
		zap.String("query", query),
		zap.Any("args", args),
	)

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		log.Error("failed to count orders",
			zap.Error(err),
		)
		return 0, err
	}

	log.Info("orders counted",
		zap.Int64("total", total),
	)

	return total, nil
}

func (r *repository) FetchOrders(
	ctx context.Context,
	filter *OrderFilterInput,
	sort *OrderSortInput,
	limit int32,
	offset int32,
) ([]*Order, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "repository"),
		zap.String("method", "fetchOrders"),
	)

	var (
		args  []any
		where []string
	)

	baseQuery := `
		SELECT o.id, o.user_id, o.status, o.total_amount, o.created_at
		FROM orders o
	`

	// Default condition
	// where = append(where, "o.deleted_at IS NULL")

	if filter != nil {

		if filter.Search != nil && *filter.Search != "" {
			args = append(args, "%"+*filter.Search+"%")
			where = append(where,
				fmt.Sprintf("(o.id::text ILIKE $%d OR o.external_id ILIKE $%d)", len(args), len(args)),
			)
		}

		if filter.Status != nil {
			args = append(args, *filter.Status)
			where = append(where,
				fmt.Sprintf("o.status = $%d", len(args)),
			)
		}

		if filter.DateFrom != nil {
			args = append(args, *filter.DateFrom)
			where = append(where,
				fmt.Sprintf("o.created_at >= $%d", len(args)),
			)
		}

		if filter.DateTo != nil {
			args = append(args, *filter.DateTo)
			where = append(where,
				fmt.Sprintf("o.created_at <= $%d", len(args)),
			)
		}
	}

	orderBy := "o.created_at DESC"
	if sort != nil {
		switch sort.Field {
		case OrderSortFieldCreatedAt:
			if sort.Direction == SortDirectionAsc {
				orderBy = "o.created_at ASC"
			}
		case OrderSortFieldTotal:
			if sort.Direction == SortDirectionAsc {
				orderBy = "o.total_amount ASC"
			} else {
				orderBy = "o.total_amount DESC"
			}
		}
	}

	query := baseQuery
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	args = append(args, limit, offset)
	query += fmt.Sprintf(
		" ORDER BY %s LIMIT $%d OFFSET $%d",
		orderBy,
		len(args)-1,
		len(args),
	)

	log.Debug("fetch orders query built",
		zap.String("query", query),
		zap.Any("args", args),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to query orders", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalAmount,
			&o.CreatedAt,
		); err != nil {
			log.Error("failed to scan order row", zap.Error(err))
			return nil, err
		}
		orders = append(orders, &o)
	}

	return orders, rows.Err()
}

func (r *repository) FetchOrderItems(
	ctx context.Context,
	orderIDs []uint,
) (map[uint][]*OrderItem, error) {

	if len(orderIDs) == 0 {
		return map[uint][]*OrderItem{}, nil
	}

	query := `
		SELECT id, order_id, quantity, unit_price, variant_id, variant_name, product_name, subtotal
		FROM order_items
		WHERE order_id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(orderIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemsMap := make(map[uint][]*OrderItem)

	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.Quantity,
			&item.Price,
			&item.VariantID,
			&item.VariantName,
			&item.ProductName,
			&item.Subtotal,
		); err != nil {
			return nil, err
		}
		itemsMap[item.OrderID] = append(itemsMap[item.OrderID], &item)
	}

	return itemsMap, rows.Err()
}
