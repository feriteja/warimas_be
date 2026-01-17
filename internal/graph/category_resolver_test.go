package graph

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/category"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

// MockCategoryService matches the Service interface in internal/category/service.go
type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) GetCategories(ctx context.Context, filter *string, limit, offset *int32) ([]*category.Category, int64, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*category.Category), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryService) AddCategory(ctx context.Context, name string) (*category.Category, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, offset *int32) ([]*category.Subcategory, int64, error) {
	args := m.Called(ctx, categoryID, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*category.Subcategory), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryService) AddSubcategory(ctx context.Context, categoryID string, name string) (*category.Subcategory, error) {
	args := m.Called(ctx, categoryID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Subcategory), args.Error(1)
}

// --- Tests ---

func TestMutationResolver_AddCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCategoryService)
		// Initialize the Resolver with the mock service
		resolver := &Resolver{CategorySvc: mockSvc}
		// Create the mutation resolver wrapper
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		name := "New Cat"
		expected := &category.Category{ID: "1", Name: name}

		mockSvc.On("AddCategory", ctx, name).Return(expected, nil)

		res, err := mr.AddCategory(ctx, name)

		assert.NoError(t, err)
		assert.Equal(t, expected.ID, res.ID)
		assert.Equal(t, expected.Name, res.Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockSvc := new(MockCategoryService)
		resolver := &Resolver{CategorySvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		name := "New Cat"

		mockSvc.On("AddCategory", ctx, name).Return(nil, errors.New("svc error"))

		_, err := mr.AddCategory(ctx, name)

		assert.Error(t, err)
	})
}

func TestMutationResolver_AddSubcategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCategoryService)
		resolver := &Resolver{CategorySvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		catID := "cat1"
		name := "Sub Cat"
		expected := &category.Subcategory{ID: "sub1", Name: name, CategoryID: catID}

		mockSvc.On("AddSubcategory", ctx, catID, name).Return(expected, nil)

		res, err := mr.AddSubcategory(ctx, catID, name)

		assert.NoError(t, err)
		assert.Equal(t, expected.ID, res.ID)
		assert.Equal(t, expected.Name, res.Name)
	})
}

func TestQueryResolver_Category(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCategoryService)
		resolver := &Resolver{CategorySvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		expectedList := []*category.Category{{ID: "1", Name: "Cat 1"}}
		expectedTotal := int64(1)

		mockSvc.On("GetCategories", ctx, (*string)(nil), (*int32)(nil), (*int32)(nil)).Return(expectedList, expectedTotal, nil)

		res, err := qr.Category(ctx, nil, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, int32(1), res.PageInfo.TotalItems)
		assert.Len(t, res.Items, 1)
	})
}

func TestQueryResolver_Subcategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockCategoryService)
		resolver := &Resolver{CategorySvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		catID := "cat1"
		expectedList := []*category.Subcategory{{ID: "sub1", Name: "Sub 1", CategoryID: catID}}
		expectedTotal := int64(1)

		mockSvc.On("GetSubcategories", ctx, catID, (*string)(nil), (*int32)(nil), (*int32)(nil)).Return(expectedList, expectedTotal, nil)

		res, err := qr.Subcategory(ctx, nil, catID, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, int32(1), res.PageInfo.TotalItems)
		assert.Len(t, res.Items, 1)
	})
}
