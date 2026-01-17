package category

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetCategories(ctx context.Context, filter *string, limit, page *int32) ([]*Category, int64, error) {
	args := m.Called(ctx, filter, limit, page)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*Category), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) AddCategory(ctx context.Context, name string) (*Category, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Category), args.Error(1)
}

func (m *MockRepository) GetSubcategories(ctx context.Context, categoryID string, filter *string, limit, page *int32) ([]*Subcategory, int64, error) {
	args := m.Called(ctx, categoryID, filter, limit, page)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*Subcategory), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetSubcategoriesByIds(ctx context.Context, categoryID []string) (map[string][]*Subcategory, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]*Subcategory), args.Error(1)
}

func (m *MockRepository) AddSubcategory(ctx context.Context, categoryID string, name string) (*Subcategory, error) {
	args := m.Called(ctx, categoryID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Subcategory), args.Error(1)
}

// --- Tests ---

func TestService_AddCategory(t *testing.T) {
	ctx := context.Background()
	name := "Electronics"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := &Category{ID: "cat-1", Name: name}
		mockRepo.On("AddCategory", ctx, name).Return(expected, nil)

		res, err := svc.AddCategory(ctx, name)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("AddCategory", ctx, name).Return(nil, errors.New("db error"))
		_, err := svc.AddCategory(ctx, name)
		assert.Error(t, err)
	})
}

func TestService_AddSubcategory(t *testing.T) {
	ctx := context.Background()
	catID := "cat-1"
	name := "Laptops"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		expected := &Subcategory{ID: "sub-1", CategoryID: catID, Name: name}
		mockRepo.On("AddSubcategory", ctx, catID, name).Return(expected, nil)

		res, err := svc.AddSubcategory(ctx, catID, name)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		mockRepo.On("AddSubcategory", ctx, catID, name).Return(nil, errors.New("db error"))
		_, err := svc.AddSubcategory(ctx, catID, name)
		assert.Error(t, err)
	})
}

func TestService_GetCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_WithPagination", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		limit := int32(10)
		page := int32(2)
		filter := "test"

		expectedList := []*Category{{ID: "cat-1"}}
		expectedTotal := int64(50)
		expectedSubMap := map[string][]*Subcategory{
			"cat-1": {{ID: "sub-1", Name: "Sub 1"}},
		}

		mockRepo.On("GetCategories", ctx, &filter, &limit, &page).Return(expectedList, expectedTotal, nil)
		mockRepo.On("GetSubcategoriesByIds", ctx, []string{"cat-1"}).Return(expectedSubMap, nil)

		res, total, err := svc.GetCategories(ctx, &filter, &limit, &page)
		assert.NoError(t, err)
		assert.Equal(t, expectedList, res)
		assert.Equal(t, expectedTotal, total)
		assert.Equal(t, expectedSubMap["cat-1"], res[0].Subcategories)
	})

	t.Run("Success_Defaults", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("GetCategories", ctx, (*string)(nil), (*int32)(nil), (*int32)(nil)).Return([]*Category{}, int64(0), nil)

		res, _, err := svc.GetCategories(ctx, nil, nil, nil)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)

		mockRepo.On("GetCategories", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("db error"))
		_, _, err := svc.GetCategories(ctx, nil, nil, nil)
		assert.Error(t, err)
	})
}

func TestService_GetSubcategories(t *testing.T) {
	ctx := context.Background()
	catID := "cat-1"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo)
		limit := int32(5)
		page := int32(1)

		expectedList := []*Subcategory{{ID: "sub-1"}}
		expectedTotal := int64(1)

		mockRepo.On("GetSubcategories", ctx, catID, (*string)(nil), &limit, &page).Return(expectedList, expectedTotal, nil)

		res, total, err := svc.GetSubcategories(ctx, catID, nil, &limit, &page)
		assert.NoError(t, err)
		assert.Equal(t, expectedList, res)
		assert.Equal(t, expectedTotal, total)
	})
}
