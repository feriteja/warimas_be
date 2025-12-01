package graph

import (
	"context"
	"fmt"
	"time"

	"warimas-be/internal/graph/model"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

// Ensure your main resolver struct has access to the Cart service.
// Example:
// type Resolver struct {
//     CartService cart.Service
// }

// Add to Cart
func (r *mutationResolver) AddToCart(ctx context.Context, input model.AddToCartInput) (*model.AddToCartResponse, error) {
	log := logger.FromCtx(ctx).With(
		zap.String("product_id", input.ProductID),
		zap.Int("quantity", int(input.Quantity)),
	)

	log.Info("AddToCart resolver called")

	// 1️⃣ Get user ID from context
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("unauthorized access (no user ID in context)")

		return &model.AddToCartResponse{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	log = log.With(zap.Uint("user_id", uint(userID)))
	log.Info("user authenticated")

	// 2️⃣ Call the service
	cartItemRes, err := r.CartSvc.AddToCart(ctx, uint(userID), input.ProductID, uint(input.Quantity))
	if err != nil {
		log.Warn("failed to add item to cart", zap.Error(err))

		return &model.AddToCartResponse{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	log.Info("cart item added successfully",
		zap.Uint("cart_id", cartItemRes.ID),
		zap.String("product_id", cartItemRes.Product.ID),
		zap.Int("final_qty", int(cartItemRes.Quantity)),
	)

	var resultVariant []*model.VariantCart
	for _, item := range cartItemRes.Product.Variants {
		imgURL := ""
		if item.ImageUrl != nil {
			imgURL = *item.ImageUrl
		}
		resultVariant = append(resultVariant, &model.VariantCart{
			ID:            item.ID,
			Name:          item.Name,
			ProductID:     item.ProductID,
			QuantityType:  item.QuantityType,
			Qty:           int32(cartItemRes.Quantity),
			Price:         item.Price,
			Stock:         int32(item.Stock),
			ImageURL:      imgURL,
			SubcategoryID: item.SubcategoryId,
		})
	}

	// 3️⃣ Build GraphQL response
	cartItem := &model.CartItem{
		ID:     fmt.Sprint(cartItemRes.ID),
		UserID: fmt.Sprint(userID),
		Product: &model.ProductCart{
			ID:       fmt.Sprint(cartItemRes.Product.ID),
			Name:     cartItemRes.Product.Name,
			Variants: resultVariant,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	log.Info("AddToCart resolver completed successfully")

	return &model.AddToCartResponse{
		Success:  true,
		Message:  utils.StrPtr("Added to cart"),
		CartItem: cartItem,
	}, nil
}

// Update cart quantity
func (r *mutationResolver) UpdateCart(ctx context.Context, input model.UpdateCartInput) (*model.Response, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	err := r.CartSvc.UpdateCartQuantity(uint(userID), input.ProductID, int(input.Quantity))
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
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return &model.Response{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	err := r.CartSvc.RemoveFromCart(uint(userID), productID)
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
func (r *queryResolver) MyCart(
	ctx context.Context,
	filter *model.CartFilterInput,
	sort *model.CartSortInput,
	limit, page *int32,
) (*model.MyCartResponse, error) {

	log := logger.FromCtx(ctx)
	log.Info("MyCart resolver called",
		zap.Any("filter", filter),
		zap.Any("sort", sort),
		zap.Any("limit", limit),
		zap.Any("offset", page),
	)

	// === USER ID ===
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		log.Warn("Unauthorized access to MyCart")
		return &model.MyCartResponse{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	log.Info("User authenticated for MyCart", zap.Uint("user_id", uint(userID)))

	// === FETCH DATA FROM SERVICE ===
	var limitVal, pageVal uint16
	if limit != nil {
		limitVal = uint16(*limit)
	}
	if page != nil {
		pageVal = uint16(*page)
	}
	items, err := r.CartSvc.GetCart(ctx, uint(userID), filter, sort, &limitVal, &pageVal)
	if err != nil {
		log.Error("Failed to fetch cart items",
			zap.Uint("user_id", uint(userID)),
			zap.Error(err),
		)
		return nil, err
	}

	log.Info("Cart items fetched",
		zap.Int("count", len(items)),
		zap.Uint("user_id", uint(userID)),
	)

	// === GROUPING LOGIC ===
	group := make(map[string]*model.CartItem)

	for _, it := range items {
		log.Debug("Processing cart item",
			zap.String("cart_item_id", it.ID),
			zap.String("product_id", it.Product.ID),
			zap.Int("quantity", int(it.Product.Variants[0].Qty)),
		)

		v := &model.VariantCart{
			ID:            it.Product.Variants[0].ID,
			CartID:        it.ID,
			Name:          it.Product.Variants[0].Name,
			ProductID:     it.Product.Variants[0].ProductID,
			QuantityType:  it.Product.Variants[0].QuantityType,
			Qty:           it.Product.Variants[0].Qty,
			Price:         it.Product.Variants[0].Price,
			Stock:         it.Product.Variants[0].Stock,
			ImageURL:      it.Product.Variants[0].ImageURL,
			SubcategoryID: it.Product.Variants[0].SubcategoryID,
		}

		if existing, found := group[it.Product.ID]; found {
			// Append variant
			existing.Product.Variants = append(existing.Product.Variants, v)

			log.Debug("Product already in group, appending variant",
				zap.String("product_id", it.Product.ID),
				zap.Int("variants_now", len(existing.Product.Variants)),
			)

			continue
		}

		// NEW entry
		group[it.Product.ID] = &model.CartItem{
			ID:     it.ID,
			UserID: it.UserID,
			Product: &model.ProductCart{
				ID:         it.Product.ID,
				Name:       it.Product.Name,
				SellerID:   it.Product.SellerID,
				CategoryID: it.Product.CategoryID,
				Slug:       it.Product.Slug,
				ImageURL:   it.Product.ImageURL,
				Variants:   []*model.VariantCart{v},
			},
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		}

		log.Debug("Created new grouped cart item",
			zap.String("product_id", it.Product.ID),
		)
	}

	// === MAP → SLICE ===
	response := make([]*model.CartItem, 0, len(group))
	for _, item := range group {
		response = append(response, item)
	}

	log.Info("MyCart response built",
		zap.Int("grouped_items", len(response)),
		zap.Uint("user_id", uint(userID)),
	)

	return &model.MyCartResponse{
		Success:  true,
		CartItem: response,
	}, nil
}
