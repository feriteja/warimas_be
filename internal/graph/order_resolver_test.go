package graph

import (
	"context"
	"errors"
	"testing"
	"time"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateFromSession(ctx context.Context, externalID string) (*order.Order, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *MockOrderService) OrderToPaymentProcess(ctx context.Context, sessionExternalID, externalID string, orderId uint) (*payment.PaymentResponse, error) {
	args := m.Called(ctx, sessionExternalID, externalID, orderId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentResponse), args.Error(1)
}

func (m *MockOrderService) UpdateOrderStatus(ctx context.Context, orderID uint, status order.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderService) MarkAsPaid(ctx context.Context, referenceID, paymentRequestID, paymentProviderID string) error {
	args := m.Called(ctx, referenceID, paymentRequestID, paymentProviderID)
	return args.Error(0)
}

func (m *MockOrderService) MarkAsFailed(ctx context.Context, referenceID, paymentRequestID, paymentProviderID string) error {
	args := m.Called(ctx, referenceID, paymentRequestID, paymentProviderID)
	return args.Error(0)
}

func (m *MockOrderService) CreateSession(ctx context.Context, input model.CreateCheckoutSessionInput) (*order.CheckoutSession, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.CheckoutSession), args.Error(1)
}

func (m *MockOrderService) UpdateSessionAddress(ctx context.Context, externalID string, addressID string, guestID *string) error {
	args := m.Called(ctx, externalID, addressID, guestID)
	return args.Error(0)
}

func (m *MockOrderService) ConfirmSession(ctx context.Context, externalID string) (*string, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockOrderService) GetOrders(ctx context.Context, filter *order.OrderFilterInput, sort *order.OrderSortInput, limit, page int32) ([]*order.Order, int64, map[uuid.UUID][]address.Address, error) {
	args := m.Called(ctx, filter, sort, limit, page)
	if args.Get(0) == nil {
		return nil, 0, nil, args.Error(3)
	}
	return args.Get(0).([]*order.Order), args.Get(1).(int64), args.Get(2).(map[uuid.UUID][]address.Address), args.Error(3)
}

func (m *MockOrderService) GetOrderDetail(ctx context.Context, orderID uint) (*order.Order, *address.Address, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*order.Order), args.Get(1).(*address.Address), args.Error(2)
}

func (m *MockOrderService) GetOrderDetailByExternalID(ctx context.Context, externalID string) (*order.Order, *address.Address, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*order.Order), args.Get(1).(*address.Address), args.Error(2)
}

func (m *MockOrderService) GetSession(ctx context.Context, externalID string) (*order.CheckoutSession, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.CheckoutSession), args.Error(1)
}

func (m *MockOrderService) GetPaymentOrderInfo(ctx context.Context, externalID string) (*order.PaymentOrderInfoResponse, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.PaymentOrderInfoResponse), args.Error(1)
}

