package payment

import (
	"encoding/json"
	"time"
)

type Payment struct {
	ID                uint
	OrderID           uint
	ExternalReference string
	ProviderPaymentID string
	InvoiceURL        string
	Amount            int64
	Status            string
	PaymentMethod     string
	ChannelCode       string
	PaymentCode       string
	Currency          string
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	PaymentMethod     string           `json:"payment_method,omitempty"`
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
	ChannelBCA       ChannelCode = "BCA_VIRTUAL_ACCOUNT"
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
		DisplayName string    `json:"display_name,omitempty"`
		ExpiresAt   time.Time `json:"expires_at"`
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
