package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"warimas-be/internal/order"
	"warimas-be/internal/payment"
)

// WebhookPayload represents the JSON Xendit sends
type WebhookPayload struct {
	ID         string  `json:"id"`
	ExternalID string  `json:"external_id"`
	Status     string  `json:"status"`
	Amount     float64 `json:"amount"`
	PaidAt     string  `json:"paid_at,omitempty"`
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
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Step 1️⃣ – Verify signature for security
	if err := h.Gateway.VerifySignature(r); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Step 2️⃣ – Parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("✅ Webhook received: %+v\n", payload)

	// Step 3️⃣ – Match webhook status to your order status
	switch payload.Status {
	case "PAID":
		err = h.OrderSvc.MarkOrderAsPaid(payload.ExternalID)
	case "EXPIRED", "FAILED":
		err = h.OrderSvc.MarkOrderAsFailed(payload.ExternalID)
	default:
		// Ignore other statuses
		w.WriteHeader(http.StatusOK)
		return
	}

	// Step 4️⃣ – Handle update result
	if err != nil {
		fmt.Println("❌ Failed to update order:", err)
		http.Error(w, "failed to update order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}
