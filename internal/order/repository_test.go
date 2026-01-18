package order

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"warimas-be/internal/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_FetchOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success", func(t *testing.T) {
		limit := int32(10)
		offset := int32(0)

		rows := sqlmock.NewRows([]string{
			"id", "external_id", "invoice_number", "user_id", "currency",
			"subtotal", "tax", "discount", "shipping_fee", "total_amount",
			"status", "address_id", "created_at", "updated_at",
		}).AddRow(
			1, "ext-1", "INV-1", 1, "IDR",
			10000, 1000, 0, 5000, 16000,
			"PENDING", uuid.New(), time.Now(), time.Now(),
		)

		// Regex for the query
		query := `SELECT o.id, o.external_id, .* FROM orders o WHERE o.user_id = \$1 ORDER BY o.created_at DESC LIMIT \$2 OFFSET \$3`

		mock.ExpectQuery(query).
			WithArgs(userID, limit, offset).
			WillReturnRows(rows)

		orders, err := repo.FetchOrders(ctx, nil, nil, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, int32(1), orders[0].ID)
	})

	t.Run("DBError", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM orders").
			WillReturnError(errors.New("db error"))

		_, err := repo.FetchOrders(ctx, nil, nil, 10, 0)
		assert.Error(t, err)
	})
}

func TestRepository_FetchOrders_Filters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	limit := int32(10)
	offset := int32(0)

	// Helper to create full rows for FetchOrders
	newFullRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "external_id", "invoice_number", "user_id", "currency", "subtotal", "tax", "discount", "shipping_fee", "total_amount", "status", "address_id", "created_at", "updated_at"}).
			AddRow(1, "ext-1", "INV-1", userID, "IDR", 10000, 1000, 0, 5000, 16000, "PAID", uuid.New(), time.Now(), time.Now())
	}

	t.Run("SearchAndStatus", func(t *testing.T) {
		search := "INV-123"
		status := OrderStatusPaid
		filter := &OrderFilterInput{
			Search: &search,
			Status: &status,
		}

		// Expect query with WHERE clauses for user_id, search, and status
		// user_id=$1, search=$2, status=$3, limit=$4, offset=$5
		mock.ExpectQuery(`SELECT .* FROM orders o WHERE o.user_id = \$1 AND \(o.id::text ILIKE \$2 OR o.external_id ILIKE \$2\) AND o.status = \$3 ORDER BY o.created_at DESC LIMIT \$4 OFFSET \$5`).
			WithArgs(userID, "%"+search+"%", status, limit, offset).
			WillReturnRows(newFullRows())

		_, err := repo.FetchOrders(ctx, filter, nil, limit, offset)
		assert.NoError(t, err)
	})

	t.Run("SortTotalAsc", func(t *testing.T) {
		sort := &OrderSortInput{
			Field:     OrderSortFieldTotal,
			Direction: SortDirectionAsc,
		}

		mock.ExpectQuery(`SELECT .* FROM orders o WHERE o.user_id = \$1 ORDER BY o.total_amount ASC LIMIT \$2 OFFSET \$3`).
			WithArgs(userID, limit, offset).
			WillReturnRows(newFullRows())

		_, err := repo.FetchOrders(ctx, nil, sort, limit, offset)
		assert.NoError(t, err)
	})

	t.Run("DateRange", func(t *testing.T) {
		now := time.Now()
		filter := &OrderFilterInput{DateFrom: &now, DateTo: &now}

		mock.ExpectQuery(`SELECT .* FROM orders o WHERE o.user_id = \$1 AND o.created_at >= \$2 AND o.created_at <= \$3`).
			WithArgs(userID, now, now, limit, offset).
			WillReturnRows(newFullRows())

		_, err := repo.FetchOrders(ctx, filter, nil, limit, offset)
		assert.NoError(t, err)
	})
}

