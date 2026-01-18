package cart

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
	"warimas-be/internal/graph/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateCartItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	params := CreateCartItemParams{
		UserID:    1,
		VariantID: "var-1",
		Quantity:  2,
	}

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "variant_id", "quantity", "created_at", "updated_at"}).
			AddRow("cart-1", 1, "var-1", 2, time.Now(), nil)

		mock.ExpectQuery("INSERT INTO carts").
			WithArgs(params.UserID, params.VariantID, params.Quantity).
			WillReturnRows(rows)

		res, err := repo.CreateCartItem(context.Background(), params)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "cart-1", res.ID)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO carts").
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateCartItem(context.Background(), params)
		assert.Error(t, err)
	})
}

func TestRepository_UpdateCartQuantity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	params := UpdateToCartParams{
		UserID:    1,
		VariantID: "var-1",
		Quantity:  5,
	}

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("UPDATE carts SET quantity = \\$1").
			WithArgs(params.Quantity, params.UserID, params.VariantID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateCartQuantity(context.Background(), params)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("UPDATE carts SET quantity").
			WillReturnError(errors.New("db error"))

		err := repo.UpdateCartQuantity(context.Background(), params)
		assert.Error(t, err)
	})
}

func TestRepository_RemoveFromCart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	params := DeleteFromCartParams{
		UserID:    1,
		VariantID: []string{"var-1", "var-2"},
	}

	t.Run("Success", func(t *testing.T) {
		// Expect ANY($2) for array arguments
		mock.ExpectExec("DELETE FROM carts WHERE user_id = \\$1 AND variant_id = ANY").
			WithArgs(params.UserID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 2))

		err := repo.RemoveFromCart(context.Background(), params)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM carts").
			WillReturnError(errors.New("db error"))

		err := repo.RemoveFromCart(context.Background(), params)
		assert.Error(t, err)
	})
}

func TestRepository_GetCartRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)
	limit := uint16(10)
	page := uint16(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"c_id", "c_user_id", "c_quantity", "c_created_at", "c_updated_at",
			"p_id", "p_name", "p_seller_id", "s_name", "p_category_id", "p_subcategory_id", "p_slug", "p_status", "p_imageurl",
			"v_id", "v_name", "v_product_id", "v_quantity_type", "v_price", "v_stock", "v_imageurl",
		}).AddRow(
			"cart-1", 1, 2, time.Now(), nil,
			"prod-1", "Shirt", "sel-1", "Seller A", "cat-1", "sub-1", "shirt", "active", "img.jpg",
			"var-1", "Red", "prod-1", "pcs", 10000, 10, "img.jpg",
		)

		mock.ExpectQuery("SELECT .* FROM carts").
			WithArgs(userID, limit, 0). // limit 10, offset 0
			WillReturnRows(rows)

		items, err := repo.GetCartRows(context.Background(), userID, nil, nil, &limit, &page)

		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "var-1", items[0].VariantID)
	})

	t.Run("WithFilters", func(t *testing.T) {
		search := "shirt"
		inStock := true
		filter := &model.CartFilterInput{
			Search:  &search,
			InStock: &inStock,
		}

		// Expect query to contain filter logic (ILIKE and stock > 0)
		mock.ExpectQuery("SELECT .* FROM carts .* WHERE .* > 0 AND .* ILIKE").
			WithArgs(userID, "%shirt%", limit, 0).
			WillReturnRows(sqlmock.NewRows([]string{}))

		items, err := repo.GetCartRows(context.Background(), userID, filter, nil, &limit, &page)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestRepository_GetCartItemByUserAndVariant(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)
	variantID := "var-1"

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "variant_id", "quantity", "created_at", "updated_at"}).
			AddRow("cart-1", 1, "var-1", 2, time.Now(), nil)

		mock.ExpectQuery("SELECT .* FROM carts").
			WithArgs(userID, variantID).
			WillReturnRows(rows)

		item, err := repo.GetCartItemByUserAndVariant(context.Background(), userID, variantID)
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "cart-1", item.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM carts").
			WithArgs(userID, variantID).
			WillReturnError(sql.ErrNoRows)

		item, err := repo.GetCartItemByUserAndVariant(context.Background(), userID, variantID)
		assert.NoError(t, err)
		assert.Nil(t, item)
	})
}

func TestRepository_UpdateCartItemQuantity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	cartItemID := "cart-1"
	quantity := uint32(5)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "variant_id", "quantity", "created_at", "updated_at"}).
			AddRow("cart-1", 1, "var-1", 5, time.Now(), nil)

		mock.ExpectQuery("UPDATE carts").
			WithArgs(quantity, cartItemID).
			WillReturnRows(rows)

		item, err := repo.UpdateCartItemQuantity(context.Background(), cartItemID, quantity)
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, int32(5), item.Quantity)
	})

	t.Run("InvalidQuantity", func(t *testing.T) {
		_, err := repo.UpdateCartItemQuantity(context.Background(), cartItemID, 0)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidQuantity, err)
	})
}

func TestRepository_ClearCart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM carts").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 5))

		err := repo.ClearCart(context.Background(), userID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM carts").
			WillReturnError(errors.New("db error"))

		err := repo.ClearCart(context.Background(), userID)
		assert.Error(t, err)
	})
}

func TestRepository_CountCartItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM carts").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := repo.CountCartItems(context.Background(), userID, nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT").
			WillReturnError(errors.New("db error"))

		_, err := repo.CountCartItems(context.Background(), userID, nil)
		assert.Error(t, err)
	})
}
