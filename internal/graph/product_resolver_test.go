package graph

import (
	"context"
	"errors"
	"testing"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/product"
	"warimas-be/internal/utils"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- Mocks ---

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) Create(ctx context.Context, input product.NewProductInput) (product.Product, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductService) Update(ctx context.Context, input product.UpdateProductInput) (product.Product, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(product.Product), args.Error(1)
}

func (m *MockProductService) GetList(ctx context.Context, opts product.ProductQueryOptions) (*product.ProductListResult, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.ProductListResult), args.Error(1)
}

func (m *MockProductService) GetProductsByGroup(ctx context.Context, opts product.ProductQueryOptions) ([]product.ProductByCategory, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]product.ProductByCategory), args.Error(1)
}

func (m *MockProductService) GetProductByID(ctx context.Context, id string) (*product.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}

// Stubs for interface satisfaction (if needed by your specific Service interface definition)
func (m *MockProductService) CreateVariants(ctx context.Context, input []*product.NewVariantInput) ([]*product.Variant, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}
func (m *MockProductService) UpdateVariants(ctx context.Context, input []*product.UpdateVariantInput) ([]*product.Variant, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*product.Variant), args.Error(1)
}
func (m *MockProductService) GetPackages(ctx context.Context, filter *product.PackageFilterInput, sort *product.PackageSortInput, limit, page int32) ([]*product.Package, error) {
	return nil, nil
}

// --- Tests ---

func TestMutationResolver_CreateProduct(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		// Setup Auth Context
		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		input := model.NewProduct{Name: "New Product"}
		expected := product.Product{ID: "100", Name: "New Product"}

		mockSvc.On("Create", ctx, MapNewProductInput(input)).Return(expected, nil)

		res, err := mr.CreateProduct(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, "100", res.ID)
		assert.Equal(t, "New Product", res.Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		// Empty context (no user)
		ctx := context.Background()
		input := model.NewProduct{Name: "New Product"}

		_, err := mr.CreateProduct(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		input := model.NewProduct{Name: "New Product"}

		mockSvc.On("Create", ctx, mock.Anything).Return(product.Product{}, errors.New("db error"))

		_, err := mr.CreateProduct(ctx, input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_UpdateProduct(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		name := "Updated Name"
		input := model.UpdateProduct{ID: "100", Name: &name}
		expected := product.Product{ID: "100", Name: "Updated Name"}

		mockSvc.On("Update", ctx, mock.Anything).Return(expected, nil)

		res, err := mr.UpdateProduct(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, "100", res.ID)
		assert.Equal(t, "Updated Name", res.Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		name := "Updated Name"
		input := model.UpdateProduct{ID: "100", Name: &name}

		_, err := mr.UpdateProduct(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		input := model.UpdateProduct{ID: "100"}

		mockSvc.On("Update", ctx, mock.Anything).Return(product.Product{}, errors.New("db error"))
		_, err := mr.UpdateProduct(ctx, input)
		assert.Error(t, err)
	})
}

func TestQueryResolver_ProductList(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Setup Operation Context
		opCtx := &graphql.OperationContext{
			Operation: &ast.OperationDefinition{
				SelectionSet: ast.SelectionSet{},
			},
		}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)

		// Setup Field Context (Required for CollectFieldsCtx)
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
			Field: graphql.CollectedField{
				Field: &ast.Field{Name: "productList", SelectionSet: ast.SelectionSet{}},
			},
		})
		limit := int32(10)
		page := int32(1)

		// Mock Service Response
		expectedItems := []*product.Product{
			{ID: "1", Name: "Product A"},
		}
		totalCount := 1
		mockRes := &product.ProductListResult{
			Items:      expectedItems,
			TotalCount: &totalCount,
		}

		// We expect GetList to be called with specific options
		mockSvc.On("GetList", ctx, mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Limit == 10 && opts.Page == 1
		})).Return(mockRes, nil)

		res, err := qr.ProductList(ctx, nil, nil, &page, &limit)

		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
		assert.Equal(t, "Product A", res.Items[0].Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("WithFilter", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Setup Context
		opCtx := &graphql.OperationContext{Operation: &ast.OperationDefinition{SelectionSet: ast.SelectionSet{}}}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{Name: "productList", SelectionSet: ast.SelectionSet{}}}})

		filter := &model.ProductFilterInput{
			Search: utils.StrPtr("Phone"),
		}

		totalCount := 0
		mockRes := &product.ProductListResult{Items: []*product.Product{}, TotalCount: &totalCount}

		mockSvc.On("GetList", ctx, mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return *opts.Search == "Phone"
		})).Return(mockRes, nil)

		res, err := qr.ProductList(ctx, filter, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("WithSort", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Setup Context
		opCtx := &graphql.OperationContext{Operation: &ast.OperationDefinition{SelectionSet: ast.SelectionSet{}}}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{Name: "productList", SelectionSet: ast.SelectionSet{}}}})

		sortInput := &model.ProductSortInput{
			Field:     model.ProductSortFieldPrice,
			Direction: model.SortDirectionDesc,
		}

		// Expect GetList to be called (we assume mapping logic works, just verifying flow)
		mockSvc.On("GetList", ctx, mock.Anything).Return(&product.ProductListResult{}, nil)

		_, err := qr.ProductList(ctx, nil, sortInput, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("PaginationDefaults", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Setup Context
		opCtx := &graphql.OperationContext{Operation: &ast.OperationDefinition{SelectionSet: ast.SelectionSet{}}}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{Name: "productList", SelectionSet: ast.SelectionSet{}}}})

		// Expect defaults: page=1, limit=20
		mockSvc.On("GetList", ctx, mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Page == 1 && opts.Limit == 20
		})).Return(&product.ProductListResult{}, nil)

		_, err := qr.ProductList(ctx, nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("PaginationCap", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Setup Context
		opCtx := &graphql.OperationContext{Operation: &ast.OperationDefinition{SelectionSet: ast.SelectionSet{}}}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{Name: "productList", SelectionSet: ast.SelectionSet{}}}})

		limit := int32(150)
		// Expect capped limit: 100
		mockSvc.On("GetList", ctx, mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Limit == 100
		})).Return(&product.ProductListResult{}, nil)

		_, err := qr.ProductList(ctx, nil, nil, nil, &limit)
		assert.NoError(t, err)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{Operation: &ast.OperationDefinition{}})
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{Name: "productList"}}})

		mockSvc.On("GetList", ctx, mock.Anything).Return(nil, errors.New("db error"))

		_, err := qr.ProductList(ctx, nil, nil, nil, nil)
		assert.Error(t, err)
	})
}

