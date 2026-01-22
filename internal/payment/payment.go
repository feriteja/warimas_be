// internal/payment/payment.go
package payment

import (
	"context"
	"net/http"
)

type Gateway interface {
	CreateInvoice(ctx context.Context,
		externalID string,
		buyer BuyerInfo,
		amount int64,
		items []XenditItem,
		channelCode ChannelCode,
	) (*PaymentResponse, error)
	GetPaymentStatus(ctx context.Context, externalID string) (*PaymentStatus, error)
	CancelPayment(ctx context.Context, externalID string) error
	VerifySignature(r *http.Request) error
}
