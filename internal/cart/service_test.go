package cart

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/product"
	"warimas-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetCartItemByUserAndVariant(ctx context.Context, userID uint, variantID string) (*CartItem, error) {
	args := m.Called(ctx, userID, variantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockRepository) CreateCartItem(ctx context.Context, params CreateCartItemParams) (*CartItem, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockRepository) UpdateCartItemQuantity(ctx context.Context, id string, quantity uint32) (*CartItem, error) {
	args := m.Called(ctx, id, quantity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockRepository) GetCartRows(ctx context.Context, userID uint, filter *model.CartFilterInput, sort *model.CartSortInput, limit, page *uint16) ([]*CartRow, error) {
	args := m.Called(ctx, userID, filter, sort, limit, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CartRow), args.Error(1)
}

func (m *MockRepository) CountCartItems(ctx context.Context, userID uint, filter *model.CartFilterInput) (int64, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) RemoveFromCart(ctx context.Context, params DeleteFromCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockRepository) UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockRepository) ClearCart(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetProductVariantByID(ctx context.Context, opts product.GetVariantOptions) (*product.Variant, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Variant), args.Error(1)
}

func (m *MockProductRepository) BulkCreateVariants(ctx context.Context, variants []*product.NewVariantInput, sellerID string) ([]*product.Variant, error) {
	args := m.Called(ctx, variants, sellerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}

func (m *MockProductRepository) BulkUpdateVariants(ctx context.Context, input []*product.UpdateVariantInput, sellerID string) ([]*product.Variant, error) {
	args := m.Called(ctx, input, sellerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}

func (m *MockProductRepository) GetProductsByGroup(ctx context.Context, opts product.ProductQueryOptions) ([]product.ProductByCategory, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]product.ProductByCategory), args.Error(1)
}

func (m *MockProductRepository) GetList(ctx context.Context, opts product.ProductQueryOptions) ([]*product.Product, *int, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*product.Product), args.Get(1).(*int), args.Error(2)
}

func (m *MockProductRepository) Create(ctx context.Context, input product.NewProductInput, sellerID string) (product.Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductRepository) Update(ctx context.Context, input product.UpdateProductInput, sellerID string) (product.Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductRepository) GetPackages(ctx context.Context, filter *product.PackageFilterInput, sort *product.PackageSortInput, limit, page int32, includeDisabled bool) ([]*product.Package, error) {
	args := m.Called(ctx, filter, sort, limit, page, includeDisabled)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Package), args.Error(1)
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, opts product.GetProductOptions) (*product.Product, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}

// --- Helpers ---

func mockContextWithUser(userID uint) context.Context {
	return utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
}

// --- Tests ---

func TestService_AddToCart(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	variantID := "var-123"

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		_, err := svc.AddToCart(context.Background(), AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})

	t.Run("ProductVariantNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		mockProdRepo.On("GetProductVariantByID", ctx, product.GetVariantOptions{VariantID: variantID, OnlyActive: true}).Return(nil, nil)

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
		assert.Equal(t, ErrProductNotFound, err)
	})

	t.Run("InsufficientStock_NewItem", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 5}
		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, nil)

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 10})
		assert.Error(t, err)
		assert.Equal(t, ErrInsufficientStock, err)
	})

	t.Run("Success_NewItem", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 10}
		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, nil)

		expectedItem := &CartItem{ID: "cart-1", Quantity: 2}
		mockRepo.On("CreateCartItem", ctx, CreateCartItemParams{UserID: userID, VariantID: variantID, Quantity: 2}).Return(expectedItem, nil)

		res, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 2})
		assert.NoError(t, err)
		assert.Equal(t, expectedItem, res)
	})

	t.Run("Success_UpdateItem", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 10}
		existingItem := &CartItem{ID: "cart-1", Quantity: 3}

		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(existingItem, nil)

		// Existing 3 + New 2 = 5. Stock 10. OK.
		updatedItem := &CartItem{ID: "cart-1", Quantity: 5}
		mockRepo.On("UpdateCartItemQuantity", ctx, "cart-1", uint32(5)).Return(updatedItem, nil)

		res, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 2})
		assert.NoError(t, err)
		assert.Equal(t, updatedItem, res)
	})

	t.Run("InsufficientStock_UpdateItem", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 5}
		existingItem := &CartItem{ID: "cart-1", Quantity: 4}

		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(existingItem, nil)

		// Existing 4 + New 2 = 6 > Stock 5. Error.
		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 2})
		assert.Error(t, err)
		assert.Equal(t, ErrInsufficientStock, err)
	})

	t.Run("UpdateItem_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 10}
		existingItem := &CartItem{ID: "cart-1", Quantity: 1}

		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(existingItem, nil)
		mockRepo.On("UpdateCartItemQuantity", ctx, "cart-1", uint32(2)).Return(nil, errors.New("update error"))

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
	})

	t.Run("GetProductVariant_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(nil, errors.New("db error"))

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
	})

	t.Run("GetCartItem_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 10}
		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, errors.New("db error"))

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
	})

	t.Run("CreateCartItem_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		variant := &product.Variant{Stock: 10}
		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(variant, nil)
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, nil)
		mockRepo.On("CreateCartItem", ctx, mock.Anything).Return(nil, errors.New("create error"))

		_, err := svc.AddToCart(ctx, AddToCartParams{VariantID: variantID, Quantity: 1})
		assert.Error(t, err)
	})
}

