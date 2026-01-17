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

type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockCartRepository) RemoveFromCart(ctx context.Context, params DeleteFromCartParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockCartRepository) ClearCart(ctx context.Context, userId uint) error {
	args := m.Called(ctx, userId)
	return args.Error(0)
}

func (m *MockCartRepository) GetCartItemByUserAndVariant(ctx context.Context, userID uint, variantID string) (*CartItem, error) {
	args := m.Called(ctx, userID, variantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockCartRepository) UpdateCartItemQuantity(ctx context.Context, cartItemID string, quantity uint32) (*CartItem, error) {
	args := m.Called(ctx, cartItemID, quantity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockCartRepository) CreateCartItem(ctx context.Context, params CreateCartItemParams) (*CartItem, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CartItem), args.Error(1)
}

func (m *MockCartRepository) GetCartRows(ctx context.Context, userID uint, filter *model.CartFilterInput, sort *model.CartSortInput, limit, page *uint16) ([]*CartRow, error) {
	args := m.Called(ctx, userID, filter, sort, limit, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CartRow), args.Error(1)
}

func (m *MockCartRepository) CountCartItems(ctx context.Context, userID uint, filter *model.CartFilterInput) (int64, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).(int64), args.Error(1)
}

// MockProductRepository is required by NewService
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

// Stub other methods to satisfy interface
func (m *MockProductRepository) GetProductsByGroup(ctx context.Context, opts product.ProductQueryOptions) ([]product.ProductByCategory, error) {
	return nil, nil
}
func (m *MockProductRepository) GetList(ctx context.Context, opts product.ProductQueryOptions) ([]*product.Product, *int, error) {
	return nil, nil, nil
}
func (m *MockProductRepository) Create(ctx context.Context, input product.NewProductInput, sellerID string) (product.Product, error) {
	return product.Product{}, nil
}
func (m *MockProductRepository) Update(ctx context.Context, input product.UpdateProductInput, sellerID string) (product.Product, error) {
	return product.Product{}, nil
}
func (m *MockProductRepository) BulkCreateVariants(ctx context.Context, input []*product.NewVariantInput, sellerID string) ([]*product.Variant, error) {
	return nil, nil
}
func (m *MockProductRepository) BulkUpdateVariants(ctx context.Context, input []*product.UpdateVariantInput, sellerID string) ([]*product.Variant, error) {
	return nil, nil
}
func (m *MockProductRepository) GetPackages(ctx context.Context, filter *product.PackageFilterInput, sort *product.PackageSortInput, limit, page int32, includeDisabled bool) ([]*product.Package, error) {
	return nil, nil
}
func (m *MockProductRepository) GetProductByID(ctx context.Context, productParams product.GetProductOptions) (*product.Product, error) {
	args := m.Called(ctx, productParams)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}

// --- Tests ---

func TestService_AddToCart(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	input := AddToCartParams{
		VariantID: "var-123",
		Quantity:  2,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		// 1. Mock Product Check
		mockProdRepo.On("GetProductVariantByID", ctx, mock.Anything).Return(&product.Variant{
			ID: "var-123", Stock: 10,
		}, nil)

		// 2. Mock Check Existing Cart
		mockRepo.On("GetCartItemByUserAndVariant", ctx, userID, input.VariantID).Return(nil, nil)

		// 3. Mock Create
		expectedItem := &CartItem{ID: "cart-1", UserID: 1, Quantity: 2}
		mockRepo.On("CreateCartItem", ctx, mock.MatchedBy(func(p CreateCartItemParams) bool {
			return p.UserID == userID && p.VariantID == input.VariantID && p.Quantity == input.Quantity
		})).Return(expectedItem, nil)

		res, err := svc.AddToCart(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "cart-1", res.ID)
		mockRepo.AssertExpectations(t)
		mockProdRepo.AssertExpectations(t)
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		_, err := svc.AddToCart(context.Background(), input)
		assert.Error(t, err)
	})
}

func TestService_UpdateCartQuantity(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	input := UpdateToCartParams{
		VariantID: "var-123",
		Quantity:  5,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		mockRepo.On("UpdateCartQuantity", ctx, mock.MatchedBy(func(p UpdateToCartParams) bool {
			return p.UserID == uint32(userID) && p.VariantID == input.VariantID && p.Quantity == input.Quantity
		})).Return(nil)

		err := svc.UpdateCartQuantity(ctx, input)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		mockRepo.On("UpdateCartQuantity", ctx, mock.Anything).Return(errors.New("db error"))

		err := svc.UpdateCartQuantity(ctx, input)
		assert.Error(t, err)
	})
}

func TestService_RemoveFromCart(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	variantIDs := []string{"var-1", "var-2"}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		mockRepo.On("RemoveFromCart", ctx, mock.MatchedBy(func(p DeleteFromCartParams) bool {
			return p.UserID == uint32(userID) && len(p.VariantID) == 2
		})).Return(nil)

		err := svc.RemoveFromCart(ctx, variantIDs)
		assert.NoError(t, err)
	})
}

func TestService_GetCart(t *testing.T) {
	userID := uint(1)
	ctx := context.Background()

	limit := uint16(10)
	page := uint16(1)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockCartRepository)
		mockProdRepo := new(MockProductRepository)
		svc := NewService(mockRepo, mockProdRepo)

		expectedRows := []*CartRow{
			{CartID: "cart-1", UserID: 1, VariantID: "var-1", Quantity: 1},
		}
		expectedTotal := int64(1)

		mockRepo.On("GetCartRows", ctx, userID, (*model.CartFilterInput)(nil), (*model.CartSortInput)(nil), &limit, &page).
			Return(expectedRows, nil)

		mockRepo.On("CountCartItems", ctx, userID, (*model.CartFilterInput)(nil)).
			Return(expectedTotal, nil)

		items, total, err := svc.GetCart(ctx, userID, nil, nil, &limit, &page)

		assert.NoError(t, err)
		assert.Equal(t, expectedTotal, total)
		assert.Len(t, items, 1)
	})
}
