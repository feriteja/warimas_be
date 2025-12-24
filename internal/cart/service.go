package cart

import (
	"context"
	"errors"
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/product"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// Service defines the business logic for carts.
type Service interface {
	AddToCart(ctx context.Context, params AddToCartParams) (*CartItem, error)
	GetCart(ctx context.Context, userID uint,
		filter *model.CartFilterInput,
		sort *model.CartSortInput,
		limit, page *uint16) ([]*model.CartItem, error)
	UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error
	RemoveFromCart(ctx context.Context, param DeleteFromCartParams) error
	ClearCart(userID uint) error
}

// service implements the Service interface
type service struct {
	repo        Repository
	productRepo product.Repository
}

// NewService creates a new cart service
func NewService(repo Repository, productRepo product.Repository) Service {
	return &service{repo: repo, productRepo: productRepo}
}

// AddToCart adds a product to a user's cart
func (s *service) AddToCart(
	ctx context.Context,
	params AddToCartParams,
) (*CartItem, error) {

	// 1️⃣ Get product (only active products allowed)
	variant, err := s.productRepo.GetProductVariantByID(ctx, product.GetVariantOptions{
		VariantID:  params.VariantID,
		OnlyActive: true,
	})
	if err != nil {
		return nil, err
	}
	if variant == nil {
		return nil, ErrProductNotFound
	}

	// 2️⃣ Get existing cart item (if any)
	cartItem, err := s.repo.GetCartItemByUserAndVariant(
		ctx,
		params.UserID,
		params.VariantID,
	)
	if err != nil {
		return nil, err
	}

	// 3️⃣ Calculate final quantity
	finalQty := params.Quantity
	if cartItem != nil {
		finalQty += uint32(cartItem.Quantity)
	}

	// 4️⃣ Validate stock
	if uint32(variant.Stock) < finalQty {
		return nil, ErrInsufficientStock
	}

	// 5️⃣ Create or update cart item
	if cartItem == nil {
		cartItem, err = s.repo.CreateCartItem(ctx, CreateCartItemParams{
			UserID:    params.UserID,
			VariantID: params.VariantID,
			Quantity:  params.Quantity,
		})
	} else {
		cartItem, err = s.repo.UpdateCartItemQuantity(
			ctx,
			cartItem.ID,
			finalQty,
		)
	}

	if err != nil {
		return nil, err
	}

	return cartItem, nil
}

// service/cart_service.go
// service/cart_service.go
func (s *service) GetCart(
	ctx context.Context,
	userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, page *uint16,
) ([]*model.CartItem, error) {

	rows, err := s.repo.GetCartRows(ctx, userID, filter, sort, limit, page)
	if err != nil {
		return nil, err
	}

	items := make([]*model.CartItem, 0, len(rows))

	for _, r := range rows {
		item := &model.CartItem{
			ID:        r.CartID,
			UserID:    r.UserID,
			Quantity:  r.Quantity,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
			Product: &model.ProductCart{
				ID:            r.ProductID,
				Name:          r.ProductName,
				SellerID:      r.SellerID,
				SellerName:    r.SellerName,
				CategoryID:    r.CategoryID,
				SubcategoryID: r.SubcategoryID,
				Slug:          r.Slug,
				ImageURL:      r.ProductImageURL,
				Variant: &model.Variant{
					ID:           r.VariantID,
					ProductID:    r.VariantProductID,
					Name:         r.VariantName,
					QuantityType: r.QuantityType,
					Price:        r.Price,
					Stock:        int32(r.Stock),
					ImageURL:     *r.VariantImageURL,
				},
			},
		}

		items = append(items, item)
	}

	return items, nil
}

// UpdateCartQuantity updates the quantity of a specific product in the user's cart
func (s *service) UpdateCartQuantity(ctx context.Context, updateParams UpdateToCartParams) error {
	if updateParams.UserID == 0 {
		return errors.New("user ID is required")
	}
	if updateParams.VariantID == "" {
		return errors.New("variant ID is required")
	}

	if updateParams.Quantity <= 0 {
		// If the quantity is 0 or negative, remove the item
		return s.repo.RemoveFromCart(ctx, DeleteFromCartParams{
			UserID:    updateParams.UserID,
			VariantID: updateParams.VariantID,
		})
	}

	return s.repo.UpdateCartQuantity(ctx, updateParams)
}

// RemoveFromCart deletes a product from the user's cart
func (s *service) RemoveFromCart(ctx context.Context, param DeleteFromCartParams) error {
	if param.UserID == 0 {
		return errors.New("user ID is required")
	}
	if param.VariantID == "" {
		return errors.New("variant ID is required")
	}
	return s.repo.RemoveFromCart(ctx, param)
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