func TestService_GetCart(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		rows := []*CartRow{{CartID: "1"}}
		mockRepo.On("GetCartRows", ctx, userID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
		mockRepo.On("CountCartItems", ctx, userID, mock.Anything).Return(int64(1), nil)

		res, count, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, rows, res)
		assert.Equal(t, int64(1), count)
	})

	t.Run("GetRows_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("GetCartRows", ctx, userID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

		_, _, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, ErrFailedGetCartRows, err)
	})

	t.Run("Count_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("GetCartRows", ctx, userID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*CartRow{}, nil)
		mockRepo.On("CountCartItems", ctx, userID, mock.Anything).Return(int64(0), errors.New("count error"))

		_, _, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)
		assert.Error(t, err)
	})
}

func TestService_UpdateCartQuantity(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	variantID := "var-1"

	t.Run("Unauthorized", func(t *testing.T) {
		svc := NewService(nil, nil)
		err := svc.UpdateCartQuantity(context.Background(), UpdateToCartParams{})
		assert.Error(t, err)
		assert.Equal(t, "user ID is required", err.Error())
	})

	t.Run("VariantID_Empty", func(t *testing.T) {
		svc := NewService(nil, nil)
		err := svc.UpdateCartQuantity(ctx, UpdateToCartParams{VariantID: ""})
		assert.Error(t, err)
		assert.Equal(t, "variant ID is required", err.Error())
	})

	t.Run("RemoveItem_ZeroQuantity", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("RemoveFromCart", ctx, DeleteFromCartParams{UserID: uint32(userID), VariantID: []string{variantID}}).Return(nil)

		err := svc.UpdateCartQuantity(ctx, UpdateToCartParams{VariantID: variantID, Quantity: 0})
		assert.NoError(t, err)
	})

	t.Run("RemoveItem_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("RemoveFromCart", ctx, mock.Anything).Return(errors.New("db error"))
		err := svc.UpdateCartQuantity(ctx, UpdateToCartParams{VariantID: variantID, Quantity: 0})
		assert.Error(t, err)
	})

	t.Run("UpdateQuantity_Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("UpdateCartQuantity", ctx, UpdateToCartParams{UserID: uint32(userID), VariantID: variantID, Quantity: 5}).Return(nil)

		err := svc.UpdateCartQuantity(ctx, UpdateToCartParams{VariantID: variantID, Quantity: 5})
		assert.NoError(t, err)
	})

	t.Run("UpdateQuantity_Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("UpdateCartQuantity", ctx, mock.Anything).Return(errors.New("update error"))
		err := svc.UpdateCartQuantity(ctx, UpdateToCartParams{VariantID: variantID, Quantity: 5})
		assert.Error(t, err)
	})
}

func TestService_RemoveFromCart(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)
	variantIDs := []string{"var-1"}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("RemoveFromCart", ctx, DeleteFromCartParams{UserID: uint32(userID), VariantID: variantIDs}).Return(nil)

		err := svc.RemoveFromCart(ctx, variantIDs)
		assert.NoError(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		svc := NewService(nil, nil)
		err := svc.RemoveFromCart(context.Background(), variantIDs)
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotAuthenticated, err)
	})

	t.Run("EmptyIDs", func(t *testing.T) {
		svc := NewService(nil, nil)
		err := svc.RemoveFromCart(ctx, []string{})
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRemoveCartInput, err)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)
		mockRepo.On("RemoveFromCart", ctx, mock.Anything).Return(errors.New("db error"))
		err := svc.RemoveFromCart(ctx, variantIDs)
		assert.Error(t, err)
	})
}

func TestService_ClearCart(t *testing.T) {
	userID := uint(1)
	ctx := mockContextWithUser(userID)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)

		mockRepo.On("ClearCart", ctx, userID).Return(nil)

		err := svc.ClearCart(ctx)
		assert.NoError(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		svc := NewService(nil, nil)
		err := svc.ClearCart(context.Background())
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotAuthenticated, err)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil)
		mockRepo.On("ClearCart", ctx, userID).Return(errors.New("db error"))
		err := svc.ClearCart(ctx)
		assert.Error(t, err)
	})
}
