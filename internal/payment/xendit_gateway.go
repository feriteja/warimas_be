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

// Constructor
func NewXenditGateway(apiKey string) Gateway {
	xenditApiKey := os.Getenv("XENDIT_APIKEY")
	if xenditApiKey == "" {
		logger.Error("Xendit APIKEY is missing")
	}

	return &xenditGateway{
		apiKey: xenditApiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// --- CreateInvoice --------------------------------------------------

func (x *xenditGateway) CreateInvoice(orderID uint,
	buyerName string,
	amount float64,
	customerEmail string,
	items []OrderItem,
	channelCode ChannelCode,
) (*PaymentResponse, error) {
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
				"given_names": "buyerName",
			},
		},
		"metadata": map[string]interface{}{
			"items": items,
		},
		"channel_code": string(channelCode),
		"channel_properties": map[string]interface{}{
			"expires_at":   expiry,
			"payer_name":   "buyerName",
			"display_name": "buyerName",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.xendit.co/v3/payment_requests", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	authString := base64.StdEncoding.EncodeToString([]byte(x.apiKey + ":"))
	req.Header.Add("Authorization", "Basic "+authString)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-version", "2024-11-11")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		logger.Error("payment failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("body: %v", resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xendit: create invoice failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res XenditPaymentResponse

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode xendit response: %w", err)
	}

	return &PaymentResponse{
		ExternalID:     res.ID,
		Amount:         res.Amount,
		Status:         res.Status,
		PaymentMethod:  res.PaymentMethod,
		PaymentCode:    res.Actions[0].Value,
		ChannelCode:    res.ChannelCode,
		ExpirationTime: res.ChannelProps.ExpiresAt,
	}, nil
}

// --- GetPaymentStatus -----------------------------------------------

func (x *xenditGateway) GetPaymentStatus(externalID string) (*PaymentStatus, error) {
	url := fmt.Sprintf("https://api.xendit.co/v2/invoices?external_id=%s", externalID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(x.apiKey, "")
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xendit: get status failed (%d): %s", resp.StatusCode, string(body))
	}

	var invoices []struct {
		Status string     `json:"status"`
		PaidAt *time.Time `json:"paid_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invoices); err != nil {
		return nil, err
	}

	if len(invoices) == 0 {
		return nil, errors.New("invoice not found")
	}

	return &PaymentStatus{
		Status: invoices[0].Status,
		PaidAt: invoices[0].PaidAt,
	}, nil
}

// --- CancelPayment --------------------------------------------------

func (x *xenditGateway) CancelPayment(externalID string) error {
	url := fmt.Sprintf("https://api.xendit.co/invoices/%s/expire!", externalID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(x.apiKey, "")
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("xendit: cancel payment failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// --- VerifySignature ------------------------------------------------

// For webhook verification using "x-callback-token" header
func (x *xenditGateway) VerifySignature(r *http.Request) error {
	signature := r.Header.Get("x-callback-token")
	expected := os.Getenv("XENDIT_CALLBACK_TOKEN")

	if expected == "" {
		return nil // Skip if not configured (e.g., local dev)
	}

	if signature != expected {
		return errors.New("invalid webhook signature")
	}
	return nil
}