func (m *MockOrderService) GetOrderForWebhook(ctx context.Context, externalID string) (*order.Order, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

// --- Tests ---

func TestMutationResolver_CreateCheckoutSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.CreateCheckoutSessionInput{
			Items: []*model.CheckoutSessionItemInput{{VariantID: "v1", Quantity: 1}},
		}
		expected := &order.CheckoutSession{ExternalID: "sess_123", Status: "PENDING"}

		mockSvc.On("CreateSession", ctx, input).Return(expected, nil)

		res, err := mr.CreateCheckoutSession(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, "sess_123", res.ExternalID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("EmptyItems", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		input := model.CreateCheckoutSessionInput{Items: []*model.CheckoutSessionItemInput{}}
		_, err := mr.CreateCheckoutSession(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, "items must not be empty", err.Error())
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.CreateCheckoutSessionInput{Items: []*model.CheckoutSessionItemInput{{VariantID: "v1", Quantity: 1}}}

		mockSvc.On("CreateSession", ctx, input).Return(nil, errors.New("stock error"))

		_, err := mr.CreateCheckoutSession(ctx, input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_ConfirmCheckoutSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ConfirmCheckoutSessionInput{ExternalID: "sess_123"}
		orderExtID := "ord_123"

		mockSvc.On("ConfirmSession", ctx, "sess_123").Return(&orderExtID, nil)

		res, err := mr.ConfirmCheckoutSession(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "ord_123", res.OrderExternalID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.ConfirmCheckoutSessionInput{ExternalID: "sess_123"}
		mockSvc.On("ConfirmSession", ctx, "sess_123").Return(nil, errors.New("db error"))
		_, err := mr.ConfirmCheckoutSession(ctx, input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_UpdateSessionAddress(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.UpdateSessionAddressInput{ExternalID: "sess_123", AddressID: "addr_1"}

		mockSvc.On("UpdateSessionAddress", ctx, "sess_123", "addr_1", (*string)(nil)).Return(nil)

		res, err := mr.UpdateSessionAddress(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.UpdateSessionAddressInput{ExternalID: "sess_123", AddressID: "addr_1"}
		mockSvc.On("UpdateSessionAddress", ctx, "sess_123", "addr_1", (*string)(nil)).Return(errors.New("db error"))
		_, err := mr.UpdateSessionAddress(ctx, input)
		assert.Error(t, err)
	})
}

func TestMutationResolver_UpdateOrderStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := context.Background()
		input := model.UpdateOrderStatusInput{
			OrderID: "10",
			Status:  model.OrderStatusPaid,
		}

		mockSvc.On("UpdateOrderStatus", ctx, uint(10), order.OrderStatusPaid).Return(nil)

		res, err := mr.UpdateOrderStatus(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		mockSvc.AssertExpectations(t)
	})

	t.Run("InvalidID", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		input := model.UpdateOrderStatusInput{OrderID: "abc"}
		res, _ := mr.UpdateOrderStatus(context.Background(), input)

		assert.False(t, res.Success)
		assert.Equal(t, "Invalid order ID", *res.Message)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		input := model.UpdateOrderStatusInput{OrderID: "10", Status: model.OrderStatusPaid}
		mockSvc.On("UpdateOrderStatus", context.Background(), uint(10), order.OrderStatusPaid).Return(errors.New("db error"))
		res, _ := mr.UpdateOrderStatus(context.Background(), input)
		assert.False(t, res.Success)
		assert.Equal(t, "db error", *res.Message)
	})
}

func TestMutationResolver_CreateOrderFromSession(t *testing.T) {
	t.Run("Forbidden_ExternalRequest", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		// Context without internal flag
		ctx := context.Background()
		input := model.CreateOrderFromSessionInput{ExternalID: "sess_123"}

		_, err := mr.CreateOrderFromSession(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, "forbidden", err.Error())
	})

	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		// Mock internal request (ensure utils.SetInternalContext exists or implement it)
		ctx := utils.SetInternalContext(context.Background())
		input := model.CreateOrderFromSessionInput{ExternalID: "sess_123"}

		userID := int32(1)
		now := time.Now()
		expectedOrder := &order.Order{
			ID:         1,
			ExternalID: "ord_123",
			UserID:     &userID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		mockSvc.On("CreateFromSession", ctx, "sess_123").Return(expectedOrder, nil)

		res, err := mr.CreateOrderFromSession(ctx, input)

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, int32(1), res.Order.ID)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		mr := &mutationResolver{resolver}

		ctx := utils.SetInternalContext(context.Background())
		input := model.CreateOrderFromSessionInput{ExternalID: "sess_123"}

		mockSvc.On("CreateFromSession", ctx, "sess_123").Return(nil, errors.New("db error"))
		_, err := mr.CreateOrderFromSession(ctx, input)
		assert.Error(t, err)
	})
}

