package graph

import (
	"context"
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

// Helper to extract userID from context (depends on your middleware)
func getUserIDFromContext(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value("userID").(uint)
	if !ok || userID == 0 {
		return 0, fmt.Errorf("unauthorized")
	}
	return userID, nil
}

// Add to Cart
func (r *mutationResolver) AddToCart(ctx context.Context, input model.AddToCartInput) (*model.AddToCartResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr("Unauthorized"),
		}, nil
	}

	pid, err := utils.ToUint(input.ProductID)
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr(err.Error()),
		}, nil
	}

	product, err := r.CartSvc.AddToCart(userID, pid, uint(input.Quantity))
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr(err.Error()),
		}, nil
	}

	cartItem := &model.CartItem{
		UserID:    fmt.Sprint(userID),
		ProductID: fmt.Sprint(input.ProductID),
		Quantity:  input.Quantity,
		Product: &model.Product{
			ID:    fmt.Sprint(product.ID),
			Name:  product.Name,
			Price: product.Price,
			Stock: int32(product.Stock),
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	return &model.AddToCartResponse{
		Success:  true,
		Message:  strPtr("Added to cart"),
		CartItem: cartItem,
	}, nil
}

// Update cart quantity
func (r *mutationResolver) UpdateCart(ctx context.Context, input model.UpdateCartInput) (*model.AddToCartResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr("Unauthorized"),
		}, nil
	}

	pid, err := utils.ToUint(input.ProductID)
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr(err.Error()),
		}, nil
	}

	err = r.CartSvc.UpdateCartQuantity(userID, pid, int(input.Quantity))
	if err != nil {
		return &model.AddToCartResponse{
			Success: false,
			Message: strPtr(err.Error()),
		}, nil
	}

	return &model.AddToCartResponse{
		Success: true,
		Message: strPtr("Cart updated"),
	}, nil
}

// Remove item from cart
func (r *mutationResolver) RemoveFromCart(ctx context.Context, productID string) (bool, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	pid := parseUint(productID)
	err = r.CartSvc.RemoveFromCart(userID, pid)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get all items in my cart
func (r *queryResolver) MyCart(ctx context.Context) ([]*model.CartItem, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	items, err := r.CartSvc.GetCart(userID)
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

// --- helpers ---

func strPtr(s string) *string {
	return &s
}

func parseUint(s string) uint {
	var id uint
	fmt.Sscan(s, &id)
	return id
}
