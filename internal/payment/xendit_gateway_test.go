package payment

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockRoundTripper allows us to mock the HTTP response
type MockRoundTripper func(req *http.Request) *http.Response

func (f MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

type MockRoundTripperWithError func(req *http.Request) (*http.Response, error)

func (f MockRoundTripperWithError) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestXenditGateway_CreateInvoice(t *testing.T) {
	apiKey := "test-secret"
	gw := NewXenditGateway(apiKey).(*xenditGateway)

	externalID := "ord-123"
	amount := int64(100000)
	email := "test@example.com"
	buyer := BuyerInfo{
		Name:  "Buyer",
		Email: &email,
		Phone: "08123456789",
	}
	items := []XenditItem{{Name: "Item 1", Price: 100000, Quantity: 1}}
	channel := ChannelCode(MethodBCAVA)

	t.Run("Success", func(t *testing.T) {
		// Mock Response
		respBody := `{
			"payment_request_id": "pr-123",
			"reference_id": "ord-123",
			"request_amount": 100000,
			"status": "PENDING",
			"channel_code": "BCA_VIRTUAL_ACCOUNT",
			"channel_properties": {
				"expires_at": "2024-12-31T23:59:59Z"
			},
			"actions": [
				{
					"descriptor": "VIRTUAL_ACCOUNT_NUMBER",
					"value": "1234567890"
				}
			]
		}`

		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://api.xendit.co/v3/payment_requests", req.URL.String())

			// Verify Auth
			user, _, ok := req.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, apiKey, user)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}
		})

		resp, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "pr-123", resp.ProviderPaymentID)
		assert.Equal(t, "1234567890", resp.PaymentCode)
	})

	t.Run("Success_StatusCreated", func(t *testing.T) {
		// Mock Response for 201 Created
		respBody := `{
			"payment_request_id": "pr-123",
			"reference_id": "ord-123",
			"request_amount": 100000,
			"status": "PENDING",
			"channel_code": "BCA_VIRTUAL_ACCOUNT",
			"channel_properties": {
				"expires_at": "2024-12-31T23:59:59Z"
			},
			"actions": []
		}`

		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}
		})

		resp, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "pr-123", resp.ProviderPaymentID)
	})

	t.Run("APIError", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error_code": "INVALID_DATA"}`)),
				Header:     make(http.Header),
			}
		})

		_, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "xendit error")
	})

	t.Run("NetworkError", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripperWithError(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})

		_, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{invalid-json`)),
				Header:     make(http.Header),
			}
		})

		_, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.Error(t, err)
	})

	t.Run("NoVACode", func(t *testing.T) {
		// Response without actions
		respBody := `{
			"payment_request_id": "pr-123",
			"status": "PENDING",
			"actions": []
		}`

		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}
		})

		resp, err := gw.CreateInvoice(context.Background(), externalID, buyer, amount, items, channel)
		assert.NoError(t, err)
		assert.Equal(t, "", resp.PaymentCode)
	})

	t.Run("Success_WithRedirectURL", func(t *testing.T) {
		// Mock Response with a redirect URL in actions
		respBody := `{
			"payment_request_id": "pr-456",
			"reference_id": "ord-456",
			"status": "PENDING",
			"channel_code": "SHOPEEPAY",
			"channel_properties": {
				"expires_at": "2024-12-31T23:59:59Z"
			},
			"actions": [
				{
					"descriptor": "WEB_URL",
					"value": "https://shopee.co.id/pay"
				}
			]
		}`

		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}
		})

		resp, err := gw.CreateInvoice(context.Background(), "ord-456", buyer, amount, items, ChannelCode(MethodSHOPEE))
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "pr-456", resp.ProviderPaymentID)
		assert.Equal(t, "", resp.PaymentCode)                        // No payment code in this case
		assert.Equal(t, "https://shopee.co.id/pay", resp.InvoiceURL) // Check the URL
	})
}

func TestXenditGateway_GetPaymentStatus(t *testing.T) {
	apiKey := "test-secret"
	gw := NewXenditGateway(apiKey).(*xenditGateway)
	externalID := "ord-123"

	t.Run("Success", func(t *testing.T) {
		respBody := `[
			{
				"status": "PAID",
				"paid_at": "2024-01-01T10:00:00Z"
			}
		]`

		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, "GET", req.Method)
			assert.Contains(t, req.URL.String(), "/v2/invoices")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
				Header:     make(http.Header),
			}
		})

		status, err := gw.GetPaymentStatus(context.Background(), externalID)
		assert.NoError(t, err)
		assert.Equal(t, "PAID", status.Status)
		assert.NotNil(t, status.PaidAt)
	})

	t.Run("NotFound", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`[]`)), // Empty array
				Header:     make(http.Header),
			}
		})

		_, err := gw.GetPaymentStatus(context.Background(), externalID)
		assert.Error(t, err)
		assert.Equal(t, "invoice not found", err.Error())
	})

	t.Run("NetworkError", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripperWithError(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		})

		_, err := gw.GetPaymentStatus(context.Background(), externalID)
		assert.Error(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`invalid`)),
				Header:     make(http.Header),
			}
		})

		_, err := gw.GetPaymentStatus(context.Background(), externalID)
		assert.Error(t, err)
	})
}

func TestXenditGateway_VerifySignature(t *testing.T) {

	t.Run("SkipInDev", func(t *testing.T) {
		// Explicitly set to empty to ensure dev mode is tested
		t.Setenv("XENDIT_CALLBACK_TOKEN", "")
		gw := NewXenditGateway("secret").(*xenditGateway)
		req, _ := http.NewRequest("POST", "/", nil)
		err := gw.VerifySignature(req)
		assert.NoError(t, err)
	})

	t.Run("ValidSignature", func(t *testing.T) {
		t.Setenv("XENDIT_CALLBACK_TOKEN", "valid-token")
		gw := NewXenditGateway("secret").(*xenditGateway)
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("x-callback-token", "valid-token")

		err := gw.VerifySignature(req)
		assert.NoError(t, err)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		t.Setenv("XENDIT_CALLBACK_TOKEN", "valid-token")
		gw := NewXenditGateway("secret").(*xenditGateway)
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("x-callback-token", "invalid-token")

		err := gw.VerifySignature(req)
		assert.Error(t, err)
		assert.Equal(t, "invalid webhook signature", err.Error())
	})
}

func TestXenditGateway_CancelPayment(t *testing.T) {
	apiKey := "test-secret"
	gw := NewXenditGateway(apiKey).(*xenditGateway)
	externalID := "ord-123"

	t.Run("Success", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, "POST", req.Method)
			assert.Contains(t, req.URL.String(), "/expire!")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}
		})

		err := gw.CancelPayment(context.Background(), externalID)
		assert.NoError(t, err)
	})

	t.Run("NetworkError", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripperWithError(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("net error")
		})

		err := gw.CancelPayment(context.Background(), externalID)
		assert.Error(t, err)
	})

	t.Run("APIError", func(t *testing.T) {
		gw.httpClient.Transport = MockRoundTripper(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "bad request"}`)),
				Header:     make(http.Header),
			}
		})

		err := gw.CancelPayment(context.Background(), externalID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "xendit cancel error")
	})
}

func TestNewXenditGateway(t *testing.T) {
	t.Run("EmptyKey", func(t *testing.T) {
		gw := NewXenditGateway("")
		assert.NotNil(t, gw)
	})
}
