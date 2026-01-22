package product

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"warimas-be/internal/user"
	"warimas-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetProductsByGroup(ctx context.Context, opts ProductQueryOptions) ([]ProductByCategory, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ProductByCategory), args.Error(1)
}

func (m *MockRepository) GetList(ctx context.Context, opts ProductQueryOptions) ([]*Product, *int, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*Product), args.Get(1).(*int), args.Error(2)
}

func (m *MockRepository) Create(ctx context.Context, input NewProductInput, sellerID string) (Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(Product), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, input UpdateProductInput, sellerID string) (Product, error) {
	args := m.Called(ctx, input, sellerID)
	return args.Get(0).(Product), args.Error(1)
}

func (m *MockRepository) BulkCreateVariants(ctx context.Context, input []*NewVariantInput, sellerID string) ([]*Variant, error) {
	args := m.Called(ctx, input, sellerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Variant), args.Error(1)
}

func (m *MockRepository) BulkUpdateVariants(ctx context.Context, input []*UpdateVariantInput, sellerID string) ([]*Variant, error) {
	args := m.Called(ctx, input, sellerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Variant), args.Error(1)
}

func (m *MockRepository) GetProductByID(ctx context.Context, opts GetProductOptions) (*Product, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Product), args.Error(1)
}

func (m *MockRepository) GetProductVariantByID(ctx context.Context, opts GetVariantOptions) (*Variant, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Variant), args.Error(1)
}

// --- Helpers ---

func mockContextWithSeller(sellerID string) context.Context {
	return context.WithValue(context.Background(), utils.SellerIDKey, sellerID)
}

func mockContextWithRole(role string) context.Context {
	return utils.SetUserContext(context.Background(), 1, "test@example.com", role)
}

// --- Tests ---

func TestService_GetProductsByGroup(t *testing.T) {
	ctx := context.Background()
	opts := ProductQueryOptions{}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := []ProductByCategory{{CategoryName: "Cat1"}}
		mockRepo.On("GetProductsByGroup", ctx, opts).Return(expected, nil)

		res, err := svc.GetProductsByGroup(ctx, opts)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetProductsByGroup", ctx, opts).Return(nil, errors.New("db error"))
		_, err := svc.GetProductsByGroup(ctx, opts)
		assert.Error(t, err)
	})
}

func TestService_GetList(t *testing.T) {
	t.Run("Success_Admin", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		ctx := mockContextWithRole(string(user.RoleAdmin))
		opts := ProductQueryOptions{Page: 1, Limit: 10}

		// Expect OnlyActive to be false for Admin
		expectedOpts := opts
		expectedOpts.OnlyActive = false

		zero := 0
		mockRepo.On("GetList", ctx, expectedOpts).Return([]*Product{}, &zero, nil)

		_, err := svc.GetList(ctx, opts)
		assert.NoError(t, err)
	})

	t.Run("Success_User_Defaults", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		ctx := mockContextWithRole("USER")
		opts := ProductQueryOptions{} // Page 0, Limit 0

		// Service sets defaults
		expectedOpts := opts
		expectedOpts.OnlyActive = true
		expectedOpts.Page = 1
		expectedOpts.Limit = 20

		zero := 0
		mockRepo.On("GetList", ctx, expectedOpts).Return([]*Product{}, &zero, nil)

		_, err := svc.GetList(ctx, opts)
		assert.NoError(t, err)
	})

	t.Run("InvalidPriceRange", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		ctx := context.Background()
		min := 100.0
		max := 50.0
		opts := ProductQueryOptions{MinPrice: &min, MaxPrice: &max}

		_, err := svc.GetList(ctx, opts)
		assert.Error(t, err)
		assert.Equal(t, "min_price cannot be greater than max_price", err.Error())
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		ctx := mockContextWithRole("USER")
		opts := ProductQueryOptions{Page: 1, Limit: 10}
		expectedOpts := opts
		expectedOpts.OnlyActive = true

		mockRepo.On("GetList", ctx, expectedOpts).Return(nil, nil, errors.New("db error"))

		_, err := svc.GetList(ctx, opts)
		assert.Error(t, err)
	})

	t.Run("PaginationLogic", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		ctx := context.Background()

		// Test limit capping (limit > 100 becomes 100)
		opts := ProductQueryOptions{Limit: 150, Page: -1}

		mockRepo.On("GetList", ctx, mock.MatchedBy(func(o ProductQueryOptions) bool {
			return o.Limit == 100 && o.Page == 1
		})).Return([]*Product{}, new(int), nil)

		_, err := svc.GetList(ctx, opts)
		assert.NoError(t, err)
	})
}

