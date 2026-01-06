package order

import (
	"fmt"
	"warimas-be/internal/graph/model"
)

func ToGraphQLOrderItem(item *OrderItem) *model.OrderItem {
	return &model.OrderItem{
		ID:       fmt.Sprint(item.ID),
		Product:  &model.Product{ID: fmt.Sprint(item.Product.ID), Name: item.Product.Name},
		Quantity: int32(item.Quantity),
		Price:    int32(item.Price),
	}
}

func ToGraphQLOrder(o *Order) *model.Order {
	if o == nil {
		return nil
	}

	var items []*model.OrderItem
	for _, i := range o.Items {
		items = append(items, ToGraphQLOrderItem(&i))
	}

	return &model.Order{
		ID:         fmt.Sprint(o.ID),
		TotalPrice: int32(o.Total),
		Status:     model.OrderStatus(o.Status),
		CreatedAt:  o.CreatedAt,
		UpdatedAt:  o.UpdatedAt,
		Items:      items,
	}
}
