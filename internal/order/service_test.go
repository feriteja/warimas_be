package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/payment"
	"warimas-be/internal/product"
	"warimas-be/internal/user"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

// Stubs for methods used in tests
func (m *MockRepository) GetOrderDetail(ctx context.Context, orderID uint) (*Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Order), args.Error(1)
}

func (m *MockRepository) GetCheckoutSession(ctx context.Context, externalID string) (*CheckoutSession, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CheckoutSession), args.Error(1)
}

func (m *MockRepository) GetOrderBySessionID(ctx context.Context, sessionID uuid.UUID) (*Order, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Order), args.Error(1)
}

func (m *MockRepository) CreateOrderTx(ctx context.Context, order *Order, session *CheckoutSession) error {
	args := m.Called(ctx, order, session)
	return args.Error(0)
}

// Stubbing other interface methods to satisfy Repository interface (if strict)
func (m *MockRepository) FetchOrders(ctx context.Context, filter *OrderFilterInput, sort *OrderSortInput, limit int32, offset int32) ([]*Order, error) {
	args := m.Called(ctx, filter, sort, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Order), args.Error(1)
}
func (m *MockRepository) CountOrders(ctx context.Context, filter *OrderFilterInput) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockRepository) FetchOrderItems(ctx context.Context, orderIDs []int32) (map[int32][]*OrderItem, error) {
	args := m.Called(ctx, orderIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int32][]*OrderItem), args.Error(1)
}
func (m *MockRepository) GetOrderDetailByExternalID(ctx context.Context, externalId string) (*Order, error) {
	args := m.Called(ctx, externalId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Order), args.Error(1)
}
func (m *MockRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status OrderStatus, invoiceNumber *string) error {
	args := m.Called(ctx, orderID, status, invoiceNumber)
	return args.Error(0)
}
func (m *MockRepository) GetByReferenceID(ctx context.Context, refID string) (*Order, error) {
	args := m.Called(ctx, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Order), args.Error(1)
}
func (m *MockRepository) UpdateStatusByReferenceID(ctx context.Context, refID, payReqID, payProvID, status string) error {
	args := m.Called(ctx, refID, payReqID, payProvID, status)
	return args.Error(0)
}
func (m *MockRepository) GetVariantForCheckout(ctx context.Context, variantID string) (*product.Variant, *product.Product, error) {
	args := m.Called(ctx, variantID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*product.Variant), args.Get(1).(*product.Product), args.Error(2)
}
func (m *MockRepository) CreateCheckoutSession(ctx context.Context, session *CheckoutSession, items []CheckoutSessionItem) error {
	args := m.Called(ctx, session, items)
	return args.Error(0)
}
func (m *MockRepository) GetUserAddress(ctx context.Context, addressID string, userID uint) (*address.Address, error) {
	args := m.Called(ctx, addressID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*address.Address), args.Error(1)
}
func (m *MockRepository) UpdateSessionAddressAndPricing(ctx context.Context, session *CheckoutSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) UpdateSessionPaymentMethod(ctx context.Context, sessionID uuid.UUID, paymentMethod payment.ChannelCode) error {
	args := m.Called(ctx, sessionID, paymentMethod)
	return args.Error(0)
}

func (m *MockRepository) ValidateVariantStock(ctx context.Context, variantID string, qty int) (bool, error) {
	args := m.Called(ctx, variantID, qty)
	return args.Bool(0), args.Error(1)
}
func (m *MockRepository) ConfirmCheckoutSession(ctx context.Context, session *CheckoutSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}
func (m *MockRepository) MarkSessionExpired(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockRepository) GetOrderByExternalID(ctx context.Context, externalID string) (*Order, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Order), args.Error(1)
}

type MockAddressRepository struct {
	mock.Mock
}

func (m *MockAddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*address.Address, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*address.Address), args.Error(1)
}

func (m *MockAddressRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]address.Address, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]address.Address), args.Error(1)
}

func (m *MockAddressRepository) GetByUserID(ctx context.Context, userID uint) ([]*address.Address, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*address.Address), args.Error(1)
}
func (m *MockAddressRepository) Create(ctx context.Context, addr *address.Address) error {
	args := m.Called(ctx, addr)
	return args.Error(0)
}
func (m *MockAddressRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockAddressRepository) ClearDefault(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
func (m *MockAddressRepository) SetDefault(ctx context.Context, userID uint, addressID uuid.UUID) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) SavePayment(ctx context.Context, p *payment.Payment) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *MockPaymentRepository) UpdatePaymentStatus(ctx context.Context, externalID, status string) error {
	args := m.Called(ctx, externalID, status)
	return args.Error(0)
}
func (m *MockPaymentRepository) GetPaymentByOrder(ctx context.Context, orderID uint) (*payment.Payment, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.Payment), args.Error(1)
}

