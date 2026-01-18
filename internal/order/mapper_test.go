package order

import (
	"testing"
	"time"
	"warimas-be/internal/address"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMapOrderItemToGraphQL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		item := &OrderItem{
			ID:           1,
			Quantity:     2,
			QuantityType: "pcs",
			Price:        10000,
			Subtotal:     20000,
			VariantID:    "var-1",
			VariantName:  "Variant 1",
			ProductName:  "Product 1",
			ImageURL:     utils.StrPtr("http://image.com"),
		}

		res := MapOrderItemToGraphQL(item)

		assert.Equal(t, int32(1), res.ID)
		assert.Equal(t, int32(2), res.Quantity)
		assert.Equal(t, "pcs", res.QuantityType)
		assert.Equal(t, int32(10000), res.Pricing.Price)
		assert.Equal(t, int32(20000), res.Pricing.Subtotal)
		assert.Equal(t, "var-1", res.Variant.ID)
		assert.Equal(t, "Variant 1", res.Variant.Name)
		assert.Equal(t, "Product 1", res.Variant.ProductName)
		assert.Equal(t, "http://image.com", *res.Variant.ImageURL)
	})
}

func TestToGraphQLOrder(t *testing.T) {
	t.Run("NilOrder", func(t *testing.T) {
		res := ToGraphQLOrder(nil, nil)
		assert.Nil(t, res)
	})

	t.Run("Success_WithAddress", func(t *testing.T) {
		now := time.Now()
		userID := int32(10)
		order := &Order{
			ID:            100,
			ExternalID:    "ext-1",
			UserID:        &userID,
			CreatedAt:     now,
			UpdatedAt:     now,
			InvoiceNumber: utils.StrPtr("INV-1"),
			Currency:      "IDR",
			Subtotal:      10000,
			Tax:           1000,
			Discount:      0,
			ShippingFee:   5000,
			TotalAmount:   16000,
			Status:        OrderStatusPaid,
			Items: []*OrderItem{
				{ID: 1, Price: 10000, Quantity: 1},
			},
		}

		addrID := uuid.New()
		addr := &address.Address{
			ID:           addrID,
			Name:         "Home",
			ReceiverName: "John",
			Phone:        "08123",
			Address1:     "Street 1",
			City:         "Jakarta",
			Province:     "DKI",
			Country:      "ID",
			Postal:       "12345",
		}

		res := ToGraphQLOrder(order, addr)

		assert.Equal(t, int32(100), res.ID)
		assert.Equal(t, "ext-1", res.ExternalID)
		assert.Equal(t, int32(10), res.User.ID)
		assert.Equal(t, now, res.Timestamps.CreatedAt)
		assert.Equal(t, "INV-1", *res.InvoiceNumber)
		assert.Equal(t, model.OrderStatusPaid, res.Status)
		assert.Len(t, res.Items, 1)

		// Pricing
		assert.Equal(t, int32(16000), res.Pricing.Total)

		// Shipping
		assert.NotNil(t, res.Shipping)
		assert.Equal(t, addrID.String(), res.Shipping.Address.ID)
		assert.Equal(t, "Jakarta", res.Shipping.Address.City)
	})

	t.Run("Success_NoAddress", func(t *testing.T) {
		userID := int32(10)
		order := &Order{
			ID:     100,
			UserID: &userID,
		}
		res := ToGraphQLOrder(order, nil)
		assert.Nil(t, res.Shipping)
	})
}

func TestMapCheckoutSessionToGraphQL(t *testing.T) {
	t.Run("NilSession", func(t *testing.T) {
		res := MapCheckoutSessionToGraphQL(nil)
		assert.Nil(t, res)
	})

	t.Run("Success", func(t *testing.T) {
		sessID := uuid.New()
		addrID := uuid.New()
		now := time.Now()

		session := &CheckoutSession{
			ID:          sessID,
			ExternalID:  "sess-ext",
			Status:      CheckoutSessionStatusPending,
			ExpiresAt:   now,
			CreatedAt:   now,
			AddressID:   &addrID,
			Subtotal:    10000,
			Tax:         1000,
			ShippingFee: 5000,
			Discount:    0,
			TotalPrice:  16000,
			Items: []CheckoutSessionItem{
				{ID: uuid.New(), VariantID: "v1", Quantity: 1, Price: 10000},
			},
		}

		res := MapCheckoutSessionToGraphQL(session)

		assert.Equal(t, sessID.String(), res.ID)
		assert.Equal(t, "sess-ext", res.ExternalID)
		assert.Equal(t, model.CheckoutSessionStatusPending, res.Status)
		assert.Equal(t, addrID.String(), *res.AddressID)
		assert.Equal(t, int32(16000), res.TotalPrice)
		assert.Len(t, res.Items, 1)
	})

	t.Run("NoAddress", func(t *testing.T) {
		session := &CheckoutSession{
			ID: uuid.New(),
		}
		res := MapCheckoutSessionToGraphQL(session)
		assert.Nil(t, res.AddressID)
	})
}
