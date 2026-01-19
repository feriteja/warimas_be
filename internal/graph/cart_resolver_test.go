package graph

import (
	"context"
	"errors"
	"testing"
	"time"
	"warimas-be/internal/cart"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockCartService struct {
	mock.Mock
}

func (m *MockCartService) AddToCart(ctx context.Context, params cart.AddToCartParams) (*cart.CartItem, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cart.CartItem), args.Error(1)
}

func (m *MockCartService) UpdateCartQuantity(ctx context.Context, params cart.UpdateToCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockCartService) RemoveFromCart(ctx context.Context, variantIDs []string) error {
	args := m.Called(ctx, variantIDs)
	return args.Error(0)
}

func (m *MockCartService) ClearCart(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCartService) GetCart(ctx context.Context, userID uint, filter *model.CartFilterInput, sort *model.CartSortInput, limit, page *uint16) ([]*cart.CartRow, int64, error) {
	args := m.Called(ctx, userID, filter, sort, limit, page)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*cart.CartRow), args.Get(1).(int64), args.Error(2)
}

func (m *MockCartService) GetCartCount(ctx context.Context, userID uint) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// --- Tests ---

func TestMutationResolver_AddToCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		input := model.AddToCartInput{
			VariantID: "var-1",
			Quantity:  2,
		}

		updateTime := time.Now()

		expectedItem := &cart.CartItem{
			ID:        "cart-1",
			UserID:    1,
			Quantity:  2,
			CreatedAt: time.Now(),
			UpdatedAt: &updateTime,
		}

		mockSvc.On("AddToCart", ctx, mock.MatchedBy(func(p cart.AddToCartParams) bool {
			return p.VariantID == "var-1" && p.Quantity == 2
		})).Return(expectedItem, nil)

		res, err := mr.AddToCart(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "cart-1", res.CartItem.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		// No user in context
		res, err := mr.AddToCart(context.Background(), model.AddToCartInput{})

		// Resolver handles unauthorized by returning success: false, not an error
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Equal(t, "unauthorized", *res.Message)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		input := model.AddToCartInput{VariantID: "v1", Quantity: 1}

		mockSvc.On("AddToCart", ctx, mock.Anything).Return(nil, errors.New("insufficient stock"))

		res, err := mr.AddToCart(ctx, input)
		assert.NoError(t, err) // Resolver returns success: false, not error
		assert.False(t, res.Success)
		assert.Contains(t, *res.Message, "insufficient stock")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		input := model.AddToCartInput{VariantID: "", Quantity: 0}

		res, err := mr.AddToCart(ctx, input)
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Equal(t, "invalid product or quantity", *res.Message)
	})
}

func TestMutationResolver_UpdateCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.UpdateCartInput{
			VariantID: "var-1",
			Quantity:  5,
		}

		mockSvc.On("UpdateCartQuantity", ctx, mock.MatchedBy(func(p cart.UpdateToCartParams) bool {
			return p.VariantID == "var-1" && p.Quantity == 5
		})).Return(nil)

		res, err := mr.UpdateCart(ctx, input)
		assert.NoError(t, err)
		assert.True(t, res.Success)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.UpdateCartInput{
			VariantID: "var-1",
			Quantity:  5,
		}

		mockSvc.On("UpdateCartQuantity", ctx, mock.Anything).Return(errors.New("db error"))

		res, err := mr.UpdateCart(ctx, input)
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Contains(t, *res.Message, "Failed to update cart")
	})
}

func TestMutationResolver_RemoveFromCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		ids := []string{"v1", "v2"}

		mockSvc.On("RemoveFromCart", ctx, ids).Return(nil)

		res, err := mr.RemoveFromCart(ctx, ids)
		assert.NoError(t, err)
		assert.True(t, res.Success)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		res, err := mr.RemoveFromCart(context.Background(), []string{"v1"})
		assert.NoError(t, err)
		assert.False(t, res.Success)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		// Empty slice
		res, err := mr.RemoveFromCart(ctx, []string{})

		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Equal(t, "Variant IDs are required", *res.Message)
	})

	t.Run("ItemNotFound", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		mockSvc.On("RemoveFromCart", ctx, []string{"v1"}).Return(cart.ErrCartItemNotFound)

		res, err := mr.RemoveFromCart(ctx, []string{"v1"})
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Equal(t, "Item not found in cart", *res.Message)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		mockSvc.On("RemoveFromCart", ctx, []string{"v1"}).Return(errors.New("db error"))

		res, err := mr.RemoveFromCart(ctx, []string{"v1"})
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Equal(t, "db error", *res.Message)
	})
}

func TestQueryResolver_MyCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		now := time.Now()
		expectedItems := []*cart.CartRow{
			{CartID: "c1", UserID: 1, VariantID: "v1", Quantity: 1, CreatedAt: now, UpdatedAt: &now},
		}

		// Match any pointer for limit/page since resolver sets defaults
		mockSvc.On("GetCart", ctx, uint(1), (*model.CartFilterInput)(nil), (*model.CartSortInput)(nil), mock.Anything, mock.Anything).
			Return(expectedItems, int64(1), nil)

		res, err := qr.MyCart(ctx, nil, nil, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		_, err := qr.MyCart(context.Background(), nil, nil, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		mockSvc.On("GetCart", ctx, uint(1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, int64(0), errors.New("db error"))

		_, err := qr.MyCart(ctx, nil, nil, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, "failed to fetch cart items", err.Error())
	})

	t.Run("InvalidPagination", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		limit := int32(0)

		_, err := qr.MyCart(ctx, nil, nil, &limit, nil)
		assert.Error(t, err)
		assert.Equal(t, "limit must be greater than 0", err.Error())
	})

	t.Run("LimitTooLarge", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")
		limit := int32(70000) // > 65535 (MaxUint16)

		_, err := qr.MyCart(ctx, nil, nil, &limit, nil)
		assert.Error(t, err)
		assert.Equal(t, "limit too large", err.Error())
	})
}

func TestQueryResolver_MyCartCount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		mockSvc.On("GetCartCount", ctx, uint(1)).Return(int64(5), nil)

		count, err := qr.MyCartCount(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int32(5), count)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		_, err := qr.MyCartCount(context.Background())
		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockCartService)
		resolver := &Resolver{CartSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

		mockSvc.On("GetCartCount", ctx, uint(1)).Return(int64(0), errors.New("db error"))

		_, err := qr.MyCartCount(ctx)
		assert.Error(t, err)
	})
}