func (m *MockPaymentRepository) MarkWebhookFailed(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *MockPaymentRepository) MarkWebhookProcessed(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPaymentRepository) SavePaymentWebhook(
	ctx context.Context,
	provider string,
	eventID string,
	eventType string,
	externalID string,
	payload json.RawMessage,
	signatureValid bool,
) (int64, bool, error) {
	args := m.Called(ctx, provider, eventID, eventType, externalID, payload, signatureValid)
	return args.Get(0).(int64), args.Bool(1), args.Error(2)
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreateInvoice(externalID, userEmail string, amount int64, payerEmail string, items []payment.XenditItem, channel payment.ChannelCode) (*payment.PaymentResponse, error) {
	args := m.Called(externalID, userEmail, amount, payerEmail, items, channel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentResponse), args.Error(1)
}

func (m *MockPaymentGateway) GetPaymentStatus(externalID string) (*payment.PaymentStatus, error) {
	args := m.Called(externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentStatus), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetProfile(ctx context.Context, userID uint) (*user.Profile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Profile), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockPaymentGateway) CancelPayment(externalID string) error {
	args := m.Called(externalID)
	return args.Error(0)
}

func (m *MockPaymentGateway) VerifySignature(r *http.Request) error {
	args := m.Called(r)
	return args.Error(0)
}

// --- Tests ---

func TestService_GetOrderDetail(t *testing.T) {
	orderID := uint(100)
	userID := uint(1)
	addrID := uuid.New()
	userInt32 := int32(userID)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

		mockOrder := &Order{
			ID:        int32(orderID),
			UserID:    &userInt32,
			AddressID: addrID,
		}
		mockAddr := &address.Address{ID: addrID, Name: "Home"}

		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)
		mockAddrRepo.On("GetByID", ctx, addrID).Return(mockAddr, nil)

		resOrder, resAddr, err := svc.GetOrderDetail(ctx, orderID)

		assert.NoError(t, err)
		assert.Equal(t, mockOrder, resOrder)
		assert.Equal(t, mockAddr, resAddr)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		// Context without user
		ctx := context.Background()

		mockOrder := &Order{ID: int32(orderID)}
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)

		_, _, err := svc.GetOrderDetail(ctx, orderID)

		// Should return ErrUnauthorized (and log error)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("Unauthorized_WrongUser", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

		otherUser := int32(999)
		mockOrder := &Order{
			ID:     int32(orderID),
			UserID: &otherUser, // Belongs to someone else
		}

		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)

		_, _, err := svc.GetOrderDetail(ctx, orderID)

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

		mockRepo.On("GetOrderDetail", ctx, orderID).Return(nil, nil) // Repo returns nil, nil for not found

		_, _, err := svc.GetOrderDetail(ctx, orderID)

		assert.Error(t, err)
		assert.Equal(t, ErrOrderNotFound, err)
	})

	t.Run("AddressRepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)
		ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

		mockOrder := &Order{ID: int32(orderID), UserID: &userInt32, AddressID: addrID}
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)
		mockAddrRepo.On("GetByID", ctx, addrID).Return(nil, errors.New("addr error"))

		_, _, err := svc.GetOrderDetail(ctx, orderID)
		assert.Error(t, err)
	})

	t.Run("InvalidData_NilUserID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

		mockOrder := &Order{ID: int32(orderID), UserID: nil} // Invalid
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)

		_, _, err := svc.GetOrderDetail(ctx, orderID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid order data")
	})

	t.Run("Success_Admin", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		// Context with ADMIN role
		ctx := utils.SetUserContext(context.Background(), userID, "admin@example.com", "ADMIN")

		otherUser := int32(999)
		mockOrder := &Order{ID: int32(orderID), UserID: &otherUser, AddressID: addrID}
		mockAddr := &address.Address{ID: addrID}

		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)
		mockAddrRepo.On("GetByID", ctx, addrID).Return(mockAddr, nil)

		res, _, err := svc.GetOrderDetail(ctx, orderID)
		assert.NoError(t, err)
		assert.Equal(t, mockOrder, res)
	})
}

func TestService_CreateFromSession(t *testing.T) {
	ctx := context.Background()
	externalID := "sess-123"
	sessionID := uuid.New()
	userID := int32(1)
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			ID:          sessionID,
			UserID:      &userID,
			Status:      CheckoutSessionStatusPaid,
			ConfirmedAt: &now,
			TotalPrice:  10000,
			Currency:    "IDR",
		}

		// 1. Get Session
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		// 2. Check Existing Order (Not found)
		mockRepo.On("GetOrderBySessionID", ctx, sessionID).Return(nil, errors.New("not found"))
		// 3. Create Order
		mockRepo.On("CreateOrderTx", ctx, mock.AnythingOfType("*order.Order"), mockSession).Return(nil)

		order, err := svc.CreateFromSession(ctx, externalID)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, uint(10000), order.TotalAmount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("SessionNotPaid", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			ID:          sessionID,
			Status:      CheckoutSessionStatusPending, // Not paid
			ConfirmedAt: &now,
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.CreateFromSession(ctx, externalID)

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "payment not completed")
		}
	})

	t.Run("Idempotency_OrderExists", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			ID:          sessionID,
			UserID:      &userID,
			Status:      CheckoutSessionStatusPaid,
			ConfirmedAt: &now,
		}

		mockExistingOrder := &Order{ID: 123, ExternalID: "ord-123"}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetOrderBySessionID", ctx, sessionID).Return(mockExistingOrder, nil)

		order, err := svc.CreateFromSession(ctx, externalID)

		assert.NoError(t, err)
		assert.Equal(t, mockExistingOrder, order)
		mockRepo.AssertExpectations(t)
	})

	t.Run("SessionNotConfirmed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{ID: sessionID, ConfirmedAt: nil}
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.CreateFromSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checkout session not confirmed")
	})
}