func TestRepository_GetOrderDetail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	orderID := uint(100)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "total_amount", "status", "created_at", "updated_at",
			"currency", "address_id", "external_id", "subtotal", "tax",
			"shipping_fee", "discount", "invoice_number",
		}).AddRow(
			orderID, 1, 15000, "PAID", time.Now(), time.Now(),
			"IDR", uuid.New(), "ext-123", 10000, 1000, 4000, 0, "INV-123",
		)

		itemRows := sqlmock.NewRows([]string{
			"id", "order_id", "quantity", "unit_price", "variant_id",
			"variant_name", "product_name", "subtotal", "image_url", "quantity_type",
		}).AddRow(
			1, orderID, 1, 10000, "var-1", "Var A", "Prod A", 10000, "http://img", "pcs",
		)

		mock.ExpectQuery(`SELECT .* FROM orders WHERE id = \$1`).
			WithArgs(orderID).
			WillReturnRows(rows)

		mock.ExpectQuery(`SELECT .* FROM order_items WHERE order_id = \$1`).
			WithArgs(orderID).
			WillReturnRows(itemRows)

		order, err := repo.GetOrderDetail(ctx, orderID)
		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, int32(orderID), order.ID)
		assert.Len(t, order.Items, 1)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM orders WHERE id = \$1`).
			WithArgs(orderID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetOrderDetail(ctx, orderID)
		assert.Error(t, err)
		assert.Equal(t, ErrOrderNotFound, err)
	})
}

func TestRepository_GetOrderDetailByExternalID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	extID := "ext-123"
	orderID := int32(100)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "total_amount", "status", "created_at", "updated_at",
			"currency", "address_id", "external_id", "subtotal", "tax",
			"shipping_fee", "discount", "invoice_number",
		}).AddRow(
			orderID, 1, 15000, "PAID", time.Now(), time.Now(),
			"IDR", uuid.New(), extID, 10000, 1000, 4000, 0, "INV-123",
		)

		itemRows := sqlmock.NewRows([]string{
			"id", "order_id", "quantity", "unit_price", "variant_id",
			"variant_name", "product_name", "subtotal", "image_url", "quantity_type",
		}).AddRow(
			1, orderID, 1, 10000, "var-1", "Var A", "Prod A", 10000, "http://img", "pcs",
		)

		mock.ExpectQuery(`SELECT .* FROM orders WHERE external_id = \$1`).
			WithArgs(extID).
			WillReturnRows(rows)

		mock.ExpectQuery(`SELECT .* FROM order_items WHERE order_id = \$1`).
			WithArgs(orderID).
			WillReturnRows(itemRows)

		order, err := repo.GetOrderDetailByExternalID(ctx, extID)
		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, orderID, order.ID)
		assert.Len(t, order.Items, 1)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .* FROM orders WHERE external_id = \$1`).
			WithArgs(extID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetOrderDetailByExternalID(ctx, extID)
		assert.Error(t, err)
		assert.Equal(t, ErrOrderNotFound, err)
	})
}

func TestRepository_CreateCheckoutSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	sessionID := uuid.New()
	userID := int32(1)
	session := &CheckoutSession{
		ID:         sessionID,
		UserID:     &userID,
		Status:     CheckoutSessionStatusPending,
		Subtotal:   10000,
		TotalPrice: 11000,
		ExternalID: "sess-ext",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}
	items := []CheckoutSessionItem{
		{
			ID:          uuid.New(),
			VariantID:   "var-1",
			VariantName: "V1",
			ProductName: "P1",
			Quantity:    1,
			Price:       10000,
			Subtotal:    10000,
		},
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectExec(`INSERT INTO checkout_sessions`).
			WithArgs(
				session.ID, session.UserID, session.Status, session.Subtotal,
				session.Tax, session.ShippingFee, session.Discount,
				session.TotalPrice, session.ExpiresAt, session.ExternalID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO checkout_session_items`).
			WithArgs(
				items[0].ID, session.ID, items[0].VariantID, items[0].VariantName,
				items[0].ProductName, items[0].Quantity, items[0].QuantityType,
				items[0].ImageURL, items[0].Price, items[0].Subtotal,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := repo.CreateCheckoutSession(ctx, session, items)
		assert.NoError(t, err)
	})

	t.Run("RollbackOnError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO checkout_sessions`).
			WillReturnError(errors.New("insert error"))
		mock.ExpectRollback()

		err := repo.CreateCheckoutSession(ctx, session, items)
		assert.Error(t, err)
	})
}

