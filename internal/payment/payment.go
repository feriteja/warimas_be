// internal/payment/payment.go
package payment

import (
	"net/http"
)

type Gateway interface {
	CreateInvoice(externalID string,
		buyerName string,
		amount int64,
		customerEmail string,
		items []XenditItem,
		channelCode ChannelCode,
	) (*PaymentResponse, error)
	GetPaymentStatus(externalID string) (*PaymentStatus, error)
	CancelPayment(externalID string) error
	VerifySignature(r *http.Request) error
}
