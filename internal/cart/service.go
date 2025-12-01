package cart

import (
	"context"
	"errors"
	"warimas-be/internal/graph/model"
)

// Service defines the business logic for carts.
type Service interface {
	AddToCart(ctx context.Context, userID uint, variantId string, quantity uint) (*CartItem, error)
	GetCart(ctx context.Context, userID uint, filter *model.CartFilterInput,
		sort *model.CartSortInput,
		limit, page *uint16) ([]*model.CartItem, error)
	UpdateCartQuantity(userID uint, productID string, quantity int) error
	RemoveFromCart(userID uint, productID string) error
	ClearCart(userID uint) error
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new cart service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// AddToCart adds a product to a user's cart
func (s *service) AddToCart(ctx context.Context, userID uint, variantId string, quantity uint) (*CartItem, error) {
	if userID == 0 {
		return nil, errors.New("user ID is required")
	}
	if variantId == "" {
		return nil, errors.New("product ID is required")
	}
	if quantity == 0 {
		return nil, errors.New("quantity must be greater than 0")
	}

	return s.repo.AddToCart(ctx, userID, variantId, quantity)
}

// GetCart retrieves all cart items for a user
func (s *service) GetCart(ctx context.Context, userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, page *uint16) ([]*model.CartItem, error) {
	if userID == 0 {
		return nil, errors.New("user ID is required")
	}
	return s.repo.GetCart(ctx, userID, filter, sort, limit, page)
}

// UpdateCartQuantity updates the quantity of a specific product in the user's cart
func (s *service) UpdateCartQuantity(userID uint, productID string, quantity int) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}
	if productID == "" {
		return errors.New("product ID is required")
	}

	if quantity <= 0 {
		// If the quantity is 0 or negative, remove the item
		return s.repo.RemoveFromCart(userID, productID)
	}

	return s.repo.UpdateCartQuantity(userID, productID, quantity)
}

// RemoveFromCart deletes a product from the user's cart
func (s *service) RemoveFromCart(userID uint, productID string) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}
	if productID == "" {
		return errors.New("product ID is required")
	}
	return s.repo.RemoveFromCart(userID, productID)
}

// ClearCart removes all items for a given user (optional utility)
func (s *service) ClearCart(userID uint) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}

	if err := s.repo.ClearCart(userID); err != nil {
		return err
	}

	return nil
}