func TestRepository_GetCheckoutSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	extID := "sess-ext-1"

	t.Run("Success", func(t *testing.T) {
		sessionID := uuid.New()
		itemID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "external_id", "status", "expires_at", "created_at",
			"user_id", "address_id", "subtotal", "tax", "shipping_fee", "discount",
			"total_amount", "currency", "confirmed_at",
			"item_id", "variant_id", "variant_name", "product_name",
			"imageurl", "quantity", "quantity_type", "unit_price", "item_subtotal",
		}).AddRow(
			sessionID, extID, "PENDING", time.Now(), time.Now(),
			1, nil, 10000, 0, 0, 0, 10000, "IDR", nil,
			itemID, "var-1", "V1", "P1", "img", 1, "pcs", 10000, 10000,
		)

		mock.ExpectQuery(`SELECT .* FROM checkout_sessions s LEFT JOIN checkout_session_items i`).
			WithArgs(extID).
			WillReturnRows(rows)

		sess, err := repo.GetCheckoutSession(ctx, extID)
		assert.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, sessionID, sess.ID)
		assert.Len(t, sess.Items, 1)
	})
}

func TestRepository_CreateOrderTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	sessionID := uuid.New()
	userID := int32(1)
	addrID := uuid.New()

	session := &CheckoutSession{
		ID:          sessionID,
		UserID:      &userID,
		AddressID:   &addrID,
		Subtotal:    10000,
		Tax:         1000,
		ShippingFee: 5000,
		Discount:    0,
		Items: []CheckoutSessionItem{
			{
				VariantID:   "var-1",
				Quantity:    1,
				Price:       10000,
				VariantName: "V1",
				ProductName: "P1",
				Subtotal:    10000,
				ImageURL:    utils.StrPtr("img"),
			},
		},
	}

	order := &Order{
		UserID:      &userID,
		Status:      OrderStatusPendingPayment,
		TotalAmount: 16000,
		Currency:    "IDR",
		ExternalID:  "ord-ext-1",
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		// 1. Insert Order
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(
				order.UserID, session.ID, order.Status, order.TotalAmount,
				order.Currency, order.ExternalID, session.Subtotal, session.Tax,
				session.ShippingFee, session.Discount, session.AddressID,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))

		// 2. Insert Order Item
		mock.ExpectExec(`INSERT INTO order_items`).
			WithArgs(
				100, session.Items[0].Quantity, session.Items[0].Price,
				session.Items[0].VariantID, session.Items[0].VariantName,
				session.Items[0].ProductName, session.Items[0].Subtotal, session.Items[0].ImageURL,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 3. Deduct Stock
		mock.ExpectExec(`UPDATE variants SET stock = stock - \$1`).
			WithArgs(session.Items[0].Quantity, session.Items[0].VariantID).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

		mock.ExpectCommit()

		err := repo.CreateOrderTx(ctx, order, session)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), order.ID)
	})

	t.Run("InsufficientStock", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO orders`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(`INSERT INTO order_items`).WillReturnResult(sqlmock.NewResult(1, 1))

		// 0 rows affected implies stock condition failed
		mock.ExpectExec(`UPDATE variants SET stock`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectRollback() // Implicitly handled by db.BeginTx defer rollback if panic/error, but here function returns error

		err := repo.CreateOrderTx(ctx, order, session)
		assert.Error(t, err)
		assert.Equal(t, "insufficient stock", err.Error())
	})

	t.Run("InsertOrderError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO orders`).WillReturnError(errors.New("insert order error"))
		mock.ExpectRollback()
		err := repo.CreateOrderTx(ctx, order, session)
		assert.Error(t, err)
	})

	t.Run("InsertItemError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO orders`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(`INSERT INTO order_items`).WillReturnError(errors.New("insert item error"))
		mock.ExpectRollback()
		err := repo.CreateOrderTx(ctx, order, session)
		assert.Error(t, err)
	})
}

