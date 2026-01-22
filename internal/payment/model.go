package payment

import (
	"encoding/json"
	"time"
)

type Payment struct {
	ID                uint
	OrderID           uint
	ExternalReference string //PaymentRequestID from xendit
	ProviderPaymentID string
	InvoiceURL        string
	Amount            int64
	Status            string
	PaymentMethod     ChannelCode
	ChannelCode       string
	PaymentCode       string
	Currency          string
	Provider          string
	PaidAt            time.Time
	failure_reason    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ExpireAt          time.Time
}

type BuyerInfo struct {
	Name  string
	Email *string
	Phone string
}

type OrderItem struct {
	ProductID uint
	Quantity  int
}

type XenditItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"`
}

type PaymentResponse struct {
	ProviderPaymentID string           `json:"provider_payment_id"`
	ReferenceID       string           `json:"reference_id"`
	Amount            int64            `json:"amount"`
	Status            string           `json:"status"`
	PaymentMethod     ChannelCode      `json:"payment_method,omitempty"`
	PaymentCode       string           `json:"payment_code,omitempty"`
	InvoiceURL        string           `json:"invoice_url,omitempty"`
	ChannelCode       string           `json:"channel_code,omitempty"`
	ExpirationTime    time.Time        `json:"expires_at,omitempty"`
	RawResponse       *json.RawMessage `json:"raw_response,omitempty"`
}

type PaymentStatus struct {
	Status string
	PaidAt *time.Time
}

type ChannelCode string

const (
	ChannelIndomaret ChannelCode = "INDOMARET"
	ChannelAlfamart  ChannelCode = "ALFAMART"

	// Virtual Account
	MethodBCAVA     ChannelCode = "BCA_VIRTUAL_ACCOUNT"
	MethodBNIVA     ChannelCode = "BNI_VIRTUAL_ACCOUNT"
	MethodMandiriVA ChannelCode = "MANDIRI_VIRTUAL_ACCOUNT"

	// QRIS
	MethodQRIS ChannelCode = "QRIS"
	MethodCOD  ChannelCode = "COD"

	// E-Wallet
	MethodOVO     ChannelCode = "OVO"
	MethodDANA    ChannelCode = "DANA"
	MethodLINKAJA ChannelCode = "LINKAJA"
	MethodSHOPEE  ChannelCode = "SHOPEEPAY"
	MethodGOPAY   ChannelCode = "GOPAY"

	// Retail Outlet
	MethodAlfamart  ChannelCode = "ALFAMART"
	MethodIndomaret ChannelCode = "INDOMARET"

	// Credit Card
	MethodCreditCard ChannelCode = "CARDS"
)

const (
	ActionQRCode      = "QR_CODE"
	ActionCheckoutURL = "CHECKOUT_URL"
)

type XenditPaymentResponse struct {
	PaymentRequestID string `json:"payment_request_id"`
	Country          string `json:"country"`
	Currency         string `json:"currency"`
	BusinessID       string `json:"business_id"`
	ReferenceID      string `json:"reference_id"`

	RequestAmount int64  `json:"request_amount"`
	Status        string `json:"status"`
	Type          string `json:"type"`
	ChannelCode   string `json:"channel_code"`
	CustomerID    string `json:"customer_id,omitempty"`

	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`

	ChannelProperties struct {
		DisplayName string     `json:"display_name,omitempty"`
		ExpiresAt   *time.Time `json:"expires_at"`
	} `json:"channel_properties"`

	Actions []struct {
		Type       string `json:"type"`       // PRESENT_TO_CUSTOMER
		Descriptor string `json:"descriptor"` // VIRTUAL_ACCOUNT_NUMBER
		Value      string `json:"value"`      // VA number
	} `json:"actions,omitempty"`

	Metadata struct {
		Items []XenditItem `json:"items,omitempty"`
	} `json:"metadata,omitempty"`
}

type WebhookPayload struct {
	Created    time.Time `json:"created"`
	BusinessID string    `json:"business_id"`
	Event      string    `json:"event"`
	APIVersion string    `json:"api_version"`
	Data       struct {
		Type             string    `json:"type"`
		Status           string    `json:"status"`
		Country          string    `json:"country"`
		Created          string    `json:"created"`
		Updated          time.Time `json:"updated"`
		Currency         string    `json:"currency"`
		PaymentID        string    `json:"payment_id"`
		BusinessID       string    `json:"business_id"`
		CustomerID       string    `json:"customer_id"`
		ChannelCode      string    `json:"channel_code"`
		ReferenceID      string    `json:"reference_id"`
		CaptureMethod    string    `json:"capture_method"`
		RequestAmount    int64     `json:"request_amount"`
		PaymentRequestID string    `json:"payment_request_id"`

		Captures []struct {
			CaptureID        string `json:"capture_id"`
			CaptureAmount    int64  `json:"capture_amount"`
			CaptureTimestamp string `json:"capture_timestamp"`
		} `json:"captures"`

		Metadata struct {
			Items []XenditItem `json:"items"`
		} `json:"metadata"`
	} `json:"data"`
}
