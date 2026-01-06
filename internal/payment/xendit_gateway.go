package payment

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"warimas-be/internal/logger"

	"go.uber.org/zap"
)

type xenditGateway struct {
	apiKey     string
	httpClient *http.Client
}

type XenditPaymentResponse struct {
	ID            string  `json:"payment_request_id"`
	ReferenceID   string  `json:"reference_id"`
	Amount        float64 `json:"request_amount"`
	ChannelCode   string  `json:"channel_code"`
	Status        string  `json:"status"`
	PaymentMethod string  `json:"type"`
	ChannelProps  struct {
		DisplayName string `json:"display_name"`
		ExpiresAt   string `json:"expires_at"`
	} `json:"channel_properties"`
	Actions []struct {
		Type       string `json:"type"`
		Descriptor string `json:"descriptor"`
		Value      string `json:"value"`
	} `json:"actions"`
}

// ----------------- Constructor -----------------

func NewXenditGateway(apiKey string) Gateway {
	if apiKey == "" {
		logger.L().Warn("Xendit API key is empty")
	}

	return &xenditGateway{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ----------------- CreateInvoice -----------------

func (x *xenditGateway) CreateInvoice(
	orderID uint,
	buyerName string,
	amount uint,
	customerEmail string,
	items []OrderItem,
	channelCode ChannelCode,
) (*PaymentResponse, error) {

	log := logger.L().With(
		zap.Uint("order_id", orderID),
		zap.String("buyer", buyerName),
		zap.Uint("amount", amount),
		zap.String("channel", string(channelCode)),
	)

	expiry := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	body := map[string]interface{}{
		"reference_id":   fmt.Sprintf("%d", orderID),
		"type":           "PAY",
		"country":        "ID",
		"currency":       "IDR",
		"request_amount": amount,
		"customer": map[string]interface{}{
			"type":         "INDIVIDUAL",
			"reference_id": fmt.Sprintf("%d", orderID),
			"email":        customerEmail,
			"individual_detail": map[string]interface{}{
				"given_names": buyerName,
			},
		},
		"metadata": map[string]interface{}{
			"items": items,
		},
		"channel_code": string(channelCode),
		"channel_properties": map[string]interface{}{
			"expires_at":   expiry,
			"payer_name":   buyerName,
			"display_name": buyerName,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Error("Failed to marshal payment request", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.xendit.co/v3/payment_requests", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Failed creating request", zap.Error(err))
		return nil, err
	}

	authString := base64.StdEncoding.EncodeToString([]byte(x.apiKey + ":"))
	req.Header.Add("Authorization", "Basic "+authString)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-version", "2024-11-11")

	log.Info("Sending payment request to Xendit")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		log.Error("Xendit request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Error("Xendit returned non-success status",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("response", bodyBytes),
		)
		return nil, fmt.Errorf("xendit error: %s", string(bodyBytes))
	}

	var res XenditPaymentResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		log.Error("Failed decoding Xendit response", zap.Error(err))
		return nil, err
	}

	log.Info("Xendit payment created",
		zap.String("payment_id", res.ID),
		zap.String("reference_id", res.ReferenceID),
		zap.String("status", res.Status),
	)

	// Prevent panic if Actions is empty
	var paymentCode string
	if len(res.Actions) > 0 {
		paymentCode = res.Actions[0].Value
	}

	return &PaymentResponse{
		ExternalID:     res.ID,
		Amount:         res.Amount,
		Status:         res.Status,
		PaymentMethod:  res.PaymentMethod,
		PaymentCode:    paymentCode,
		ChannelCode:    res.ChannelCode,
		ExpirationTime: res.ChannelProps.ExpiresAt,
	}, nil
}

// ----------------- GetPaymentStatus -----------------

func (x *xenditGateway) GetPaymentStatus(externalID string) (*PaymentStatus, error) {
	log := logger.L().With(zap.String("external_id", externalID))

	url := fmt.Sprintf("https://api.xendit.co/v2/invoices?external_id=%s", externalID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("Failed building request", zap.Error(err))
		return nil, err
	}

	req.SetBasicAuth(x.apiKey, "")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		log.Error("Request to Xendit failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error("Xendit returned error",
			zap.Int("http_status", resp.StatusCode),
			zap.ByteString("response", bodyBytes),
		)
		return nil, fmt.Errorf("xendit error: %s", string(bodyBytes))
	}

	var invoices []struct {
		Status string     `json:"status"`
		PaidAt *time.Time `json:"paid_at"`
	}
	if err := json.Unmarshal(bodyBytes, &invoices); err != nil {
		log.Error("Failed decoding invoice", zap.Error(err))
		return nil, err
	}

	if len(invoices) == 0 {
		log.Warn("Invoice not found")
		return nil, errors.New("invoice not found")
	}

	return &PaymentStatus{
		Status: invoices[0].Status,
		PaidAt: invoices[0].PaidAt,
	}, nil
}

// ----------------- Cancel Payment -----------------

func (x *xenditGateway) CancelPayment(externalID string) error {
	log := logger.L().With(zap.String("external_id", externalID))

	url := fmt.Sprintf("https://api.xendit.co/invoices/%s/expire!", externalID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Error("Failed creating request", zap.Error(err))
		return err
	}

	req.SetBasicAuth(x.apiKey, "")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		log.Error("Xendit request failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to cancel payment",
			zap.Int("http_status", resp.StatusCode),
			zap.ByteString("response", bodyBytes),
		)
		return fmt.Errorf("xendit cancel error: %s", string(bodyBytes))
	}

	log.Info("Payment cancelled successfully")
	return nil
}

// ----------------- Verify Signature -----------------

func (x *xenditGateway) VerifySignature(r *http.Request) error {
	sig := r.Header.Get("x-callback-token")
	expected := os.Getenv("XENDIT_CALLBACK_TOKEN")

	if expected == "" {
		return nil // skip in dev
	}

	if sig != expected {
		return errors.New("invalid webhook signature")
	}
	return nil
}
