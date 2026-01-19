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

// MockRepository is a mock implementation of the Repository interface
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

func (m *MockRepository) ClearCart(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// MockProductRepository is a mock for the product repository
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

func (m *MockProductRepository) BulkCreateVariants(ctx context.Context, inputs []*product.NewVariantInput, productID string) ([]*product.Variant, error) {
	args := m.Called(ctx, inputs, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}

func (m *MockProductRepository) UpdateVariants(ctx context.Context, inputs []*product.UpdateVariantInput) ([]*product.Variant, error) {
	args := m.Called(ctx, inputs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}

func (m *MockProductRepository) BulkUpdateVariants(ctx context.Context, inputs []*product.UpdateVariantInput, productID string) ([]*product.Variant, error) {
	args := m.Called(ctx, inputs, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, opts product.GetProductOptions) (*product.Product, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}

func (m *MockProductRepository) Create(ctx context.Context, input product.NewProductInput, sellerID string) (product.Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductRepository) Update(ctx context.Context, input product.UpdateProductInput, sellerID string) (product.Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductRepository) GetList(ctx context.Context, opts product.ProductQueryOptions) ([]*product.Product, *int, error) {
	args := m.Called(ctx, opts)
	var r0 []*product.Product
	if args.Get(0) != nil {
		r0 = args.Get(0).([]*product.Product)
	}
	var r1 *int
	if args.Get(1) != nil {
		r1 = args.Get(1).(*int)
	}
	return r0, r1, args.Error(2)
}

func (m *MockProductRepository) GetPackages(ctx context.Context, filter *product.PackageFilterInput, sort *product.PackageSortInput, limit, page int32, includeCount bool) ([]*product.Package, error) {
	args := m.Called(ctx, filter, sort, limit, page, includeCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Package), args.Error(1)
}

func (m *MockProductRepository) GetProductsByGroup(ctx context.Context, opts product.ProductQueryOptions) ([]product.ProductByCategory, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]product.ProductByCategory), args.Error(1)
}

func TestService_GetCartCount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		ctx := context.Background()
		userID := uint(1)
		expectedCount := int64(5)

		// Expectation: CountCartItems is called with nil filter
		mockRepo.On("CountCartItems", ctx, userID, (*model.CartFilterInput)(nil)).Return(expectedCount, nil)

		// Act
		count, err := svc.GetCartCount(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		ctx := context.Background()
		userID := uint(1)
		expectedErr := errors.New("db error")

		mockRepo.On("CountCartItems", ctx, userID, (*model.CartFilterInput)(nil)).Return(int64(0), expectedErr)

		// Act
		count, err := svc.GetCartCount(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_AddToCart(t *testing.T) {
	userID := uint(1)
	variantID := "var-1"
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	params := AddToCartParams{
		VariantID: variantID,
		Quantity:  2,
	}

	t.Run("Success - New Item", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProductRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProductRepo)

		mockProductRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(&product.Variant{Stock: 10}, nil).Once()
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, nil).Once()
		mockRepo.On("CreateCartItem", ctx, mock.Anything).Return(&CartItem{ID: "cart-1"}, nil).Once()

		_, err := svc.AddToCart(ctx, params)

		assert.NoError(t, err)
		mockProductRepo.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Update Existing Item", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProductRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProductRepo)

		existingItem := &CartItem{ID: "cart-1", Quantity: 1}

		mockProductRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(&product.Variant{Stock: 10}, nil).Once()
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(existingItem, nil).Once()
		mockRepo.On("UpdateCartItemQuantity", ctx, "cart-1", uint32(3)).Return(&CartItem{ID: "cart-1"}, nil).Once()

		_, err := svc.AddToCart(ctx, params)

		assert.NoError(t, err)
		mockProductRepo.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProductRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProductRepo)

		_, err := svc.AddToCart(context.Background(), params) // Empty context

		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})

	t.Run("Error - Product Not Found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProductRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProductRepo)

		mockProductRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(nil, nil).Once()

		_, err := svc.AddToCart(ctx, params)

		assert.Error(t, err)
		assert.Equal(t, ErrProductNotFound, err)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Insufficient Stock", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockProductRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProductRepo)

		// Mock that the variant exists but has low stock (params requests 2, stock is 1)
		mockProductRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(&product.Variant{Stock: 1}, nil).Once()
		// Mock that there is no existing item in the cart, so final quantity is just params.Quantity
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, variantID).Return(nil, nil).Once()

		_, err := svc.AddToCart(ctx, params)

		assert.Error(t, err)
		assert.Equal(t, ErrInsufficientStock, err)
		mockProductRepo.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetCart(t *testing.T) {
	userID := uint(1)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		expectedRows := []*CartRow{{CartID: "c1"}}

		mockRepo.On("GetCartRows", ctx, userID, (*model.CartFilterInput)(nil), (*model.CartSortInput)(nil), (*uint16)(nil), (*uint16)(nil)).Return(expectedRows, nil).Once()
		mockRepo.On("CountCartItems", ctx, userID, (*model.CartFilterInput)(nil)).Return(int64(1), nil).Once()

		rows, total, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, expectedRows, rows)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - GetCartRows fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		dbErr := errors.New("db error")

		mockRepo.On("GetCartRows", ctx, userID, (*model.CartFilterInput)(nil), (*model.CartSortInput)(nil), (*uint16)(nil), (*uint16)(nil)).Return(nil, dbErr).Once()

		_, _, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrFailedGetCartRows, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - CountCartItems fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		dbErr := errors.New("db error")

		mockRepo.On("GetCartRows", ctx, userID, (*model.CartFilterInput)(nil), (*model.CartSortInput)(nil), (*uint16)(nil), (*uint16)(nil)).Return([]*CartRow{}, nil).Once()
		mockRepo.On("CountCartItems", ctx, userID, (*model.CartFilterInput)(nil)).Return(int64(0), dbErr).Once()

		_, _, err := svc.GetCart(ctx, userID, nil, nil, nil, nil)

		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_UpdateCartQuantity(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success - Update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		params := UpdateToCartParams{VariantID: "v1", Quantity: 5}

		mockRepo.On("UpdateCartQuantity", ctx, mock.MatchedBy(func(p UpdateToCartParams) bool {
			return p.UserID == uint32(userID) && p.VariantID == "v1"
		})).Return(nil).Once()

		err := svc.UpdateCartQuantity(ctx, params)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Remove item if quantity is 0", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		params := UpdateToCartParams{VariantID: "v1", Quantity: 0}

		mockRepo.On("RemoveFromCart", ctx, mock.MatchedBy(func(p DeleteFromCartParams) bool {
			return p.UserID == uint32(userID) && p.VariantID[0] == "v1"
		})).Return(nil).Once()

		err := svc.UpdateCartQuantity(ctx, params)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		svc := &service{}
		err := svc.UpdateCartQuantity(context.Background(), UpdateToCartParams{})
		assert.Error(t, err)
		assert.Equal(t, "user ID is required", err.Error())
	})
}

func TestService_RemoveFromCart(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}
		variantIDs := []string{"v1", "v2"}

		mockRepo.On("RemoveFromCart", ctx, mock.MatchedBy(func(p DeleteFromCartParams) bool {
			return p.UserID == uint32(userID) && len(p.VariantID) == 2
		})).Return(nil).Once()

		err := svc.RemoveFromCart(ctx, variantIDs)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty input", func(t *testing.T) {
		svc := &service{}
		err := svc.RemoveFromCart(ctx, []string{})
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRemoveCartInput, err)
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		svc := &service{}
		err := svc.RemoveFromCart(context.Background(), []string{"v1"})
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotAuthenticated, err)
	})
}

func TestService_ClearCart(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := &service{repo: mockRepo}

		mockRepo.On("ClearCart", ctx, userID).Return(nil).Once()

		err := svc.ClearCart(ctx)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		svc := &service{}
		err := svc.ClearCart(context.Background())
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotAuthenticated, err)
	})
}
