package order

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
	CreateOrder(userID uint) (*Order, error)
	GetOrders(ctx context.Context, filter *model.OrderFilterInput, sort *model.OrderSortInput, limit, page *int32) ([]*model.Order, error)
	GetOrderDetail(orderID uint) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
	UpdateStatusByReferenceID(referenceID string, status string) error
	GetByReferenceID(referenceID string) (*Order, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
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
		UserID: userID,
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
