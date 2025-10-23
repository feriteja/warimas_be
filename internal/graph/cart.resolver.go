package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"
)

// Ensure your main resolver struct has access to the Cart service.
// Example:
// type Resolver struct {
//     CartService cart.Service
// }

// Add to Cart
func (r *mutationResolver) AddToCart(ctx context.Context, input model.AddToCartInput) (*model.AddToCartResponse, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return &model.AddToCartResponse{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	pid, err := utils.ToUint(input.ProductID)
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	cartItemRes, err := r.CartSvc.AddToCart(uint(userID), pid, uint(input.Quantity))
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	cartItem := &model.CartItem{
		ID:        fmt.Sprint(cartItemRes.ID),
		UserID:    fmt.Sprint(userID),
		ProductID: fmt.Sprint(cartItemRes.ProductID),
		Quantity:  input.Quantity,
		Product: &model.Product{
			ID:    fmt.Sprint(cartItemRes.Product.ID),
			Name:  cartItemRes.Product.Name,
			Price: cartItemRes.Product.Price,
			Stock: int32(cartItemRes.Product.Stock),
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	return &model.AddToCartResponse{
		Success:  true,
		Message:  utils.StrPtr("Added to cart"),
		CartItem: cartItem,
	}, nil
}

// Update cart quantity
func (r *mutationResolver) UpdateCart(ctx context.Context, input model.UpdateCartInput) (*model.Response, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	pid, err := utils.ToUint(input.ProductID)
	if err != nil {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	err = r.CartSvc.UpdateCartQuantity(uint(userID), pid, int(input.Quantity))
	if err != nil {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	return &model.Response{
		Success: true,
		Message: utils.StrPtr("Cart updated"),
	}, nil
}

// Remove item from cart
func (r *mutationResolver) RemoveFromCart(ctx context.Context, productID string) (*model.Response, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	pid := utils.ParseUint(productID)
	err := r.CartSvc.RemoveFromCart(uint(userID), pid)
	if err != nil {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}
	return &model.Response{
		Success: true,
		Message: utils.StrPtr("Cart updated"),
	}, nil
}

// Get all items in my cart
func (r *queryResolver) MyCart(ctx context.Context) ([]*model.CartItem, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthorized: please login first")
	}

	items, err := r.CartSvc.GetCart(uint(userID))
	if err != nil {
		return nil, err
	}

	var result []*model.CartItem
	for _, item := range items {
		result = append(result, &model.CartItem{
			ID:        fmt.Sprint(item.ID),
			UserID:    fmt.Sprint(item.UserID),
			ProductID: fmt.Sprint(item.ProductID),
			Quantity:  int32(item.Quantity),
			Product: &model.Product{
				ID:    fmt.Sprint(item.Product.ID),
				Name:  item.Product.Name,
				Price: item.Product.Price,
				Stock: int32(item.Product.Stock),
			},
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}
