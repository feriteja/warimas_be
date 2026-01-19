package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_PaymentWebhookHandler(t *testing.T) {
	// Setup Env for Token Verification
	t.Setenv("XENDIT_WEBHOOK_TOKEN", "secret-token")
	validHeader := "secret-token"

	t.Run("Success_Paid", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"payment_id":         "pay-id-1",
				"payment_request_id": "pay-req-1",
				"reference_id":       "ord-ref-1",
				"status":             "SUCCEEDED",
				"request_amount":     100000,
				"currency":           "IDR",
				"created":            "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		// 1. Save Webhook (Not Duplicate)
		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.capture", "ord-ref-1", mock.Anything, true).
			Return(int64(1), false, nil)

		// 2. Get Order Info for Validation
		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PENDING",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		// 3. Mark as Paid
		mockOrderSvc.On("MarkAsPaid", mock.Anything, "ord-ref-1", "pay-req-1", "pay-id-1").Return(nil)

		// 4. Mark Processed
		mockPayRepo.On("MarkWebhookProcessed", mock.Anything, int64(1)).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockOrderSvc.AssertExpectations(t)
		mockPayRepo.AssertExpectations(t)
	})

	t.Run("Success_Failed", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.failed",
			"data": map[string]interface{}{
				"payment_id":         "pay-id-1",
				"payment_request_id": "pay-req-1",
				"reference_id":       "ord-ref-1",
				"status":             "FAILED",
				"request_amount":     100000,
				"currency":           "IDR",
				"created":            "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.failed", "ord-ref-1", mock.Anything, true).
			Return(int64(2), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PENDING",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		mockOrderSvc.On("MarkAsFailed", mock.Anything, "ord-ref-1", "pay-req-1", "pay-id-1").Return(nil)
		mockPayRepo.On("MarkWebhookProcessed", mock.Anything, int64(2)).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Duplicate_Webhook", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"payment_id":   "pay-id-1",
				"reference_id": "ord-ref-1",
				"created":      "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.capture", "ord-ref-1", mock.Anything, true).
			Return(int64(0), true, nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockOrderSvc.AssertNotCalled(t, "MarkAsPaid")
	})

	t.Run("Unauthorized_Token", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		req := httptest.NewRequest("POST", "/webhook/xendit", nil)
		req.Header.Set("x-callback-token", "invalid-token")
		w := httptest.NewRecorder()

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Amount_Mismatch", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id":   "ord-ref-1",
				"request_amount": 50000, // Mismatch
				"currency":       "IDR",
				"created":        "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(3), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000, // Expected
			Currency:    "IDR",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		mockPayRepo.On("MarkWebhookFailed", mock.Anything, int64(3), mock.MatchedBy(func(reason string) bool {
			return reason == "amount mismatch: webhook=50000 db=100000"
		})).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid_Transition_PaidToFailed", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.failed",
			"data": map[string]interface{}{
				"reference_id":   "ord-ref-1",
				"request_amount": 100000,
				"currency":       "IDR",
				"created":        "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(4), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PAID", // Already Paid
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		mockPayRepo.On("MarkWebhookFailed", mock.Anything, int64(4), "invalid transition PAID -> FAILED").Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Processing_Error", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"payment_id":         "pay-id-1",
				"payment_request_id": "pay-req-1",
				"reference_id":       "ord-ref-1",
				"status":             "SUCCEEDED",
				"request_amount":     100000,
				"currency":           "IDR",
				"created":            "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.capture", "ord-ref-1", mock.Anything, true).
			Return(int64(3), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PENDING",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		mockOrderSvc.On("MarkAsPaid", mock.Anything, "ord-ref-1", "pay-req-1", "pay-id-1").Return(errors.New("db error"))

		mockPayRepo.On("MarkWebhookFailed", mock.Anything, int64(3), "db error").Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unhandled_Event", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.created",
			"data": map[string]interface{}{
				"payment_id":         "pay-id-1",
				"payment_request_id": "pay-req-1",
				"reference_id":       "ord-ref-1",
				"request_amount":     100000,
				"currency":           "IDR",
				"created":            "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.created", "ord-ref-1", mock.Anything, true).
			Return(int64(5), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PENDING",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)
		mockPayRepo.On("MarkWebhookProcessed", mock.Anything, int64(5)).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid_JSON", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBufferString("{invalid-json"))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Save_Webhook_Error", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id": "ord-ref-1",
				"payment_id":   "pay-id-1",
				"created":      "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, "XENDIT", mock.Anything, "payment.capture", "ord-ref-1", mock.Anything, true).
			Return(int64(0), false, errors.New("db error"))

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Get_Order_Info_Error", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id": "ord-ref-1",
				"payment_id":   "pay-id-1",
				"created":      "2024-01-01T10:00:00Z",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(10), false, nil)

		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(nil, errors.New("order not found"))

		mockPayRepo.On("MarkWebhookFailed", mock.Anything, int64(10), "order not found").Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Currency_Mismatch", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id":   "ord-ref-1",
				"payment_id":     "pay-id-1",
				"created":        "2024-01-01T10:00:00Z",
				"request_amount": 100000,
				"currency":       "USD",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(11), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		mockPayRepo.On("MarkWebhookFailed", mock.Anything, int64(11), "currency mismatch").Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Payment_Capture_Not_Succeeded", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id":   "ord-ref-1",
				"payment_id":     "pay-id-1",
				"created":        "2024-01-01T10:00:00Z",
				"request_amount": 100000,
				"currency":       "IDR",
				"status":         "PENDING",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(12), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PENDING",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		// Should NOT call MarkAsPaid
		mockPayRepo.On("MarkWebhookProcessed", mock.Anything, int64(12)).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockOrderSvc.AssertNotCalled(t, "MarkAsPaid")
	})

	t.Run("Order_Already_Paid", func(t *testing.T) {
		mockOrderSvc := new(MockOrderService)
		mockPayRepo := new(MockPaymentRepository)
		mockGateway := new(MockGateway)
		h := NewWebhookHandler(mockOrderSvc, mockGateway, mockPayRepo)

		payload := map[string]interface{}{
			"event": "payment.capture",
			"data": map[string]interface{}{
				"reference_id":   "ord-ref-1",
				"payment_id":     "pay-id-1",
				"created":        "2024-01-01T10:00:00Z",
				"request_amount": 100000,
				"currency":       "IDR",
				"status":         "SUCCEEDED",
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/webhook/xendit", bytes.NewBuffer(body))
		req.Header.Set("x-callback-token", validHeader)
		w := httptest.NewRecorder()

		mockPayRepo.On("SavePaymentWebhook", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, true).
			Return(int64(13), false, nil)

		mockOrderInfo := &order.Order{
			TotalAmount: 100000,
			Currency:    "IDR",
			Status:      "PAID",
		}
		mockOrderSvc.On("GetOrderForWebhook", mock.Anything, "ord-ref-1").Return(mockOrderInfo, nil)

		// Should NOT call MarkAsPaid
		mockPayRepo.On("MarkWebhookProcessed", mock.Anything, int64(13)).Return(nil)

		h.PaymentWebhookHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockOrderSvc.AssertNotCalled(t, "MarkAsPaid")
	})
}

// --- Mocks ---

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) MarkAsPaid(ctx context.Context, refID, payReqID, provID string) error {
	args := m.Called(ctx, refID, payReqID, provID)
	return args.Error(0)
}

func (m *MockOrderService) MarkAsFailed(ctx context.Context, refID, payReqID, provID string) error {
	args := m.Called(ctx, refID, payReqID, provID)
	return args.Error(0)
}

// Stubs to satisfy order.Service interface
func (m *MockOrderService) CreateFromSession(ctx context.Context, externalID string) (*order.Order, error) {
	return nil, nil
}
func (m *MockOrderService) OrderToPaymentProcess(ctx context.Context, session *order.CheckoutSession, externalID string, orderId uint) (*payment.PaymentResponse, error) {
	return nil, nil
}
func (m *MockOrderService) GetOrders(ctx context.Context, filter *order.OrderFilterInput, sort *order.OrderSortInput, limit int32, page int32) ([]*order.Order, int64, map[uuid.UUID][]address.Address, error) {
	return nil, 0, nil, nil
}
func (m *MockOrderService) GetOrderDetail(ctx context.Context, orderID uint) (*order.Order, *address.Address, error) {
	return nil, nil, nil
}
func (m *MockOrderService) GetOrderDetailByExternalID(ctx context.Context, externalId string) (*order.Order, *address.Address, error) {
	return nil, nil, nil
}
func (m *MockOrderService) UpdateOrderStatus(ctx context.Context, orderID uint, status order.OrderStatus) error {
	return nil
}
func (m *MockOrderService) CreateSession(ctx context.Context, input model.CreateCheckoutSessionInput) (*order.CheckoutSession, error) {
	return nil, nil
}
func (m *MockOrderService) UpdateSessionAddress(ctx context.Context, externalID string, addressID string, guestID *string) error {
	return nil
}
func (m *MockOrderService) UpdateSessionPaymentMethod(ctx context.Context, externalID string, paymentMethod payment.ChannelCode, guestID *string) error {
	return nil
}
func (m *MockOrderService) ConfirmSession(ctx context.Context, sessionID string) (*string, error) {
	return nil, nil
}
func (m *MockOrderService) GetSession(ctx context.Context, externalID string) (*order.CheckoutSession, error) {
	return nil, nil
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

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) SavePaymentWebhook(ctx context.Context, provider, eventID, eventType, externalID string, payload json.RawMessage, valid bool) (int64, bool, error) {
	args := m.Called(ctx, provider, eventID, eventType, externalID, payload, valid)
	return args.Get(0).(int64), args.Bool(1), args.Error(2)
}

func (m *MockPaymentRepository) MarkWebhookProcessed(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPaymentRepository) MarkWebhookFailed(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

// Stubs
func (m *MockPaymentRepository) SavePayment(ctx context.Context, p *payment.Payment) error {
	return nil
}
func (m *MockPaymentRepository) UpdatePaymentStatus(ctx context.Context, eid, status string) error {
	return nil
}
func (m *MockPaymentRepository) GetPaymentByOrder(ctx context.Context, oid uint) (*payment.Payment, error) {
	return nil, nil
}

type MockGateway struct {
	mock.Mock
}

func (m *MockGateway) VerifySignature(r *http.Request) error {
	args := m.Called(r)
	return args.Error(0)
}

// Stubs
func (m *MockGateway) CreateInvoice(extID, email string, amt int64, payer string, items []payment.XenditItem, ch payment.ChannelCode) (*payment.PaymentResponse, error) {
	return nil, nil
}
func (m *MockGateway) GetPaymentStatus(extID string) (*payment.PaymentStatus, error) {
	return nil, nil
}
func (m *MockGateway) CancelPayment(extID string) error {
	args := m.Called(extID)
	return args.Error(0)
}
