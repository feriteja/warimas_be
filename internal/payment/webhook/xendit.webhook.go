package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"warimas-be/internal/logger"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"

	"go.uber.org/zap"
)

type Handler struct {
	OrderSvc    order.Service
	Gateway     payment.Gateway
	PaymentRepo payment.Repository
}

func NewWebhookHandler(orderSvc order.Service, gateway payment.Gateway, paymentRepo payment.Repository) *Handler {
	return &Handler{
		OrderSvc:    orderSvc,
		Gateway:     gateway,
		PaymentRepo: paymentRepo,
	}
}

func (h *Handler) PaymentWebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromCtx(ctx)

	// 1. Verify callback token
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken != os.Getenv("XENDIT_WEBHOOK_TOKEN") {
		log.Warn("Invalid webhook token")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed reading webhook body", zap.Error(err))
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	rawPayload := json.RawMessage(body)

	// 3. Parse payload
	var payload payment.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("Invalid webhook JSON", zap.Error(err))
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	jsonPayload, err := json.Marshal(payload)

	log.Info("jsonpayload", zap.Any("payload", json.RawMessage(jsonPayload)))

	// 4. Derive event ID (Xendit sometimes lacks one)
	eventID := payload.Event + ":" + payload.Data.PaymentID + ":" + payload.Data.Created

	// 5. Save webhook FIRST (idempotency happens here)
	webhookID, isDuplicate, err := h.PaymentRepo.SavePaymentWebhook(
		ctx,
		"XENDIT",
		eventID,
		payload.Event,
		payload.Data.ReferenceID,
		rawPayload,
		true,
	)
	if err != nil {
		log.Error("Failed saving webhook", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if isDuplicate {
		log.Info("Duplicate webhook ignored", zap.String("event_id", eventID))
		w.WriteHeader(http.StatusOK)
		return
	}

	// 6. Process webhook safely
	if err := h.processPaymentEvent(ctx, payload); err != nil {
		log.Error("Webhook processing failed", zap.Error(err))

		_ = h.PaymentRepo.MarkWebhookFailed(ctx, webhookID, err.Error())
		http.Error(w, "processing failed", http.StatusBadRequest)
		return
	}

	// 7. Mark webhook processed
	_ = h.PaymentRepo.MarkWebhookProcessed(ctx, webhookID)

	log.Info("Webhook processed successfully",
		zap.String("event", payload.Event),
		zap.String("reference_id", payload.Data.ReferenceID),
	)

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) processPaymentEvent(
	ctx context.Context,
	payload payment.WebhookPayload,
) error {

	log := logger.FromCtx(ctx)

	ref := payload.Data.ReferenceID

	log.Info("processing payment webhook",
		zap.String("event", payload.Event),
		zap.String("reference_id", ref),
		zap.String("payment_request_id", payload.Data.PaymentRequestID),
		zap.Int64("amount", payload.Data.RequestAmount),
		zap.String("currency", payload.Data.Currency),
		zap.String("status", payload.Data.Status),
	)

	// Lock payment/order row
	order, err := h.OrderSvc.GetOrderForWebhook(ctx, ref)
	if err != nil {
		log.Error("failed to fetch order payment info",
			zap.String("reference_id", ref),
			zap.Error(err),
		)
		return err
	}

	// Validate money
	if payload.Data.RequestAmount != int64(order.TotalAmount) {
		log.Error("payment amount mismatch",
			zap.String("reference_id", ref),
			zap.Int64("webhook_amount", payload.Data.RequestAmount),
			zap.Uint("db_amount", order.TotalAmount),
		)
		return fmt.Errorf(
			"amount mismatch: webhook=%d db=%d",
			payload.Data.RequestAmount,
			order.TotalAmount,
		)
	}

	if payload.Data.Currency != order.Currency {
		log.Error("payment currency mismatch",
			zap.String("reference_id", ref),
			zap.String("webhook_currency", payload.Data.Currency),
			zap.String("db_currency", order.Currency),
		)
		return fmt.Errorf("currency mismatch")
	}

	switch payload.Event {

	case "payment.capture":
		if payload.Data.Status != "SUCCEEDED" {
			log.Info("payment capture not succeeded, ignoring",
				zap.String("reference_id", ref),
				zap.String("status", payload.Data.Status),
			)
			return nil
		}

		if order.Status == "PAID" {
			log.Info("order already paid, skipping",
				zap.String("reference_id", ref),
				zap.String("OrderExternalID", order.ExternalID),
			)
			return nil
		}

		log.Info("marking order as PAID",
			zap.String("reference_id", ref),
			zap.String("order_id", order.ExternalID),
		)

		return h.OrderSvc.MarkAsPaid(
			ctx,
			ref,
			payload.Data.PaymentRequestID,
			payload.Data.PaymentID,
		)

	case "payment.failed", "payment.failure":
		if order.Status == "PAID" {
			log.Error("invalid payment state transition PAID -> FAILED",
				zap.String("reference_id", ref),
				zap.String("order_id", order.ExternalID),
			)
			return fmt.Errorf("invalid transition PAID -> FAILED")
		}

		log.Info("marking order as FAILED",
			zap.String("reference_id", ref),
			zap.String("order_id", order.ExternalID),
		)

		return h.OrderSvc.MarkAsFailed(
			ctx,
			ref,
			payload.Data.PaymentRequestID,
			payload.Data.PaymentID,
		)

	default:
		log.Warn("unhandled payment webhook event",
			zap.String("event", payload.Event),
			zap.String("reference_id", ref),
		)
	}

	return nil
}
