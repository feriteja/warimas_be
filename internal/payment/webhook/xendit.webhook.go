package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"warimas-be/internal/logger"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"
)

type WebhookPayload struct {
	Created    string `json:"created"`
	BusinessID string `json:"business_id"`
	Event      string `json:"event"`
	APIVersion string `json:"api_version"`
	Data       struct {
		Type             string `json:"type"`
		Status           string `json:"status"`
		Country          string `json:"country"`
		Created          string `json:"created"`
		Updated          string `json:"updated"`
		Currency         string `json:"currency"`
		PaymentID        string `json:"payment_id"`
		BusinessID       string `json:"business_id"`
		CustomerID       string `json:"customer_id"`
		ChannelCode      string `json:"channel_code"`
		ReferenceID      string `json:"reference_id"`
		CaptureMethod    string `json:"capture_method"`
		RequestAmount    int64  `json:"request_amount"`
		PaymentRequestID string `json:"payment_request_id"`

		Captures []struct {
			CaptureID        string  `json:"capture_id"`
			CaptureAmount    float64 `json:"capture_amount"`
			CaptureTimestamp string  `json:"capture_timestamp"`
		} `json:"captures"`

		Metadata struct {
			Items []struct {
				Quantity  int `json:"Quantity"`
				ProductID int `json:"ProductID"`
			} `json:"items"`
		} `json:"metadata"`
	} `json:"data"`
}

// Handler depends on your order service and payment gateway
type Handler struct {
	OrderSvc order.Service
	Gateway  payment.Gateway
}

func NewWebhookHandler(orderSvc order.Service, gateway payment.Gateway) *Handler {
	return &Handler{
		OrderSvc: orderSvc,
		Gateway:  gateway,
	}
}

// WebhookHandler is the actual route handler
func (h *Handler) PaymentWebhookHandler(w http.ResponseWriter, r *http.Request) {
	callbackToken := r.Header.Get("x-callback-token")
	expectedToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")

	if callbackToken != expectedToken {
		logger.Error("XENDIT_WEBHOOK_TOKEN doesn't match")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Body request invalid")
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		logger.Error("JSON unmarshal error", err)
		return
	}

	// logger.Infof("✅ Webhook received: event=%s status=%s ref=%s",
	// 	payload.Event,
	// 	payload.Data.Status,
	// 	payload.Data.ReferenceID,
	// )

	h.handleEvent(payload)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Webhook received successfully")
}

func (h *Handler) handleEvent(payload WebhookPayload) {
	event := payload.Event
	ref := payload.Data.ReferenceID

	switch event {
	case "payment.capture":
		fmt.Println("masuk")
		if payload.Data.Status == "SUCCEEDED" {
			if err := h.OrderSvc.MarkAsPaid(ref); err != nil {
				logger.Error("❌ Failed to mark order as paid", map[string]interface{}{
					"ref":   ref,
					"error": err.Error(),
				})
				return
			}
			// logger.Infof("✅ Order %s marked as PAID", ref)
		}

	case "payment.authorization":
		// logger.Infof("⚙️ Payment authorized for ref %s", ref)

	case "payment.failed", "payment.failure":
		if err := h.OrderSvc.MarkAsFailed(ref); err != nil {
			// logger.Errorf("❌ Failed to mark order as FAILED (%s): %v", ref, err)
			return
		}
		// logger.Infof("❌ Payment failed for order %s", ref)

	default:
		// logger.Infof("Unhandled event type: %s for ref %s", event, ref)
	}
}