func TestService_GetOrders(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAddrRepo := new(MockAddressRepository)
	svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success", func(t *testing.T) {
		filter := &OrderFilterInput{}
		sort := &OrderSortInput{Field: OrderSortFieldCreatedAt, Direction: SortDirectionDesc}
		limit := int32(10)
		page := int32(1)
		offset := int32(0)

		orderID := int32(100)
		addrID := uuid.New()

		mockOrders := []*Order{
			{ID: orderID, AddressID: addrID},
		}

		mockItems := map[int32][]*OrderItem{
			orderID: {{ID: 1, VariantID: "item-1"}},
		}

		mockAddrs := []address.Address{
			{ID: addrID, Name: "Home"},
		}

		// 1. Fetch Orders
		mockRepo.On("FetchOrders", ctx, filter, sort, limit, offset).Return(mockOrders, nil)
		// 2. Count
		mockRepo.On("CountOrders", ctx, filter).Return(int64(1), nil)
		// 3. Fetch Items
		mockRepo.On("FetchOrderItems", ctx, []int32{orderID}).Return(mockItems, nil)
		// 4. Fetch Addresses
		mockAddrRepo.On("GetByIDs", ctx, []uuid.UUID{addrID}).Return(mockAddrs, nil)

		orders, total, addrMap, err := svc.GetOrders(ctx, filter, sort, limit, page)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, orders, 1)
		assert.Len(t, addrMap, 1)
		mockRepo.AssertExpectations(t)
		mockAddrRepo.AssertExpectations(t)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		filter := &OrderFilterInput{}
		sort := &OrderSortInput{Field: OrderSortFieldCreatedAt, Direction: SortDirectionDesc}

		mockRepo.On("FetchOrders", ctx, filter, sort, int32(10), int32(0)).Return(nil, errors.New("db error"))

		_, _, _, err := svc.GetOrders(ctx, filter, sort, 10, 1)
		assert.Error(t, err)
	})

	t.Run("CountError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		filter := &OrderFilterInput{}
		sort := &OrderSortInput{Field: OrderSortFieldCreatedAt, Direction: SortDirectionDesc}

		mockRepo.On("FetchOrders", ctx, filter, sort, int32(10), int32(0)).Return([]*Order{{ID: 1}}, nil)
		mockRepo.On("CountOrders", ctx, filter).Return(int64(0), errors.New("count error"))

		_, _, _, err := svc.GetOrders(ctx, filter, sort, 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "count error")
	})

	t.Run("AddressRepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)
		filter := &OrderFilterInput{}
		sort := &OrderSortInput{Field: OrderSortFieldCreatedAt, Direction: SortDirectionDesc}
		addrID := uuid.New()

		mockRepo.On("FetchOrders", ctx, filter, sort, int32(10), int32(0)).Return([]*Order{{ID: 1, AddressID: addrID}}, nil)
		mockRepo.On("CountOrders", ctx, filter).Return(int64(1), nil)
		mockAddrRepo.On("GetByIDs", ctx, []uuid.UUID{addrID}).Return(nil, errors.New("addr error"))

		_, _, _, err := svc.GetOrders(ctx, filter, sort, 10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "addr error")
	})

	t.Run("FetchItemsError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)
		filter := &OrderFilterInput{}
		sort := &OrderSortInput{Field: OrderSortFieldCreatedAt, Direction: SortDirectionDesc}
		addrID := uuid.New()

		mockRepo.On("FetchOrders", ctx, filter, sort, int32(10), int32(0)).Return([]*Order{{ID: 1, AddressID: addrID}}, nil)
		mockRepo.On("CountOrders", ctx, filter).Return(int64(1), nil)
		mockAddrRepo.On("GetByIDs", ctx, []uuid.UUID{addrID}).Return([]address.Address{{ID: addrID}}, nil)
		mockRepo.On("FetchOrderItems", ctx, []int32{1}).Return(nil, errors.New("items error"))

		_, _, _, err := svc.GetOrders(ctx, filter, sort, 10, 1)
		assert.Error(t, err)
	})
}

