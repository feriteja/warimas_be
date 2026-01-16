package cart

import (
	"context"
	"errors"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/product"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
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
		limit, page *uint16) ([]*cartRow, int64, error)
	UpdateCartQuantity(ctx context.Context, params UpdateToCartParams) error
	RemoveFromCart(ctx context.Context, variantIDs []string) error
	ClearCart(ctx context.Context) error
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

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "AddToCart"),
		zap.String("variant_id", params.VariantID),
		zap.Uint32("requested_qty", params.Quantity),
	)

	log.Info("add to cart started")

	// 1️⃣ Get user ID
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("unauthorized user")
		return nil, errors.New("unauthorized")
	}
	log = log.With(zap.Uint("user_id", userID))

	// 2️⃣ Get product variant
	variant, err := s.productRepo.GetProductVariantByID(ctx, product.GetVariantOptions{
		VariantID:  params.VariantID,
		OnlyActive: true,
	})
	if err != nil {
		log.Error("failed to get product variant", zap.Error(err))
		return nil, err
	}
	if variant == nil {
		log.Warn("product variant not found or inactive")
		return nil, ErrProductNotFound
	}

	log.Info("product variant found",
		zap.Int32("stock", variant.Stock),
	)

	// 3️⃣ Get existing cart item
	cartItem, err := s.repo.GetCartItemByUserAndVariant(
		ctx,
		userID,
		params.VariantID,
	)
	if err != nil {
		log.Error("failed to get existing cart item", zap.Error(err))
		return nil, err
	}

	// 4️⃣ Calculate final quantity
	finalQty := params.Quantity
	if cartItem != nil {
		finalQty += uint32(cartItem.Quantity)

		log.Info("existing cart item found",
			zap.String("cart_item_id", cartItem.ID),
			zap.Uint32("existing_qty", uint32(cartItem.Quantity)),
			zap.Uint32("final_qty", finalQty),
		)
	} else {
		log.Info("no existing cart item, creating new one",
			zap.Uint32("final_qty", finalQty),
		)
	}

	// 5️⃣ Validate stock
	if uint32(variant.Stock) < finalQty {
		log.Warn("insufficient stock",
			zap.Uint32("available_stock", uint32(variant.Stock)),
			zap.Uint32("requested_qty", finalQty),
		)
		return nil, ErrInsufficientStock
	}

	// 6️⃣ Create or update cart item
	if cartItem == nil {
		cartItem, err = s.repo.CreateCartItem(ctx, CreateCartItemParams{
			UserID:    userID,
			VariantID: params.VariantID,
			Quantity:  params.Quantity,
		})
		if err != nil {
			log.Error("failed to create cart item", zap.Error(err))
			return nil, err
		}

		log.Info("cart item created",
			zap.String("cart_item_id", cartItem.ID),
			zap.Uint32("quantity", params.Quantity),
		)
	} else {
		cartItem, err = s.repo.UpdateCartItemQuantity(
			ctx,
			cartItem.ID,
			finalQty,
		)
		if err != nil {
			log.Error("failed to update cart item quantity", zap.Error(err))
			return nil, err
		}

		log.Info("cart item updated",
			zap.String("cart_item_id", cartItem.ID),
			zap.Uint32("quantity", finalQty),
		)
	}

	log.Info("add to cart completed successfully")

	return cartItem, nil
}

// service/cart_service.go

func (s *service) GetCart(
	ctx context.Context,
	userID uint,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, page *uint16,
) ([]*cartRow, int64, error) {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "GetCart"),
		zap.Uint("user_id", userID),
	)

	log.Debug("get cart started")

	rows, err := s.repo.GetCartRows(ctx, userID, filter, sort, limit, page)
	if err != nil {
		log.Error("failed to get cart rows", zap.Error(err))
		return nil, 0, ErrFailedGetCartRows
	}

	total, err := s.repo.CountCartItems(ctx, userID, filter)
	if err != nil {
		log.Error("failed to count cart items", zap.Error(err))
		return nil, 0, err
	}

	log.Info("get cart success")

	return rows, total, nil
}

// UpdateCartQuantity updates the quantity of a specific product in the user's cart
func (s *service) UpdateCartQuantity(
	ctx context.Context,
	updateParams UpdateToCartParams,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "UpdateCartQuantity"),
		zap.String("variant_id", updateParams.VariantID),
		zap.Uint32("quantity", updateParams.Quantity),
	)

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("missing user id in context")
		return errors.New("user ID is required")
	}

	log = log.With(zap.Uint("user_id", userID))

	if updateParams.VariantID == "" {
		log.Warn("variant id is empty")
		return errors.New("variant ID is required")
	}

	// Quantity <= 0 means remove item from cart
	if updateParams.Quantity <= 0 {
		log.Info("quantity <= 0, removing item from cart")

		err := s.repo.RemoveFromCart(ctx, DeleteFromCartParams{
			UserID:    uint32(userID),
			VariantID: []string{updateParams.VariantID},
		})
		if err != nil {
			log.Error("failed to remove item from cart", zap.Error(err))
			return err
		}

		log.Info("item successfully removed from cart")
		return nil
	}

	log.Info("updating cart quantity")
	updateParams.UserID = uint32(userID)

	err := s.repo.UpdateCartQuantity(ctx, updateParams)
	if err != nil {
		log.Error("failed to update cart quantity", zap.Error(err))
		return err
	}

	log.Info("cart quantity updated successfully")
	return nil
}

// RemoveFromCart deletes a product from the user's cart
func (s *service) RemoveFromCart(
	ctx context.Context,
	variantIDs []string,
) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "RemoveFromCart"),
		zap.Strings("variant_id", variantIDs),
	)

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok || userID == 0 {
		log.Warn("user not authenticated")
		return ErrUserNotAuthenticated
	}

	log = log.With(zap.Uint("user_id", userID))

	if len(variantIDs) == 0 {
		log.Warn("no variant IDs provided")
		return ErrInvalidRemoveCartInput
	}

	if err := s.repo.RemoveFromCart(ctx, DeleteFromCartParams{
		UserID:    uint32(userID),
		VariantID: variantIDs,
	}); err != nil {
		log.Error("failed to remove cart item", zap.Error(err))
		return err
	}

	log.Info("cart item removed successfully")
	return nil
}

func (s *service) ClearCart(ctx context.Context) error {

	log := logger.FromCtx(ctx).With(
		zap.String("layer", "service"),
		zap.String("method", "ClearCart"),
	)

	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok || userID == 0 {
		log.Warn("user not authenticated")
		return ErrUserNotAuthenticated
	}

	log = log.With(zap.Uint("user_id", userID))

	if err := s.repo.ClearCart(ctx, userID); err != nil {
		log.Error("failed to clear cart", zap.Error(err))
		return ErrFailedClearCart
	}

	log.Info("cart cleared successfully")
	return nil
}