func TestQueryResolver_ProductDetail(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		prodID := "123"
		expected := &product.Product{ID: "123", Name: "Detail Product"}

		mockSvc.On("GetProductByID", ctx, prodID).Return(expected, nil)

		res, err := qr.ProductDetail(ctx, prodID)

		assert.NoError(t, err)
		assert.Equal(t, "Detail Product", res.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		mockSvc.On("GetProductByID", context.Background(), "999").Return(nil, product.ErrProductNotFound)
		res, err := qr.ProductDetail(context.Background(), "999")
		assert.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetProductByID", context.Background(), "123").Return(nil, errors.New("db error"))
		_, err := qr.ProductDetail(context.Background(), "123")
		assert.Error(t, err)
	})
}

func TestQueryResolver_ProductsHome(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		limit := int32(10)
		page := int32(1)

		// Mock Service Response
		expectedGroups := []product.ProductByCategory{
			{
				CategoryName:  "Electronics",
				TotalProducts: 5,
				Products:      []*product.Product{{ID: "1", Name: "Phone"}},
			},
		}

		// Expect GetProductsByGroup call
		mockSvc.On("GetProductsByGroup", ctx, mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Limit == 10 && opts.Page == 1
		})).Return(expectedGroups, nil)

		res, err := qr.ProductsHome(ctx, nil, nil, &page, &limit)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Electronics", res[0].CategoryName)
		assert.Equal(t, int32(5), res[0].TotalProducts)
	})

	t.Run("PaginationDefaults", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		// Expect defaults: page=1, limit=20
		mockSvc.On("GetProductsByGroup", context.Background(), mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Page == 1 && opts.Limit == 20
		})).Return([]product.ProductByCategory{}, nil)

		_, err := qr.ProductsHome(context.Background(), nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("PaginationCap", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		limit := int32(100)
		// Expect capped limit: 50 (as defined in resolver)
		mockSvc.On("GetProductsByGroup", context.Background(), mock.MatchedBy(func(opts product.ProductQueryOptions) bool {
			return opts.Limit == 50
		})).Return([]product.ProductByCategory{}, nil)

		_, err := qr.ProductsHome(context.Background(), nil, nil, nil, &limit)
		assert.NoError(t, err)
	})

	t.Run("WithSort", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		sortInput := &model.ProductSortInput{
			Field:     model.ProductSortFieldPrice,
			Direction: model.SortDirectionAsc,
		}

		mockSvc.On("GetProductsByGroup", ctx, mock.Anything).Return([]product.ProductByCategory{}, nil)

		_, err := qr.ProductsHome(ctx, nil, sortInput, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetProductsByGroup", context.Background(), mock.Anything).Return(nil, errors.New("db error"))
		_, err := qr.ProductsHome(context.Background(), nil, nil, nil, nil)
		assert.Error(t, err)
	})
}
