package order

import (
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
)

func MapOrderItemToGraphQL(i *OrderItem) *model.OrderItem {
	return &model.OrderItem{
		ID:           int32(i.ID),
		Quantity:     int32(i.Quantity),
		QuantityType: i.QuantityType,
		Pricing: &model.OrderItemPricing{
			Price:    int32(i.Price),
			Subtotal: int32(i.Subtotal),
		},
		Variant: &model.VariantRef{
			ID:          i.VariantID,
			Name:        i.VariantName,
			ProductName: i.ProductName,
			ImageURL:    i.ImageURL,
		},
	}
}

func ToGraphQLOrder(o *Order, addr *address.Address) *model.Order {
	if o == nil {
		return nil
	}

	items := make([]*model.OrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, MapOrderItemToGraphQL(item))
	}

	var shipping *model.OrderShipping
	if addr != nil {
		shipping = &model.OrderShipping{Address: &model.Address{
			ID:           addr.ID.String(),
			Name:         addr.Name,
			ReceiverName: addr.ReceiverName,
			Phone:        addr.Phone,
			AddressLine1: addr.Address1,
			AddressLine2: addr.Address2,
			City:         addr.City,
			Province:     addr.Province,
			Country:      addr.Country,
			PostalCode:   addr.Postal,
		}}
	}

	return &model.Order{
		ID:         int32(o.ID),
		ExternalID: o.ExternalID,
		User:       &model.UserRef{ID: *o.UserID},
		Timestamps: &model.OrderTimestamps{
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		},
		Shipping:      shipping,
		InvoiceNumber: o.InvoiceNumber,
		Pricing: &model.OrderPricing{
			Currency:    o.Currency,
			Subtotal:    int32(o.Subtotal),
			Tax:         int32(o.Tax),
			Discount:    int32(o.Discount),
			ShippingFee: int32(o.ShippingFee),
			Total:       int32(o.TotalAmount),
		},
		Status: model.OrderStatus(o.Status),
		Items:  items,
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
			ProductName:  item.ProductName,
		})
	}

	var addressID *string
	if s.AddressID != nil {
		id := s.AddressID.String()
		addressID = &id
	}

	var paymentMethod string
	if s.PaymentMethod != nil {
		method := string(*s.PaymentMethod)
		paymentMethod = method
	}
	return &model.CheckoutSession{
		ID:            s.ID.String(),
		ExternalID:    s.ExternalID,
		Status:        model.CheckoutSessionStatus(s.Status),
		ExpiresAt:     s.ExpiresAt,
		CreatedAt:     s.CreatedAt,
		AddressID:     addressID, //field AddressID *string `json:"addressId,omitempty"`
		Items:         items,
		Subtotal:      int32(s.Subtotal),
		Tax:           int32(s.Tax),
		ShippingFee:   int32(s.ShippingFee),
		Discount:      int32(s.Discount),
		TotalPrice:    int32(s.TotalPrice),
		PaymentMethod: paymentMethod,
	}
}
