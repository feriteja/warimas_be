package cart

import (
	"time"
	"warimas-be/internal/graph/model"
)

// Changed return type to slice of CartItem
func MapCartItemToGraphQL(cr []*CartRow) []*model.CartItem {
	items := make([]*model.CartItem, 0, len(cr))

	for _, r := range cr {
		var variantImageURL string
		if r.VariantImageURL != nil {
			variantImageURL = *r.VariantImageURL
		}

		status := r.Status

		var updatedAt string
		if r.UpdatedAt != nil {
			updatedAt = r.UpdatedAt.Format(time.RFC3339)
		} else {
			updatedAt = r.CreatedAt.Format(time.RFC3339)
		}

		item := &model.CartItem{
			ID:        r.CartID,
			UserID:    r.UserID,
			Quantity:  r.Quantity,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			UpdatedAt: updatedAt,
			Product: &model.ProductCart{
				ID:            r.ProductID,
				Name:          r.ProductName,
				SellerID:      r.SellerID,
				SellerName:    r.SellerName,
				CategoryID:    r.CategoryID,
				SubcategoryID: r.SubcategoryID,
				Slug:          r.Slug,
				Status:        &status,
				ImageURL:      r.ProductImageURL,
				Variant: &model.Variant{
					ID:           r.VariantID,
					ProductID:    r.VariantProductID,
					Name:         r.VariantName,
					QuantityType: r.QuantityType,
					Price:        r.Price,
					Stock:        int32(r.Stock),
					ImageURL:     variantImageURL,
				},
			},
		}

		items = append(items, item)
	}

	return items
}
