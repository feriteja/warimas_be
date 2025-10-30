// internal/payment/payment.go
package payment

import (
	"net/http"
)

type Gateway interface {
	CreateInvoice(orderID uint,
		buyerName string,
		amount float64,
		customerEmail string,
		items []OrderItem,
		channelCode ChannelCode,
	) (*PaymentResponse, error)
	GetPaymentStatus(externalID string) (*PaymentStatus, error)
	CancelPayment(externalID string) error
	VerifySignature(r *http.Request) error
}
