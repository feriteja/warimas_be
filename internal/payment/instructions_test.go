package payment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstructions(t *testing.T) {
	// Tests that we get a template string back for known methods
	// Adjust "BCA_VA" to match the actual constants in your payment package
	t.Run("ReturnsTemplateForKnownMethod", func(t *testing.T) {
		instructions := GetInstructions(MethodBCAVA)
		assert.NotEmpty(t, instructions)

		found := false
		for _, instr := range instructions {
			if strings.Contains(instr, "{{payment_code}}") {
				found = true
				break
			}
		}
		assert.True(t, found, "Instructions should contain {{payment_code}} placeholder")
	})

	t.Run("ReturnsDefaultOrEmptyForUnknown", func(t *testing.T) {
		instructions := GetInstructions("UNKNOWN_METHOD")
		// Assert behavior based on implementation (empty string or default text)
		assert.NotNil(t, instructions)
	})
}

func TestInjectVariables(t *testing.T) {
	t.Run("ReplacesPlaceholders", func(t *testing.T) {
		template := []string{"Please transfer {{amount}} to VA {{payment_code}} before {{expiry}}."}
		vars := InstructionVars{
			"amount":       "Rp100.000",
			"payment_code": "12345678",
			"expiry":       "tomorrow",
		}

		expected := []string{"Please transfer Rp100.000 to VA 12345678 before tomorrow."}
		result := InjectVariables(template, vars)

		assert.Equal(t, expected, result)
	})

	t.Run("HandlesMissingVariables", func(t *testing.T) {
		template := []string{"Pay {{amount}}"}
		vars := InstructionVars{} // Empty map

		result := InjectVariables(template, vars)
		// Assuming it leaves the placeholder or replaces with empty string
		// Adjust assertion based on actual implementation
		assert.Contains(t, result[0], "{{amount}}")
	})
}