func TestService_UpdateOrderStatus(t *testing.T) {
	orderID := uint(100)
	ctx := context.Background()

	tests := []struct {
		name          string
		currentStatus OrderStatus
		newStatus     OrderStatus
		expectError   bool
		errorMsg      string
	}{
		// --- Success Cases ---
		{"Pending -> Paid", OrderStatusPendingPayment, OrderStatusPaid, false, ""},
		{"Pending -> Cancelled", OrderStatusPendingPayment, OrderStatusCancelled, false, ""},
		{"Pending -> Failed", OrderStatusPendingPayment, OrderStatusFailed, false, ""},

		{"Paid -> Accepted", OrderStatusPaid, OrderStatusAccepted, false, ""},
		{"Paid -> Cancelled", OrderStatusPaid, OrderStatusCancelled, false, ""},
		{"Paid -> Failed", OrderStatusPaid, OrderStatusFailed, false, ""},

		{"Accepted -> Shipped", OrderStatusAccepted, OrderStatusShipped, false, ""},
		{"Accepted -> Cancelled", OrderStatusAccepted, OrderStatusCancelled, false, ""},
		{"Accepted -> Failed", OrderStatusAccepted, OrderStatusFailed, false, ""},

		{"Shipped -> Completed", OrderStatusShipped, OrderStatusCompleted, false, ""},
		{"Shipped -> Failed", OrderStatusShipped, OrderStatusFailed, false, ""},

		// --- Invalid Transitions (Jumps) ---
		{"Pending -> Accepted", OrderStatusPendingPayment, OrderStatusAccepted, true, "invalid status transition"},
		{"Pending -> Shipped", OrderStatusPendingPayment, OrderStatusShipped, true, "invalid status transition"},
		{"Pending -> Completed", OrderStatusPendingPayment, OrderStatusCompleted, true, "invalid status transition"},

		{"Paid -> Shipped", OrderStatusPaid, OrderStatusShipped, true, "invalid status transition"},
		{"Paid -> Completed", OrderStatusPaid, OrderStatusCompleted, true, "invalid status transition"},

		{"Accepted -> Completed", OrderStatusAccepted, OrderStatusCompleted, true, "invalid status transition"},

		// --- Invalid Transitions (Backward) ---
		{"Paid -> Pending", OrderStatusPaid, OrderStatusPendingPayment, true, "invalid status transition"},
		{"Accepted -> Paid", OrderStatusAccepted, OrderStatusPaid, true, "invalid status transition"},
		{"Shipped -> Accepted", OrderStatusShipped, OrderStatusAccepted, true, "invalid status transition"},

		// --- Specific Rules ---
		// Rule 3: status can't be canceled/backward once it's been shipped
		{"Shipped -> Cancelled", OrderStatusShipped, OrderStatusCancelled, true, "invalid status transition"},

		// --- Terminal Statuses (Rule 4 & 7) ---
		{"Completed -> Failed", OrderStatusCompleted, OrderStatusFailed, true, "terminal status"},
		{"Completed -> Pending", OrderStatusCompleted, OrderStatusPendingPayment, true, "terminal status"},

		{"Cancelled -> Paid", OrderStatusCancelled, OrderStatusPaid, true, "terminal status"},
		{"Cancelled -> Failed", OrderStatusCancelled, OrderStatusFailed, true, "terminal status"},

		{"Failed -> Paid", OrderStatusFailed, OrderStatusPaid, true, "terminal status"},
		{"Failed -> Pending", OrderStatusFailed, OrderStatusPendingPayment, true, "terminal status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo, nil, nil, nil, nil)

			mockOrder := &Order{Status: tt.currentStatus}
			mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)

			if !tt.expectError {
				var invMatcher interface{}
				if tt.newStatus == OrderStatusAccepted {
					invMatcher = mock.AnythingOfType("*string")
				} else {
					invMatcher = (*string)(nil)
				}
				mockRepo.On("UpdateOrderStatus", ctx, orderID, tt.newStatus, invMatcher).Return(nil)
			}

			err := svc.UpdateOrderStatus(ctx, orderID, tt.newStatus)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}

	t.Run("OrderNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(nil, nil) // nil order
		err := svc.UpdateOrderStatus(ctx, orderID, OrderStatusPaid)
		assert.Error(t, err)
		assert.Equal(t, ErrOrderNotFound, err)
	})

	t.Run("RepoError_GetOrder", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(nil, errors.New("db error"))
		err := svc.UpdateOrderStatus(ctx, orderID, OrderStatusPaid)
		assert.Error(t, err)
	})

	t.Run("RepoError_Update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockOrder := &Order{Status: OrderStatusPendingPayment}
		mockRepo.On("GetOrderDetail", ctx, orderID).Return(mockOrder, nil)
		mockRepo.On("UpdateOrderStatus", ctx, orderID, OrderStatusPaid, (*string)(nil)).Return(errors.New("update error"))
		err := svc.UpdateOrderStatus(ctx, orderID, OrderStatusPaid)
		assert.Error(t, err)
	})
}

func TestService_ConfirmSession(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "sess-ext-1"
	sessionID := uuid.New()
	addrID := uuid.New()
	now := time.Now().Add(1 * time.Hour)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockPayRepo := new(MockPaymentRepository)
		mockPayGate := new(MockPaymentGateway)
		mockUserRepo := new(MockUserRepository)
		svc := NewService(mockRepo, mockPayRepo, mockPayGate, nil, mockUserRepo)

		pm := payment.MethodBCAVA

		mockSession := &CheckoutSession{
			ID:         sessionID,
			ExternalID: externalID,
			UserID:     &userInt32,
			Status:     CheckoutSessionStatusPending,
			ExpiresAt:  now,
			AddressID:  &addrID,
			TotalPrice: 50000,
			Items: []CheckoutSessionItem{
				{VariantID: "v1", Quantity: 1, Price: 50000, ProductName: "P1", VariantName: "V1"},
			},
			PaymentMethod: &pm,
		}

		// 1. Get Session
		mockRepo.On("GetCheckoutSession", mock.Anything, externalID).Return(mockSession, nil).Times(1)

		// 2. Validate Stock
		mockRepo.On("ValidateVariantStock", ctx, "v1", 1).Return(true, nil)

		// 3. Idempotency Check (No existing order)
		mockRepo.On("GetOrderBySessionID", ctx, sessionID).Return(nil, nil)

		// 4. Create Order Tx
		mockRepo.On("CreateOrderTx", ctx, mock.AnythingOfType("*order.Order"), mockSession).Return(nil)

		// 5. Confirm Session
		mockRepo.On("ConfirmCheckoutSession", ctx, mockSession).Return(nil)

		// 6. Payment Gateway (Create Invoice)
		mockPayResp := &payment.PaymentResponse{
			ProviderPaymentID: "pay-1",
			InvoiceURL:        "http://invoice",
			Status:            "PENDING",
		}
		mockPayGate.On("CreateInvoice", mock.AnythingOfType("string"), "userName", int64(50000), "test@example.com", mock.Anything, payment.ChannelCode(payment.MethodBCAVA)).Return(mockPayResp, nil)

		// 7. Save Payment
		mockPayRepo.On("SavePayment", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)

		// 8. Get User Profile
		mockUserRepo.On("GetProfile", ctx, userID).Return(&user.Profile{FullName: utils.StrPtr("userName")}, nil)

		res, err := svc.ConfirmSession(ctx, externalID)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockRepo.AssertExpectations(t)
		mockPayGate.AssertExpectations(t)
		mockPayRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("OutOfStock", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			ID:         sessionID,
			ExternalID: externalID,
			UserID:     &userInt32,
			Status:     CheckoutSessionStatusPending,
			ExpiresAt:  now,
			AddressID:  &addrID,
			Items: []CheckoutSessionItem{
				{VariantID: "v1", Quantity: 1},
			},
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("ValidateVariantStock", ctx, "v1", 1).Return(false, nil)

		_, err := svc.ConfirmSession(ctx, externalID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product out of stock")
		mockRepo.AssertExpectations(t)
	})
}