func TestService_Create(t *testing.T) {
	sellerID := "seller-1"
	ctx := mockContextWithSeller(sellerID)

	input := NewProductInput{Name: "Product 1"}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := Product{ID: "p1"}
		mockRepo.On("Create", ctx, input, sellerID).Return(expected, nil)

		res, err := svc.Create(ctx, input)
		assert.NoError(t, err)
		assert.Equal(t, "p1", res.ID)
	})

	t.Run("EmptyName", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.Create(ctx, NewProductInput{})
		assert.Error(t, err)
		assert.Equal(t, "name cannot be empty", err.Error())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.Create(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

func TestService_Update(t *testing.T) {
	sellerID := "seller-1"
	ctx := mockContextWithSeller(sellerID)

	name := "Updated Name"
	input := UpdateProductInput{ID: "p1", Name: &name}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := Product{ID: "p1", Name: name}
		mockRepo.On("Update", ctx, input, sellerID).Return(expected, nil)

		res, err := svc.Update(ctx, input)
		assert.NoError(t, err)
		assert.Equal(t, name, res.Name)
	})

	t.Run("MissingID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.Update(ctx, UpdateProductInput{Name: &name})
		assert.Error(t, err)
		assert.Equal(t, "product id is required", err.Error())
	})

	t.Run("EmptyName", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		empty := ""
		_, err := svc.Update(ctx, UpdateProductInput{ID: "p1", Name: &empty})
		assert.Error(t, err)
		assert.Equal(t, "name cannot be empty", err.Error())
	})

	t.Run("NoFields", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.Update(ctx, UpdateProductInput{ID: "p1"})
		assert.Error(t, err)
		assert.Equal(t, "no fields to update", err.Error())
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.Update(context.Background(), input)
		assert.Error(t, err)
		assert.Equal(t, "unauthorized", err.Error())
	})
}

func TestService_CreateVariants(t *testing.T) {
	sellerID := "seller-1"
	ctx := mockContextWithSeller(sellerID)

	input := []*NewVariantInput{{Name: "V1"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := []*Variant{{ID: "v1"}}
		mockRepo.On("BulkCreateVariants", ctx, input, sellerID).Return(expected, nil)

		res, err := svc.CreateVariants(ctx, input)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.CreateVariants(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.CreateVariants(context.Background(), input)
		assert.Error(t, err)
	})
}

func TestService_UpdateVariants(t *testing.T) {
	sellerID := "seller-1"
	ctx := mockContextWithSeller(sellerID)

	name := "V1"
	input := []*UpdateVariantInput{{ID: "v1", ProductID: "p1", Name: &name}}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := []*Variant{{ID: "v1"}}
		mockRepo.On("BulkUpdateVariants", ctx, input, sellerID).Return(expected, nil)

		res, err := svc.UpdateVariants(ctx, input)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		// Nil element
		_, err := svc.UpdateVariants(ctx, []*UpdateVariantInput{nil})
		assert.Error(t, err)

		// Missing ID
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ProductID: "p1"}})
		assert.Error(t, err)

		// Missing ProductID
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ID: "v1"}})
		assert.Error(t, err)

		// Empty Name
		empty := ""
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ID: "v1", ProductID: "p1", Name: &empty}})
		assert.Error(t, err)

		// Negative Price
		negPrice := -10.0
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ID: "v1", ProductID: "p1", Price: &negPrice}})
		assert.Error(t, err)

		// Negative Stock
		negStock := int32(-1)
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ID: "v1", ProductID: "p1", Stock: &negStock}})
		assert.Error(t, err)

		// No fields
		_, err = svc.UpdateVariants(ctx, []*UpdateVariantInput{{ID: "v1", ProductID: "p1"}})
		assert.Error(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		_, err := svc.UpdateVariants(context.Background(), input)
		assert.Error(t, err)
	})
}

func TestService_GetProductByID(t *testing.T) {
	ctx := mockContextWithRole("USER")
	pID := "p1"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := &Product{ID: pID}
		mockRepo.On("GetProductByID", ctx, GetProductOptions{ProductID: pID, OnlyActive: true}).
			Return(expected, nil)

		res, err := svc.GetProductByID(ctx, pID)
		assert.NoError(t, err)
		assert.Equal(t, pID, res.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetProductByID", ctx, mock.Anything).
			Return(nil, sql.ErrNoRows)

		_, err := svc.GetProductByID(ctx, pID)
		assert.Error(t, err)
		assert.Equal(t, ErrProductNotFound, err)
	})

	t.Run("GenericError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("GetProductByID", ctx, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetProductByID(ctx, pID)
		assert.Error(t, err)
	})
}
