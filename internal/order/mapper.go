package order

import (
	"warimas-be/internal/graph/model"
)

func MapOrderItemToGraphQL(i *OrderItem) *model.OrderItem {
	return &model.OrderItem{
		ID:          int32(i.ID),
		Quantity:    int32(i.Quantity),
		Price:       int32(i.Price),
		VariantID:   i.VariantID,
		VariantName: i.VariantName,
		ProductName: i.ProductName,
		Subtotal:    int32(i.Subtotal),
	}
}

func ToGraphQLOrder(o *Order) *model.Order {
	if o == nil {
		return nil
	}

	items := make([]*model.OrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, MapOrderItemToGraphQL(item))
	}

	var userID *int32
	if o.UserID != nil {
		v := int32(*o.UserID)
		userID = &v
	}

	return &model.Order{
		ID:          int32(o.ID),
		TotalPrice:  int32(o.TotalAmount),
		UserID:      userID,
		Currency:    o.Currency,
		Subtotal:    int32(o.Subtotal),
		Tax:         int32(o.Tax),
		ShippingFee: int32(o.ShippingFee),
		Status:      model.OrderStatus(o.Status),
		AddressID:   o.AddressID.String(),
		Discount:    int32(o.Discount),
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
		Items:       items,
	}
}

func MapCheckoutSessionToGraphQL(
	s *CheckoutSession,
) *model.CheckoutSession {

	if s == nil {
		return nil
	}

	items := make([]*model.CheckoutSessionItem, 0, len(s.Items))
	for _, item := range s.Items {
		items = append(items, &model.CheckoutSessionItem{
			ID:           item.ID.String(),
			VariantID:    item.VariantID,
			VariantName:  item.VariantName,
			ImageURL:     item.ImageURL,
			Quantity:     int32(item.Quantity),
			QuantityType: item.QuantityType,
			Price:        int32(item.Price),
			Subtotal:     int32(item.Subtotal),
		})
	}

	return &model.CheckoutSession{
		ID:          s.ID.String(),
		Status:      model.CheckoutSessionStatus(s.Status),
		ExpiresAt:   s.ExpiresAt,
		CreatedAt:   s.CreatedAt,
		Address:     nil, // resolve via field resolver if needed
		Items:       items,
		Subtotal:    int32(s.Subtotal),
		Tax:         int32(s.Tax),
		ShippingFee: int32(s.ShippingFee),
		Discount:    int32(s.Discount),
		TotalPrice:  int32(s.TotalPrice),
	}
}