func TestService_UpdateSessionAddress(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "sess-ext-1"
	addrIDStr := uuid.New().String()
	now := time.Now().Add(1 * time.Hour)

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: now,
			Subtotal:  10000,
		}

		mockAddr := &address.Address{
			ID:   uuid.MustParse(addrIDStr),
			City: "Jakarta",
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctx, addrIDStr, userID).Return(mockAddr, nil)
		mockRepo.On("UpdateSessionAddressAndPricing", ctx, mockSession).Return(nil)

		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Expired", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checkout session expired")
	})

	t.Run("NotEditable", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID: &userInt32,
			Status: CheckoutSessionStatusPaid, // Already paid
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checkout session is not editable")
	})

	t.Run("Guest_Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		ctxGuest := context.Background()
		guestID := uuid.New()
		guestIDStr := guestID.String()

		mockSession := &CheckoutSession{
			GuestID:   &guestID,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: now,
			Subtotal:  10000,
		}

		mockAddr := &address.Address{ID: uuid.MustParse(addrIDStr), City: "Jakarta"}

		mockRepo.On("GetCheckoutSession", ctxGuest, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctxGuest, addrIDStr, uint(0)).Return(mockAddr, nil)
		mockRepo.On("UpdateSessionAddressAndPricing", ctxGuest, mockSession).Return(nil)

		err := svc.UpdateSessionAddress(ctxGuest, externalID, addrIDStr, &guestIDStr)
		assert.NoError(t, err)
	})

	t.Run("Guest_Forbidden_Mismatch", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		ctxGuest := context.Background()
		guestID := uuid.New()
		otherGuestIDStr := uuid.New().String()

		mockSession := &CheckoutSession{GuestID: &guestID}
		mockRepo.On("GetCheckoutSession", ctxGuest, externalID).Return(mockSession, nil)

		err := svc.UpdateSessionAddress(ctxGuest, externalID, addrIDStr, &otherGuestIDStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("RepoError_GetSession", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(nil, errors.New("db error"))
		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.Error(t, err)
	})

	t.Run("RepoError_GetAddress", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockSession := &CheckoutSession{UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now}
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctx, addrIDStr, userID).Return(nil, errors.New("addr error"))
		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.Error(t, err)
	})

	t.Run("RepoError_Update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockSession := &CheckoutSession{UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now}
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctx, addrIDStr, userID).Return(&address.Address{ID: uuid.MustParse(addrIDStr)}, nil)
		mockRepo.On("UpdateSessionAddressAndPricing", ctx, mockSession).Return(errors.New("update error"))
		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.Error(t, err)
	})

	t.Run("ShippingFee_Jakarta", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockSession := &CheckoutSession{UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now}
		mockAddr := &address.Address{ID: uuid.MustParse(addrIDStr), City: "Jakarta"}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctx, addrIDStr, userID).Return(mockAddr, nil)
		// Expect shipping fee 10000 for Jakarta
		mockRepo.On("UpdateSessionAddressAndPricing", ctx, mock.MatchedBy(func(s *CheckoutSession) bool {
			return s.ShippingFee == 10000
		})).Return(nil)

		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.NoError(t, err)
	})

	t.Run("ShippingFee_Other", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockSession := &CheckoutSession{UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now}
		mockAddr := &address.Address{ID: uuid.MustParse(addrIDStr), City: "Bandung"}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("GetUserAddress", ctx, addrIDStr, userID).Return(mockAddr, nil)
		// Expect shipping fee 20000 for non-Jakarta
		mockRepo.On("UpdateSessionAddressAndPricing", ctx, mock.MatchedBy(func(s *CheckoutSession) bool {
			return s.ShippingFee == 20000
		})).Return(nil)

		err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
		assert.NoError(t, err)
	})
}

