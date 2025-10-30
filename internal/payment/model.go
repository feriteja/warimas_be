package payment

import (
	"time"
)

type Payment struct {
	ID            uint
	OrderID       uint
	ExternalID    string
	InvoiceURL    string
	Amount        float64
	Status        string
	PaymentMethod string
	ChannelCode   string
	PaymentCode   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderItem struct {
	ProductID uint
	Quantity  int
}

type PaymentResponse struct {
	ExternalID     string  `json:"external_id"`
	Amount         float64 `json:"amount"`
	Status         string  `json:"status"`
	PaymentMethod  string  `json:"payment_method,omitempty"`
	PaymentCode    string  `json:"payment_code,omitempty"`
	InvoiceURL     string  `json:"invoice_url,omitempty"`
	ChannelCode    string  `json:"channel_code,omitempty"`
	ExpirationTime string  `json:"expires_at,omitempty"`
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