func TestQueryResolver_OrderList(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		addrID := uuid.New()
		userID := int32(1)
		now := time.Now()
		expectedOrders := []*order.Order{{
			ID:        1,
			AddressID: addrID,
			UserID:    &userID,
			CreatedAt: now,
			UpdatedAt: now,
		}}
		expectedTotal := int64(1)
		addrMap := map[uuid.UUID][]address.Address{
			addrID: {{ID: addrID, Address1: "Street 1"}},
		}

		mockSvc.On("GetOrders", ctx, mock.Anything, mock.Anything, int32(20), int32(1)).
			Return(expectedOrders, expectedTotal, addrMap, nil)

		res, err := qr.OrderList(ctx, nil, nil, nil)

		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
		assert.Equal(t, int32(1), res.PageInfo.TotalItems)
	})

	t.Run("WithFilter", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		status := model.OrderStatusPaid
		filter := &model.OrderFilterInput{
			Status: &status,
			Search: utils.StrPtr("ORD-123"),
		}

		// Expect service to be called with mapped filter
		mockSvc.On("GetOrders", ctx, mock.MatchedBy(func(f *order.OrderFilterInput) bool {
			return *f.Status == order.OrderStatusPaid && *f.Search == "ORD-123"
		}), mock.Anything, int32(20), int32(1)).
			Return([]*order.Order{}, int64(0), map[uuid.UUID][]address.Address{}, nil)

		res, err := qr.OrderList(ctx, filter, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("WithSort", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		sortInput := &model.OrderSortInput{
			Field:     model.OrderSortFieldTotal,
			Direction: model.SortDirectionAsc,
		}

		// Expect service to be called with mapped sort options
		mockSvc.On("GetOrders", ctx, mock.Anything, mock.MatchedBy(func(s *order.OrderSortInput) bool {
			return s.Field == order.OrderSortFieldTotal && s.Direction == order.SortDirectionAsc
		}), int32(20), int32(1)).
			Return([]*order.Order{}, int64(0), map[uuid.UUID][]address.Address{}, nil)

		res, err := qr.OrderList(ctx, nil, sortInput, nil)

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("MissingAddress", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		addrID := uuid.New()
		userID := int32(1)
		now := time.Now()
		expectedOrders := []*order.Order{{
			ID:        1,
			AddressID: addrID,
			UserID:    &userID,
			CreatedAt: now,
			UpdatedAt: now,
		}}
		// Empty address map simulating missing address data
		addrMap := map[uuid.UUID][]address.Address{}

		mockSvc.On("GetOrders", ctx, mock.Anything, mock.Anything, int32(20), int32(1)).
			Return(expectedOrders, int64(1), addrMap, nil)

		res, err := qr.OrderList(ctx, nil, nil, nil)

		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
		// Ensure no panic and address is nil
		assert.Nil(t, res.Items[0].Shipping)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		mockSvc.On("GetOrders", context.Background(), mock.Anything, mock.Anything, int32(20), int32(1)).Return(nil, int64(0), nil, errors.New("db error"))
		_, err := qr.OrderList(context.Background(), nil, nil, nil)
		assert.Error(t, err)
	})
}

func TestQueryResolver_OrderDetail(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		orderID := uint(123)
		userID := int32(1)
		now := time.Now()
		expectedOrder := &order.Order{
			ID:        123,
			UserID:    &userID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		expectedAddr := &address.Address{ID: uuid.New()}

		mockSvc.On("GetOrderDetail", ctx, orderID).Return(expectedOrder, expectedAddr, nil)

		res, err := qr.OrderDetail(ctx, "123")

		assert.NoError(t, err)
		assert.Equal(t, int32(123), res.ID)
	})

	t.Run("InvalidID", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}
		_, err := qr.OrderDetail(context.Background(), "abc")
		assert.Error(t, err)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetOrderDetail", context.Background(), uint(123)).Return(nil, nil, errors.New("db error"))
		_, err := qr.OrderDetail(context.Background(), "123")
		assert.Error(t, err)
	})
}

func TestQueryResolver_OrderDetailByExternalID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		extID := "ext_123"
		userID := int32(1)
		now := time.Now()
		expectedOrder := &order.Order{
			ID:        123,
			UserID:    &userID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		expectedAddr := &address.Address{ID: uuid.New()}

		mockSvc.On("GetOrderDetailByExternalID", ctx, extID).Return(expectedOrder, expectedAddr, nil)

		res, err := qr.OrderDetailByExternalID(ctx, extID)

		assert.NoError(t, err)
		assert.Equal(t, int32(123), res.ID)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetOrderDetailByExternalID", context.Background(), "ext_123").Return(nil, nil, errors.New("db error"))
		_, err := qr.OrderDetailByExternalID(context.Background(), "ext_123")
		assert.Error(t, err)
	})
}

func TestQueryResolver_CheckoutSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		extID := "sess_123"
		expectedSession := &order.CheckoutSession{
			ExternalID: extID,
			Status:     "PENDING",
		}

		mockSvc.On("GetSession", ctx, extID).Return(expectedSession, nil)

		res, err := qr.CheckoutSession(ctx, extID)

		assert.NoError(t, err)
		assert.Equal(t, extID, res.ExternalID)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetSession", context.Background(), "sess_123").Return(nil, errors.New("db error"))
		_, err := qr.CheckoutSession(context.Background(), "sess_123")
		assert.Error(t, err)
	})
}

func TestQueryResolver_PaymentOrderInfo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}

		ctx := context.Background()
		extID := "ord_123"
		expectedInfo := &order.PaymentOrderInfoResponse{
			OrderExternalID: extID,
			Status:          "PENDING",
			TotalAmount:     10000,
			Currency:        "IDR",
			Payment: order.PaymentDetail{
				Method: "BCA",
			},
		}

		mockSvc.On("GetPaymentOrderInfo", ctx, extID).Return(expectedInfo, nil)

		res, err := qr.PaymentOrderInfo(ctx, extID)

		assert.NoError(t, err)
		assert.Equal(t, extID, res.OrderExternalID)
		assert.Equal(t, int32(10000), res.TotalAmount)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockSvc := new(MockOrderService)
		resolver := &Resolver{OrderSvc: mockSvc}
		qr := &queryResolver{resolver}
		mockSvc.On("GetPaymentOrderInfo", context.Background(), "ord_123").Return(nil, errors.New("db error"))
		_, err := qr.PaymentOrderInfo(context.Background(), "ord_123")
		assert.Error(t, err)
	})
}