func TestService_MarkAsPaid(t *testing.T) {
	ctx := context.Background()
	refID := "ord-ref-1"
	payReqID := "pay-req-1"
	provID := "prov-1"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{
			Status: OrderStatusPendingPayment,
		}

		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)
		mockRepo.On("UpdateStatusByReferenceID", ctx, refID, payReqID, provID, "PAID").Return(nil)

		err := svc.MarkAsPaid(ctx, refID, payReqID, provID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("AlreadyPaid", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{Status: OrderStatusPaid}
		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)

		err := svc.MarkAsPaid(ctx, refID, payReqID, provID)
		assert.NoError(t, err) // Should return nil (idempotent)
	})

	t.Run("InvalidTransition_FailedToPaid", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{Status: OrderStatusFailed}
		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)

		err := svc.MarkAsPaid(ctx, refID, payReqID, provID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status transition")
	})

	t.Run("RepoError_GetOrder", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetByReferenceID", ctx, refID).Return(nil, errors.New("db error"))
		err := svc.MarkAsPaid(ctx, refID, payReqID, provID)
		assert.Error(t, err)
	})

	t.Run("RepoError_UpdateStatus", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockOrder := &Order{Status: OrderStatusPendingPayment}
		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)
		mockRepo.On("UpdateStatusByReferenceID", ctx, refID, payReqID, provID, "PAID").Return(errors.New("update error"))
		err := svc.MarkAsPaid(ctx, refID, payReqID, provID)
		assert.Error(t, err)
	})
}

func TestService_MarkAsFailed(t *testing.T) {
	ctx := context.Background()
	refID := "ord-ref-1"
	payReqID := "pay-req-1"
	provID := "prov-1"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{
			Status: OrderStatusPendingPayment,
		}

		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)
		mockRepo.On("UpdateStatusByReferenceID", ctx, refID, payReqID, provID, "FAILED").Return(nil)

		err := svc.MarkAsFailed(ctx, refID, payReqID, provID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("AlreadyFailed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{Status: OrderStatusFailed}
		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)

		err := svc.MarkAsFailed(ctx, refID, payReqID, provID)
		assert.NoError(t, err)
	})

	t.Run("InvalidTransition_PaidToFailed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockOrder := &Order{Status: OrderStatusPaid}
		mockRepo.On("GetByReferenceID", ctx, refID).Return(mockOrder, nil)

		err := svc.MarkAsFailed(ctx, refID, payReqID, provID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status transition")
	})
}

func TestService_GetOrderDetailByExternalID(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	extID := "ord-ext-1"
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, nil, nil, mockAddrRepo, nil)

		mockOrder := &Order{ID: 1, ExternalID: extID, UserID: &userInt32, AddressID: addrID}
		mockAddr := &address.Address{ID: addrID}

		mockRepo.On("GetOrderDetailByExternalID", ctx, extID).Return(mockOrder, nil)
		mockAddrRepo.On("GetByID", ctx, addrID).Return(mockAddr, nil)

		res, _, err := svc.GetOrderDetailByExternalID(ctx, extID)
		assert.NoError(t, err)
		assert.Equal(t, extID, res.ExternalID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockRepo.On("GetOrderDetailByExternalID", ctx, extID).Return(nil, nil)

		_, _, err := svc.GetOrderDetailByExternalID(ctx, extID)
		assert.Error(t, err)
		assert.Equal(t, ErrOrderNotFound, err)
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		ctx := context.Background()

		mockOrder := &Order{ID: 1, ExternalID: extID}
		mockRepo.On("GetOrderDetailByExternalID", ctx, extID).Return(mockOrder, nil)

		_, _, err := svc.GetOrderDetailByExternalID(ctx, extID)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("Unauthorized_WrongUser", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		otherUser := int32(999)
		mockOrder := &Order{ID: 1, ExternalID: extID, UserID: &otherUser}
		mockRepo.On("GetOrderDetailByExternalID", ctx, extID).Return(mockOrder, nil)

		_, _, err := svc.GetOrderDetailByExternalID(ctx, extID)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
}

func TestService_CreateSession(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		input := model.CreateCheckoutSessionInput{
			Items: []*model.CheckoutSessionItemInput{
				{VariantID: "var-1", Quantity: 1},
			},
		}

		mockVariant := &product.Variant{
			ID:    "var-1",
			Price: 10000,
		}
		mockProduct := &product.Product{
			ID:   "1",
			Name: "Product 1",
		}

		// 2. Get Variant Info
		mockRepo.On("GetVariantForCheckout", ctx, "var-1").Return(mockVariant, mockProduct, nil)
		// 3. Create Session
		mockRepo.On("CreateCheckoutSession", ctx, mock.AnythingOfType("*order.CheckoutSession"), mock.Anything).Return(nil)

		res, err := svc.CreateSession(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 11000, res.TotalPrice)
		mockRepo.AssertExpectations(t)
	})

	t.Run("InvalidQuantity", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		input := model.CreateCheckoutSessionInput{
			Items: []*model.CheckoutSessionItemInput{
				{VariantID: "var-1", Quantity: 0},
			},
		}
		_, err := svc.CreateSession(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be greater than zero")
	})

	t.Run("RepoError_CreateSession", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		input := model.CreateCheckoutSessionInput{
			Items: []*model.CheckoutSessionItemInput{{VariantID: "var-1", Quantity: 1}},
		}
		mockRepo.On("GetVariantForCheckout", ctx, "var-1").Return(&product.Variant{Price: 1000}, &product.Product{}, nil)
		mockRepo.On("CreateCheckoutSession", ctx, mock.Anything, mock.Anything).Return(errors.New("db error"))

		_, err := svc.CreateSession(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("GetVariantError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		input := model.CreateCheckoutSessionInput{
			Items: []*model.CheckoutSessionItemInput{{VariantID: "var-1", Quantity: 1}},
		}
		mockRepo.On("GetVariantForCheckout", ctx, "var-1").Return(nil, nil, errors.New("var error"))

		_, err := svc.CreateSession(ctx, input)
		assert.Error(t, err)
	})
}