func TestRepository_UpdateStatusByReferenceID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	refID := "ord-ext-1"
	payReqID := "pay-req-1"
	provID := "prov-1"
	status := "PAID"
	sessionID := uuid.New().String()

	t.Run("Success_Paid", func(t *testing.T) {
		mock.ExpectBegin()

		// 1. Update Order
		mock.ExpectQuery(`UPDATE orders SET status = \$1 WHERE external_id = \$2 RETURNING checkout_session_id`).
			WithArgs(status, refID).
			WillReturnRows(sqlmock.NewRows([]string{"checkout_session_id"}).AddRow(sessionID))

		// 2. Update Session
		mock.ExpectExec(`UPDATE checkout_sessions SET status = \$1 WHERE id = \$2`).
			WithArgs(status, sessionID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 3. Update Payment (PAID includes paid_at)
		mock.ExpectExec(`UPDATE payments SET status = \$1, provider_payment_id = \$2\s*, paid_at = now\(\) WHERE external_reference = \$3`).
			WithArgs(status, provID, payReqID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		err := repo.UpdateStatusByReferenceID(ctx, refID, payReqID, provID, status)
		assert.NoError(t, err)
	})

	t.Run("UpdateOrderError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`UPDATE orders`).WillReturnError(errors.New("update order error"))
		mock.ExpectRollback()
		err := repo.UpdateStatusByReferenceID(ctx, refID, payReqID, provID, status)
		assert.Error(t, err)
	})

	t.Run("UpdateSessionError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`UPDATE orders`).WillReturnRows(sqlmock.NewRows([]string{"checkout_session_id"}).AddRow(sessionID))
		mock.ExpectExec(`UPDATE checkout_sessions`).WillReturnError(errors.New("update session error"))
		mock.ExpectRollback()
		err := repo.UpdateStatusByReferenceID(ctx, refID, payReqID, provID, status)
		assert.Error(t, err)
	})
}

func TestRepository_GetOrderByExternalID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	extID := "ord-ext-1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "status", "total_amount", "currency", "address_id", "external_id",
		}).AddRow(1, 1, "PENDING", 10000, "IDR", uuid.New(), extID)

		mock.ExpectQuery(`SELECT id, user_id, status, total_amount, currency, address_id, external_id FROM orders WHERE external_id = \$1`).
			WithArgs(extID).
			WillReturnRows(rows)

		o, err := repo.GetOrderByExternalID(ctx, extID)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), o.ID)
	})
}

func TestRepository_GetVariantForCheckout(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	variantID := "var-1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "price", "quantity_type", "imageurl", "stock", "product_name"}).
			AddRow(variantID, "Variant 1", 10000, "pcs", "img", 10, "Product 1")

		mock.ExpectQuery(`SELECT v.id, v.name, v.price, .* FROM variants v`).
			WithArgs(variantID).
			WillReturnRows(rows)

		v, p, err := repo.GetVariantForCheckout(ctx, variantID)
		assert.NoError(t, err)
		assert.Equal(t, variantID, v.ID)
		assert.Equal(t, "Product 1", p.Name)
	})
}

func TestRepository_GetUserAddress(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	addrID := uuid.New().String()
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "city"}).
			AddRow(addrID, "Jakarta")

		mock.ExpectQuery(`SELECT id, city FROM addresses`).
			WithArgs(addrID, userID).
			WillReturnRows(rows)

		addr, err := repo.GetUserAddress(ctx, addrID, userID)
		assert.NoError(t, err)
		assert.Equal(t, "Jakarta", addr.City)
	})
}

