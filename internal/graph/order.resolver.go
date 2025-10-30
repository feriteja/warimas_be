package graph

import (
	"context"
	"fmt"
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/order"
	"warimas-be/internal/utils"
)

// --- MAPPER HELPERS ---

func toGraphQLOrderItem(item order.OrderItem) *model.OrderItem {
	return &model.OrderItem{
		ID:       fmt.Sprint(item.ID),
		Product:  &model.Product{ID: fmt.Sprint(item.Product.ID), Name: item.Product.Name, Price: item.Product.Price, Stock: int32(item.Product.Stock)},
		Quantity: int32(item.Quantity),
		Price:    item.Price,
	}
}

func toGraphQLOrder(o *order.Order) *model.Order {
	if o == nil {
		return nil
	}

	var items []*model.OrderItem
	for _, i := range o.Items {
		items = append(items, toGraphQLOrderItem(i))
	}

	return &model.Order{
		ID:        fmt.Sprint(o.ID),
		Total:     o.Total,
		Status:    model.OrderStatus(o.Status),
		CreatedAt: o.CreatedAt.Format(time.RFC3339),
		UpdatedAt: o.UpdatedAt.Format(time.RFC3339),
		Items:     items,
	}
}

func toGraphQLOrders(os []order.Order) []*model.Order {
	var list []*model.Order
	for _, o := range os {
		list = append(list, toGraphQLOrder(&o))
	}
	return list
}

// --- MUTATIONS ---

func (r *mutationResolver) CreateOrder(ctx context.Context) (*model.CreateOrderResponse, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return &model.CreateOrderResponse{
			Success: false,
			Message: utils.StrPtr("Unauthorized"),
		}, nil
	}

	userEmail := utils.GetUserEmailFromContext(ctx)

	newOrder, payment, err := r.OrderSvc.CreateOrder(uint(userID), userEmail)
	if err != nil {
		return &model.CreateOrderResponse{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	return &model.CreateOrderResponse{
		Success:     true,
		Message:     utils.StrPtr("Order created successfully"),
		Order:       toGraphQLOrder(newOrder),
		PaymentURL:  payment.InvoiceURL,
		PaymentStat: payment.Status,
	}, nil
}

func (r *mutationResolver) UpdateOrderStatus(ctx context.Context, input model.UpdateOrderStatusInput) (*model.CreateOrderResponse, error) {
	// Admin only — you already handle auth via @auth(role: ADMIN)
	orderID, err := utils.ToUint(input.OrderID)
	if err != nil {
		return &model.CreateOrderResponse{
			Success: false,
			Message: utils.StrPtr("Invalid order ID"),
		}, nil
	}

	status := order.OrderStatus(input.Status.String())

	err = r.OrderSvc.UpdateOrderStatus(orderID, status)
	if err != nil {
		return &model.CreateOrderResponse{
			Success: false,
			Message: utils.StrPtr(err.Error()),
		}, nil
	}

	return &model.CreateOrderResponse{
		Success: true,
		Message: utils.StrPtr(fmt.Sprintf("Order updated to %s", status)),
	}, nil
}

// --- QUERIES ---

func (r *queryResolver) MyOrders(ctx context.Context) ([]*model.Order, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	orders, err := r.OrderSvc.GetOrders(uint(userID), false)
	if err != nil {
		return nil, err
	}

	return toGraphQLOrders(orders), nil
}

func (r *queryResolver) OrderDetail(ctx context.Context, orderID string) (*model.Order, error) {
	userID, ok := utils.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	oid, err := utils.ToUint(orderID)
	if err != nil {
		return nil, err
	}

	order, err := r.OrderSvc.GetOrderDetail(uint(userID), oid, false)
	if err != nil {
		return nil, err
	}

	return toGraphQLOrder(order), nil
}

func (r *queryResolver) AdminOrders(ctx context.Context) ([]*model.Order, error) {
	orders, err := r.OrderSvc.GetOrders(0, true)
	if err != nil {
		return nil, err
	}

	return toGraphQLOrders(orders), nil
}
