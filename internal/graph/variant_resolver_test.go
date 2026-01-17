package graph

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

func TestMutationResolver_CreateVariants(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")

		stock := 10
		input := []*model.NewVariant{
			{ProductID: "p1", Name: "Var 1", Price: 100, Stock: int32(stock)},
		}

		expected := []*product.Variant{
			{ID: "v1", ProductID: "p1", Name: "Var 1", Price: 100, Stock: 10},
		}

		mockSvc.On("CreateVariants", ctx, mock.Anything).Return(expected, nil)

		res, err := mr.CreateVariants(ctx, input)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "v1", res[0].ID)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		_, err := mr.CreateVariants(context.Background(), []*model.NewVariant{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		input := []*model.NewVariant{{ProductID: "p1", Name: "Var 1"}}

		mockSvc.On("CreateVariants", ctx, mock.Anything).Return(nil, errors.New("db error"))

		_, err := mr.CreateVariants(ctx, input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_UpdateVariants(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")

		stock := int32(20)
		name := "Var 1 Updated"
		input := []*model.UpdateVariant{{ID: "v1", Name: &name, Stock: &stock}}
		expected := []*product.Variant{{ID: "v1", Name: "Var 1 Updated", Stock: 20}}

		mockSvc.On("UpdateVariants", ctx, mock.Anything).Return(expected, nil)

		res, err := mr.UpdateVariants(ctx, input)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Var 1 Updated", res[0].Name)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		_, err := mr.UpdateVariants(context.Background(), []*model.UpdateVariant{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockProductService)
		resolver := &Resolver{ProductSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "seller")
		input := []*model.UpdateVariant{{ID: "v1"}}
		mockSvc.On("UpdateVariants", ctx, mock.Anything).Return(nil, errors.New("db error"))
		_, err := mr.UpdateVariants(ctx, input)
		assert.Error(t, err)
	})
}
