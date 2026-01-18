package cart

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapper_MapToModel(t *testing.T) {
	// Setup a sample time for consistency
	now := time.Now()

	// Create a sample CartRow input
	// Note: Adjust field names (e.g., CartID vs ID) to match your actual CartRow struct definition
	productImage := "product.jpg"
	variantImage := "variant.jpg"
	row := &CartRow{
		CartID:          "cart-1",
		UserID:          1,
		Quantity:        2,
		CreatedAt:       now,
		UpdatedAt:       &now,
		ProductID:       "prod-1",
		ProductName:     "Test Shirt",
		SellerID:        "seller-1",
		SellerName:      "Best Seller",
		CategoryID:      "cat-1",
		SubcategoryID:   "sub-1",
		Status:          "active",
		ProductImageURL: &productImage,
		VariantID:       "var-1",
		VariantName:     "Red / L",
		Price:           15000,
		Stock:           50,
		VariantImageURL: &variantImage,
		QuantityType:    "pcs",
	}

	// Act: Call your mapper function
	// Replace 'MapRowToModel' with the actual function name from your mapper.go file
	results := MapCartItemToGraphQL([]*CartRow{row})

	// Assert: Verify the mapping is correct
	assert.NotEmpty(t, results)
	result := results[0]
	assert.Equal(t, "cart-1", result.ID)
	assert.Equal(t, int32(1), result.UserID)
	assert.Equal(t, int32(2), result.Quantity)

	// Verify Product mapping
	assert.NotNil(t, result.Product)
	assert.Equal(t, "prod-1", result.Product.ID)
	assert.Equal(t, "Test Shirt", result.Product.Name)
	assert.Equal(t, "Best Seller", result.Product.SellerName)

	// Verify Variant mapping nested inside Product
	assert.NotNil(t, result.Product.Variant)
	assert.Equal(t, "var-1", result.Product.Variant.ID)
	assert.Equal(t, float64(15000), result.Product.Variant.Price)
}

func TestMapper_EdgeCases(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		results := MapCartItemToGraphQL(nil)
		assert.Empty(t, results)

		results = MapCartItemToGraphQL([]*CartRow{})
		assert.Empty(t, results)
	})

	t.Run("NilImages", func(t *testing.T) {
		// Row with nil pointer fields
		row := &CartRow{
			CartID:          "cart-1",
			ProductImageURL: nil,
			VariantImageURL: nil,
		}
		results := MapCartItemToGraphQL([]*CartRow{row})
		assert.NotEmpty(t, results)
		assert.Nil(t, results[0].Product.ImageURL)
	})
}