func TestRepository_UpdateSessionAddressAndPricing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	addrID := uuid.New()
	session := &CheckoutSession{
		ID:          uuid.New(),
		AddressID:   &addrID,
		ShippingFee: 10000,
		Tax:         1000,
		TotalPrice:  20000,
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE checkout_sessions SET address_id = \$1`).
			WithArgs(session.AddressID, session.ShippingFee, session.Tax, session.TotalPrice, session.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateSessionAddressAndPricing(ctx, session)
		assert.NoError(t, err)
	})
}

func TestRepository_ValidateVariantStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("InStock", func(t *testing.T) {
		mock.ExpectQuery(`SELECT stock >= \$1 FROM variants`).
			WithArgs(5, "var-1").
			WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

		ok, err := repo.ValidateVariantStock(ctx, "var-1", 5)
		assert.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestRepository_ConfirmCheckoutSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	session := &CheckoutSession{ID: uuid.New()}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE checkout_sessions SET confirmed_at = NOW\(\)`).
			WithArgs(session.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.ConfirmCheckoutSession(ctx, session)
		assert.NoError(t, err)
	})

	t.Run("AlreadyConfirmed", func(t *testing.T) {
		mock.ExpectExec(`UPDATE checkout_sessions`).
			WithArgs(session.ID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		err := repo.ConfirmCheckoutSession(ctx, session)
		assert.Error(t, err)
		assert.Equal(t, "checkout session already confirmed", err.Error())
	})
}

func TestRepository_CountOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(1\) FROM orders`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		count, err := repo.CountOrders(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	t.Run("WithFilters", func(t *testing.T) {
		search := "test"
		filter := &OrderFilterInput{Search: &search}

		// Query builder uses dynamic args.
		// Search is the first filter added.
		mock.ExpectQuery(`SELECT COUNT\(1\) FROM orders WHERE \(id::text ILIKE \$1 OR external_id ILIKE \$1\)`).
			WithArgs("%" + search + "%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountOrders(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestRepository_FetchOrderItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	orderIDs := []int32{1, 2}

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "order_id", "variant_name", "product_name", "image_url",
			"quantity", "quantity_type", "unit_price", "variant_id", "subtotal",
		}).AddRow(10, 1, "V1", "P1", "img", 1, "pcs", 1000, "var-1", 1000)

		// pq.Array can be tricky with sqlmock, usually matching the query string is enough
		mock.ExpectQuery(`SELECT .* FROM order_items WHERE order_id = ANY\(\$1\)`).
			WithArgs(pq.Array(orderIDs)).
			WillReturnRows(rows)

		itemsMap, err := repo.FetchOrderItems(ctx, orderIDs)
		assert.NoError(t, err)
		assert.Len(t, itemsMap[1], 1)
	})
}

func TestRepository_UpdateOrderStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	orderID := uint(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE orders SET status = \$1 WHERE id = \$2`).
			WithArgs(OrderStatusPaid, orderID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateOrderStatus(orderID, OrderStatusPaid)
		assert.NoError(t, err)
	})
}

func TestRepository_GetByReferenceID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	refID := "ref-1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "total_amount", "status"}).
			AddRow(1, 10000, "PENDING")

		mock.ExpectQuery(`SELECT id, total_amount, status FROM orders WHERE external_id = \$1`).
			WithArgs(refID).
			WillReturnRows(rows)

		o, err := repo.GetByReferenceID(ctx, refID)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), o.ID)
	})
}

func TestRepository_GetOrderBySessionID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sessID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "status", "total_amount"}).
			AddRow(1, "PENDING", 10000)

		mock.ExpectQuery(`SELECT id, status, total_amount FROM orders WHERE checkout_session_id = \$1`).
			WithArgs(sessID).
			WillReturnRows(rows)

		o, err := repo.GetOrderBySessionID(ctx, sessID)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), o.ID)
	})
}

func TestRepository_MarkSessionExpired(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	sessID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE checkout_sessions SET status = 'EXPIRED'`).
			WithArgs(sessID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.MarkSessionExpired(ctx, sessID)
		assert.NoError(t, err)
	})
}
