package order

import (
	"errors"
	"fmt"
	"warimas-be/internal/payment"
)

type Service interface {
	CreateOrder(userID uint, userEmail string) (*Order, *payment.PaymentResponse, error)
	GetOrders(userID uint, isAdmin bool) ([]Order, error)
	GetOrderDetail(userID, orderID uint, isAdmin bool) (*Order, error)
	UpdateOrderStatus(orderID uint, status OrderStatus) error
}

type service struct {
	repo        Repository
	paymentRepo payment.Repository
	paymentGate payment.Gateway
}

func NewService(repo Repository, payRepo payment.Repository, payGate payment.Gateway) Service {
	return &service{
		repo:        repo,
		paymentRepo: payRepo,
		paymentGate: payGate,
	}
}

// ✅ Create new order from cart
func (s *service) CreateOrder(userID uint, userEmail string) (*Order, *payment.PaymentResponse, error) {
	if userID == 0 {
		return nil, nil, errors.New("unauthorized")
	}

	order, err := s.repo.CreateOrder(userID)
	if err != nil {
		return nil, nil, err
	}

	var items []payment.OrderItem
	for _, oi := range order.Items {
		items = append(items, payment.OrderItem{
			ProductID: oi.ProductID,
			Quantity:  oi.Quantity,
		})
	}
	payResp, err := s.paymentGate.CreateInvoice(order.ID, userEmail, order.Total, userEmail, items, payment.ChannelBCA)
	if err != nil {
		return order, nil, fmt.Errorf("failed to create payment invoice: %w", err)
	}

	p := &payment.Payment{
		OrderID:       order.ID,
		ExternalID:    payResp.ExternalID,
		InvoiceURL:    payResp.InvoiceURL,
		Amount:        payResp.Amount,
		Status:        payResp.Status,
		PaymentMethod: payResp.PaymentMethod,
		ChannelCode:   payResp.ChannelCode,
		PaymentCode:   payResp.PaymentCode,
	}

	err = s.paymentRepo.SavePayment(p)
	if err != nil {
		return order, nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return order, payResp, nil
}

// ✅ Get list of orders (user or admin)
func (s *service) GetOrders(userID uint, isAdmin bool) ([]Order, error) {
	if !isAdmin && userID == 0 {
		return nil, errors.New("unauthorized")
	}

	orders, err := s.repo.GetOrders(userID, isAdmin)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// ✅ Get order detail (user only sees their own order)
func (s *service) GetOrderDetail(userID, orderID uint, isAdmin bool) (*Order, error) {
	order, err := s.repo.GetOrderDetail(orderID)
	if err != nil {
		return nil, err
	}

	if !isAdmin && order.UserID != userID {
		return nil, fmt.Errorf("unauthorized: cannot access others' orders")
	}

	return order, nil
}

// ✅ Update order status (admin only)
func (s *service) UpdateOrderStatus(orderID uint, status OrderStatus) error {
	validStatuses := map[OrderStatus]bool{
		StatusAccepted: true,
		StatusRejected: true,
		StatusCanceled: true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	return s.repo.UpdateOrderStatus(orderID, status)
}
