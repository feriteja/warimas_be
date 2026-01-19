package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateInvoiceNumber(t *testing.T) {
	t.Run("Format", func(t *testing.T) {
		inv := GenerateInvoiceNumber()
		// Expected format: INV-YYYYMMDD-HHMMSS-mmm-RRRR
		// Example: INV-20231027-103000-123-4567

		assert.True(t, strings.HasPrefix(inv, "INV-"), "Should start with INV-")

		parts := strings.Split(inv, "-")
		if assert.Len(t, parts, 5, "Should have 5 parts separated by hyphens") {
			assert.Equal(t, "INV", parts[0])
			assert.Len(t, parts[1], 8, "Date part YYYYMMDD should be 8 chars")
			assert.Len(t, parts[2], 6, "Time part HHMMSS should be 6 chars")
			assert.Len(t, parts[3], 3, "Milliseconds part should be 3 chars")
			assert.Len(t, parts[4], 4, "Random part should be 4 chars")
		}
	})

	t.Run("Uniqueness", func(t *testing.T) {
		inv1 := GenerateInvoiceNumber()
		inv2 := GenerateInvoiceNumber()
		assert.NotEqual(t, inv1, inv2, "Consecutive invoice numbers should be different")
	})
}