func TestService_GetSession(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "sess-ext-1"

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		res, err := svc.GetSession(ctx, externalID)
		assert.NoError(t, err)
		assert.Equal(t, mockSession, res)
	})

	t.Run("Expired", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		sessionID := uuid.New()
		mockSession := &CheckoutSession{
			ID:        sessionID,
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("MarkSessionExpired", ctx, sessionID).Return(nil)

		res, err := svc.GetSession(ctx, externalID)
		assert.NoError(t, err)
		assert.Equal(t, CheckoutSessionStatusExpired, res.Status)
	})

	t.Run("Forbidden", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		otherUser := int32(999)
		mockSession := &CheckoutSession{UserID: &otherUser}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.GetSession(ctx, externalID)
		assert.Error(t, err)
		assert.Equal(t, "forbidden", err.Error())
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(nil, errors.New("db error"))

		_, err := svc.GetSession(ctx, externalID)
		assert.Error(t, err)
	})
}

func TestService_GetPaymentOrderInfo(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "ord-ext-1"
	addrID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockPayRepo := new(MockPaymentRepository)
		mockAddrRepo := new(MockAddressRepository)
		svc := NewService(mockRepo, mockPayRepo, nil, mockAddrRepo, nil)

		mockOrder := &Order{
			ID:          1,
			UserID:      &userInt32,
			AddressID:   addrID,
			TotalAmount: 50000,
			Currency:    "IDR",
		}
		mockPayment := &payment.Payment{
			Status:        "PENDING",
			PaymentMethod: "BCA",
			PaymentCode:   "123456",
		}
		mockAddr := &address.Address{
			ID:   addrID,
			Name: "Home",
		}

		mockRepo.On("GetOrderByExternalID", ctx, externalID).Return(mockOrder, nil)
		mockPayRepo.On("GetPaymentByOrder", ctx, uint(1)).Return(mockPayment, nil)
		mockAddrRepo.On("GetByID", ctx, addrID).Return(mockAddr, nil)

		res, err := svc.GetPaymentOrderInfo(ctx, externalID)
		assert.NoError(t, err)
		assert.Equal(t, 50000, res.TotalAmount)
	})

	t.Run("PaymentNotFound", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockPayRepo := new(MockPaymentRepository)
		svc := NewService(mockRepo, mockPayRepo, nil, nil, nil)

		mockOrder := &Order{
			ID:     1,
			UserID: &userInt32,
		}
		mockRepo.On("GetOrderByExternalID", ctx, externalID).Return(mockOrder, nil)
		mockPayRepo.On("GetPaymentByOrder", ctx, uint(1)).Return(nil, errors.New("payment not found"))

		_, err := svc.GetPaymentOrderInfo(ctx, externalID)
		assert.Error(t, err)
	})
}

func TestService_UpdateSessionAddress_Forbidden(t *testing.T) {
	userID := uint(1)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "sess-ext-1"
	addrIDStr := uuid.New().String()

	mockRepo := new(MockRepository)
	svc := NewService(mockRepo, nil, nil, nil, nil)

	otherUser := int32(999)
	mockSession := &CheckoutSession{
		UserID: &otherUser,
		Status: CheckoutSessionStatusPending,
	}

	mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

	err := svc.UpdateSessionAddress(ctx, externalID, addrIDStr, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestService_ConfirmSession_EdgeCases(t *testing.T) {
	userID := uint(1)
	userInt32 := int32(userID)
	ctx := utils.SetUserContext(context.Background(), userID, "test@example.com", "user")
	externalID := "sess-ext-1"
	now := time.Now().Add(1 * time.Hour)

	t.Run("AddressNotSet", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: now,
			AddressID: nil, // Missing address
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shipping address not set")
	})

	t.Run("AlreadyConfirmed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID: &userInt32,
			Status: CheckoutSessionStatusPaid, // Not pending
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already confirmed")
	})

	t.Run("Forbidden_Ownership", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		otherUser := int32(999)
		mockSession := &CheckoutSession{UserID: &otherUser}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("NoItems", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)

		mockSession := &CheckoutSession{
			UserID:    &userInt32,
			Status:    CheckoutSessionStatusPending,
			ExpiresAt: now,
			AddressID: &uuid.UUID{},
			Items:     []CheckoutSessionItem{},
		}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checkout session has no items")
	})

	t.Run("RepoError_Confirm", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		sessID := uuid.New()
		addrID := uuid.New()
		mockSession := &CheckoutSession{ID: sessID, UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now, AddressID: &addrID, Items: []CheckoutSessionItem{{VariantID: "v1", Quantity: 1}}}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("ValidateVariantStock", ctx, "v1", 1).Return(true, nil)
		mockRepo.On("GetOrderBySessionID", ctx, sessID).Return(nil, nil)
		mockRepo.On("CreateOrderTx", ctx, mock.Anything, mock.Anything).Return(errors.New("tx error"))

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tx error")
	})

	t.Run("RepoError_GetSession", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(nil, errors.New("db error"))
		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
	})

	t.Run("RepoError_ValidateStock", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewService(mockRepo, nil, nil, nil, nil)
		addrID := uuid.New()
		mockSession := &CheckoutSession{UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now, AddressID: &addrID, Items: []CheckoutSessionItem{{VariantID: "v1", Quantity: 1}}}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("ValidateVariantStock", ctx, "v1", 1).Return(false, errors.New("stock error"))

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stock error")
	})

	t.Run("RepoError_ConfirmSession", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockPayGate := new(MockPaymentGateway)
		mockPayRepo := new(MockPaymentRepository)
		svc := NewService(mockRepo, mockPayRepo, mockPayGate, nil, nil)
		sessID := uuid.New()
		addrID := uuid.New()
		mockSession := &CheckoutSession{ID: sessID, UserID: &userInt32, Status: CheckoutSessionStatusPending, ExpiresAt: now, AddressID: &addrID, Items: []CheckoutSessionItem{{VariantID: "v1", Quantity: 1}}}

		mockRepo.On("GetCheckoutSession", ctx, externalID).Return(mockSession, nil)
		mockRepo.On("ValidateVariantStock", ctx, "v1", 1).Return(true, nil)
		mockRepo.On("GetOrderBySessionID", ctx, sessID).Return(nil, nil)
		mockRepo.On("CreateOrderTx", ctx, mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("ConfirmCheckoutSession", ctx, mockSession).Return(errors.New("confirm error"))

		_, err := svc.ConfirmSession(ctx, externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "confirm error")
	})
}

