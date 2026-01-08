package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"warimas-be/internal/logger"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"

	"go.uber.org/zap"
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
			Items []payment.XenditItem `json:"items"`
		} `json:"metadata"`
	} `json:"data"`
}

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

// -------------------------- MAIN WEBHOOK HANDLER --------------------------
func (h *Handler) PaymentWebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromCtx(ctx)

	callbackToken := r.Header.Get("x-callback-token")
	expectedToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")

	if callbackToken != expectedToken {
		log.Warn("Webhook token mismatch")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed reading webhook body", zap.Error(err))
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	raw := json.RawMessage(body)

	log.Info("raw body WebhookPayload",
		zap.Any("raw", raw),
	)

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("Failed to parse webhook JSON", zap.Error(err))
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	// Core log for tracking payment events
	log.Info("Webhook received",
		zap.String("event", payload.Event),
		zap.String("status", payload.Data.Status),
		zap.String("payment_id", payload.Data.PaymentID),
		zap.String("reference_id", payload.Data.ReferenceID),
	)

	// Process event
	h.handleEvent(ctx, payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received"))
}

// -------------------------- EVENT HANDLER --------------------------
func (h *Handler) handleEvent(ctx context.Context, payload WebhookPayload) {
	log := logger.FromCtx(ctx)

	event := payload.Event
	ref := payload.Data.ReferenceID

	log = log.With(
		zap.String("event", event),
		zap.String("reference_id", ref),
		zap.String("status", payload.Data.Status),
	)

	switch event {

	case "payment.capture":
		log.Info("Processing capture event")

		if payload.Data.Status == "SUCCEEDED" {
			if err := h.OrderSvc.MarkAsPaid(ctx, ref, payload.Data.PaymentRequestID); err != nil {
				log.Error("Failed to mark order as PAID", zap.Error(err))
				return
			}
			log.Info("Order marked as PAID")
		}

	case "payment.authorization":
		log.Info("Payment authorized")

	case "payment.failed", "payment.failure":
		log.Warn("Payment failed")

		if err := h.OrderSvc.MarkAsFailed(ctx, ref, payload.Data.PaymentRequestID); err != nil {
			log.Error("Failed to mark order as FAILED", zap.Error(err))
			return
		}
		log.Info("Order marked as FAILED")

	default:
		log.Warn("Unhandled webhook event")
	}
}
