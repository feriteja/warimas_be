package cart

import (
	"errors"
	"warimas-be/internal/product"
)

// Service defines the business logic for carts.
type Service interface {
	AddToCart(userID, productID uint, quantity uint) (*product.Product, error)
	GetCart(userID uint) ([]CartItem, error)
	UpdateCartQuantity(userID, productID uint, quantity int) error
	RemoveFromCart(userID, productID uint) error
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
func (s *service) AddToCart(userID, productID uint, quantity uint) (*product.Product, error) {
	if userID == 0 {
		return nil, errors.New("user ID is required")
	}
	if productID == 0 {
		return nil, errors.New("product ID is required")
	}
	if quantity == 0 {
		return nil, errors.New("quantity must be greater than 0")
	}

	return s.repo.AddToCart(userID, productID, quantity)
}

// GetCart retrieves all cart items for a user
func (s *service) GetCart(userID uint) ([]CartItem, error) {
	if userID == 0 {
		return nil, errors.New("user ID is required")
	}
	return s.repo.GetCart(userID)
}

// UpdateCartQuantity updates the quantity of a specific product in the user's cart
func (s *service) UpdateCartQuantity(userID, productID uint, quantity int) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}
	if productID == 0 {
		return errors.New("product ID is required")
	}

	if quantity <= 0 {
		// If the quantity is 0 or negative, remove the item
		return s.repo.RemoveFromCart(userID, productID)
	}

	return s.repo.UpdateCartQuantity(userID, productID, quantity)
}

// RemoveFromCart deletes a product from the user's cart
func (s *service) RemoveFromCart(userID, productID uint) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}
	if productID == 0 {
		return errors.New("product ID is required")
	}
	return s.repo.RemoveFromCart(userID, productID)
}

// ClearCart removes all items for a given user (optional utility)
func (s *service) ClearCart(userID uint) error {
	if userID == 0 {
		return errors.New("user ID is required")
	}

	items, err := s.repo.GetCart(userID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := s.repo.RemoveFromCart(userID, item.ProductID); err != nil {
			return err
		}
	}

	return nil
}