func TestService_OrderToPaymentProcess_GatewayError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockPayGate := new(MockPaymentGateway)
	svc := NewService(mockRepo, nil, mockPayGate, nil, nil)

	ctx := context.Background()
	orderExtID := "ord-ext-1"
	orderID := uint(1)

	pm := payment.MethodBCAVA
	mockSession := &CheckoutSession{
		TotalPrice:    10000,
		Items:         []CheckoutSessionItem{{ProductName: "P1", VariantName: "V1", Quantity: 1, Price: 10000}},
		PaymentMethod: &pm,
	}

	mockPayGate.On("CreateInvoice", orderExtID, "Guest", int64(10000), mock.Anything, mock.Anything, payment.ChannelCode(payment.MethodBCAVA)).Return(nil, errors.New("gateway error"))

	_, err := svc.OrderToPaymentProcess(ctx, mockSession, orderExtID, orderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create payment invoice")
}

func TestService_OrderToPaymentProcess_SavePaymentError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockPayRepo := new(MockPaymentRepository)
	mockPayGate := new(MockPaymentGateway)
	svc := NewService(mockRepo, mockPayRepo, mockPayGate, nil, nil)

	ctx := context.Background()
	orderExtID := "ord-ext-1"
	orderID := uint(1)

	pm := payment.MethodBCAVA
	mockSession := &CheckoutSession{
		TotalPrice:    10000,
		Items:         []CheckoutSessionItem{{ProductName: "P1", VariantName: "V1", Quantity: 1, Price: 10000}},
		PaymentMethod: &pm,
	}
	mockPayResp := &payment.PaymentResponse{ProviderPaymentID: "pay-1", Status: "PENDING"}

	mockPayGate.On("CreateInvoice", orderExtID, "Guest", int64(10000), mock.Anything, mock.Anything, payment.ChannelCode(payment.MethodBCAVA)).Return(mockPayResp, nil)
	mockPayRepo.On("SavePayment", ctx, mock.Anything).Return(errors.New("db error"))

	_, err := svc.OrderToPaymentProcess(ctx, mockSession, orderExtID, orderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save payment")
}

func TestService_GetPaymentOrderInfo_Forbidden(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo, nil, nil, nil, nil)
	ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

	otherUser := int32(999)
	mockOrder := &Order{UserID: &otherUser}
	mockRepo.On("GetOrderByExternalID", ctx, "ext-id").Return(mockOrder, nil)

	_, err := svc.GetPaymentOrderInfo(ctx, "ext-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestService_GetPaymentOrderInfo_AddressError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockPayRepo := new(MockPaymentRepository)
	mockAddrRepo := new(MockAddressRepository)
	svc := NewService(mockRepo, mockPayRepo, nil, mockAddrRepo, nil)
	ctx := utils.SetUserContext(context.Background(), 1, "test@example.com", "user")

	userID := int32(1)
	mockOrder := &Order{ID: 1, UserID: &userID, AddressID: uuid.New()}
	mockRepo.On("GetOrderByExternalID", ctx, "ext-id").Return(mockOrder, nil)
	mockPayRepo.On("GetPaymentByOrder", ctx, uint(1)).Return(&payment.Payment{}, nil)
	mockAddrRepo.On("GetByID", ctx, mockOrder.AddressID).Return(nil, errors.New("addr error"))

	_, err := svc.GetPaymentOrderInfo(ctx, "ext-id")
	assert.Error(t, err)
	assert.Equal(t, "addr error", err.Error())
}
