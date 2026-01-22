package payment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"context"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"

	"go.uber.org/zap"
)

const (
	xenditBaseURL = "https://api.xendit.co"
	apiVersion    = "2024-11-11"
)

type xenditGateway struct {
	apiKey        string
	httpClient    *http.Client
	jakartaLoc    *time.Location
	failureURL    string
	successURL    string
	cancelURL     string
	callbackToken string
}

// ----------------- Constructor -----------------

func NewXenditGateway(apiKey string) Gateway {
	if apiKey == "" {
		logger.L().Warn("Xendit API key is empty")
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		logger.L().Error("failed to load Jakarta location, defaulting to UTC", zap.Error(err))
		loc = time.UTC
	}

	return &xenditGateway{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		jakartaLoc:    loc,
		failureURL:    os.Getenv("FAILURE_URL"),
		successURL:    os.Getenv("SUCCESS_URL"),
		cancelURL:     os.Getenv("CANCEL_RETURN_URL"),
		callbackToken: os.Getenv("XENDIT_CALLBACK_TOKEN"),
	}
}

// ----------------- CreateInvoice -----------------

func (x *xenditGateway) CreateInvoice(
	ctx context.Context,
	externalID string,
	buyer BuyerInfo,
	amount int64,
	items []XenditItem,
	channelCode ChannelCode,
) (*PaymentResponse, error) {

	log := logger.L().With(
		zap.String("order_id", externalID),
		zap.String("buyer", buyer.Name),
		zap.Int64("amount", amount),
		zap.String("channel", string(channelCode)),
		zap.String("phone", buyer.Phone),
	)

	phone := utils.NormalizePhoneID(buyer.Phone)

	expiry := time.Now().In(x.jakartaLoc).Add(24 * time.Hour).Format(time.RFC3339)

	body := map[string]interface{}{
		"reference_id":   externalID,
		"type":           "PAY",
		"country":        "ID",
		"currency":       "IDR",
		"request_amount": amount,
		"customer": map[string]interface{}{
			"type":         "INDIVIDUAL",
			"reference_id": externalID,
			"email":        buyer.Email,
			"individual_detail": map[string]interface{}{
				"given_names": buyer.Name,
			},
		},
		"metadata": map[string]interface{}{
			"items": items,
		},
		"channel_code": string(channelCode),
		"channel_properties": map[string]interface{}{
			"failure_return_url":    x.failureURL,
			"success_return_url":    x.successURL,
			"cancel_return_url":     x.cancelURL,
			"expires_at":            expiry,
			"payer_name":            buyer.Name,
			"display_name":          buyer.Name,
			"account_mobile_number": phone,
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Error("Failed to marshal payment request", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", xenditBaseURL+"/v3/payment_requests", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Failed creating request", zap.Error(err))
		return nil, err
	}

	req.SetBasicAuth(x.apiKey, "")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-version", apiVersion)

	log.Info("Sending payment request to Xendit")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		log.Error("Xendit request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.Error(err))
		return nil, fmt.Errorf("failed to read xendit response: %w", err)
	}

	raw := json.RawMessage(bodyBytes)

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
		zap.String("payment_id", res.PaymentRequestID),
		zap.String("reference_id", res.ReferenceID),
		zap.String("status", res.Status),
	)

	var paymentCode string
	var invoiceURL string // For redirects and deeplinks

	// Extract relevant data from actions
	for _, action := range res.Actions {
		switch action.Descriptor {
		// These are codes to be displayed to the user
		case "VIRTUAL_ACCOUNT_NUMBER", "PAYMENT_CODE", "QR_STRING":
			if paymentCode == "" { // Take the first code-like action
				paymentCode = action.Value
			}
		// These are URLs for redirection
		case "WEB_URL", "DEEPLINK_URL":
			if invoiceURL == "" { // Take the first URL-like action
				invoiceURL = action.Value
			}
		}
	}

	// Use the expiration time returned by Xendit, or default if nil
	var expirationTime time.Time
	if res.ChannelProperties.ExpiresAt != nil {
		expirationTime = *res.ChannelProperties.ExpiresAt
	} else {
		// Fallback if Xendit doesn't return it (unlikely for created invoice)
		expirationTime = time.Now().Add(24 * time.Hour)
	}

	return &PaymentResponse{
		ProviderPaymentID: res.PaymentRequestID,
		ReferenceID:       res.ReferenceID,
		Amount:            res.RequestAmount,
		Status:            res.Status,
		PaymentMethod:     ChannelCode(res.ChannelCode),
		PaymentCode:       paymentCode,
		InvoiceURL:        invoiceURL,
		ChannelCode:       res.ChannelCode,
		ExpirationTime:    expirationTime,
		RawResponse:       &raw,
	}, nil
}

// ----------------- GetPaymentStatus -----------------

func (x *xenditGateway) GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatus, error) {
	log := logger.L().With(zap.String("external_id", externalID))

	url := fmt.Sprintf("%s/v2/invoices?external_id=%s", xenditBaseURL, externalID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.Error(err))
		return nil, fmt.Errorf("failed to read xendit response: %w", err)
	}

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

func (x *xenditGateway) CancelPayment(ctx context.Context, externalID string) error {
	log := logger.L().With(zap.String("external_id", externalID))

	url := fmt.Sprintf("%s/invoices/%s/expire!", xenditBaseURL, externalID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.Error(err))
		return fmt.Errorf("failed to read xendit response: %w", err)
	}

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
	expected := x.callbackToken

	if expected == "" {
		return nil // skip in dev
	}

	if sig != expected {
		return errors.New("invalid webhook signature")
	}
	return nil
}
