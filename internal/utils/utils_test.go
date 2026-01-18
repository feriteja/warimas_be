package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"warimas-be/internal/graph/model"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestUserContext(t *testing.T) {
	t.Run("SetUserContext and GetUserIDFromContext", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(100)
		email := "user@example.com"
		role := "user"

		// Set the user context
		ctx = SetUserContext(ctx, userID, email, role)
		assert.NotNil(t, ctx)

		// Retrieve the user ID
		id, ok := GetUserIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, userID, id)

		// Retrieve other fields
		assert.Equal(t, email, GetUserEmailFromContext(ctx))
		assert.Equal(t, role, GetUserRoleFromContext(ctx))
	})

	t.Run("GetUserIDFromContext with empty context", func(t *testing.T) {
		ctx := context.Background()
		_, ok := GetUserIDFromContext(ctx)
		assert.False(t, ok)
	})
}

func TestIsInternalRequest(t *testing.T) {
	t.Run("Returns false for empty context", func(t *testing.T) {
		ctx := context.Background()
		isInternal := IsInternalRequest(ctx)
		assert.False(t, isInternal)
	})

	t.Run("Returns true for internal request", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithInternalRequest(ctx)
		assert.True(t, IsInternalRequest(ctx))
	})
}

func TestToUint(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint
		expectErr bool
	}{
		{
			name:      "Valid number",
			input:     "123",
			expected:  123,
			expectErr: false,
		},
		{
			name:      "Zero",
			input:     "0",
			expected:  0,
			expectErr: false,
		},
		{
			name:      "Negative number",
			input:     "-1",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "Non-numeric string",
			input:     "abc",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "Empty string",
			input:     "",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToUint(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStrPtr(t *testing.T) {
	t.Run("Returns pointer to string", func(t *testing.T) {
		input := "test string"
		ptr := StrPtr(input)

		assert.NotNil(t, ptr)
		assert.Equal(t, input, *ptr)
	})
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sellerID string
		expected string
	}{
		{
			name:     "Simple",
			input:    "Product Name",
			sellerID: "seller-123",
			expected: "seller-product-name",
		},
		{
			name:     "With Special Chars",
			input:    "Product & Name!",
			sellerID: "123-456",
			expected: "123-product-name",
		},
		{
			name:     "Multiple Dashes",
			input:    "Product   Name",
			sellerID: "abc",
			expected: "abc-product-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Slugify(tt.input, tt.sellerID))
		})
	}
}

func TestFormatIDR(t *testing.T) {
	tests := []struct {
		amount   int64
		expected string
	}{
		{0, "Rp 0"},
		{100, "Rp 100"},
		{1000, "Rp 1.000"},
		{1000000, "Rp 1.000.000"},
		{123456789, "Rp 123.456.789"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatIDR(tt.amount))
		})
	}
}

func TestPtrHelpers(t *testing.T) {
	t.Run("PtrString", func(t *testing.T) {
		str := "test"
		assert.Equal(t, "test", PtrString(&str))
		assert.Equal(t, "", PtrString(nil))
	})

	t.Run("PtrInt32", func(t *testing.T) {
		val := int32(10)
		assert.Equal(t, int32(10), PtrInt32(&val))
		assert.Equal(t, int32(0), PtrInt32(nil))
	})

	t.Run("PtrInt64", func(t *testing.T) {
		val := int64(10)
		assert.Equal(t, int64(10), PtrInt64(&val))
		assert.Equal(t, int64(0), PtrInt64(nil))
	})
}

func TestHasAnyField(t *testing.T) {
	t.Run("Has field", func(t *testing.T) {
		ctx := context.Background()

		// Setup gqlgen context
		opCtx := &graphql.OperationContext{Variables: map[string]interface{}{}}
		ctx = graphql.WithOperationContext(ctx, opCtx)

		fieldCtx := &graphql.FieldContext{
			Field: graphql.CollectedField{
				Selections: ast.SelectionSet{&ast.Field{Name: "name"}},
			},
		}
		ctx = graphql.WithFieldContext(ctx, fieldCtx)

		hasField := HasAnyField(ctx, "name")
		assert.True(t, hasField)
	})

	t.Run("Does not have field", func(t *testing.T) {
		ctx := context.Background()

		opCtx := &graphql.OperationContext{Variables: map[string]interface{}{}}
		ctx = graphql.WithOperationContext(ctx, opCtx)

		fieldCtx := &graphql.FieldContext{
			Field: graphql.CollectedField{
				Selections: ast.SelectionSet{&ast.Field{Name: "email"}},
			},
		}
		ctx = graphql.WithFieldContext(ctx, fieldCtx)

		hasField := HasAnyField(ctx, "name")
		assert.False(t, hasField)
	})
}

func TestHasAnyUpdateProductField(t *testing.T) {
	t.Run("Has a field updated", func(t *testing.T) {
		input := model.UpdateProduct{
			Name: StrPtr("New Name"),
		}
		hasField := HasAnyUpdateProductField(input)
		assert.True(t, hasField)
	})

	t.Run("Does not have field updated", func(t *testing.T) {
		input := model.UpdateProduct{}
		hasField := HasAnyUpdateProductField(input)
		assert.False(t, hasField)
	})
}

func TestParseUint(t *testing.T) {
	assert.Equal(t, uint(123), ParseUint("123"))
	assert.Equal(t, uint(0), ParseUint("abc"))
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONError(w, "error message", http.StatusBadRequest)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "error message", body["error"])
}

func TestFormatTimePtr(t *testing.T) {
	now := time.Now()
	s := FormatTimePtr(&now)
	assert.NotNil(t, s)
	assert.Equal(t, now.Format(time.RFC3339), *s)
	assert.Nil(t, FormatTimePtr(nil))
}

func TestHasAnyVariantUpdateField(t *testing.T) {
	v := &model.UpdateVariant{Name: StrPtr("name")}
	assert.True(t, HasAnyVariantUpdateField(v))

	vEmpty := &model.UpdateVariant{}
	assert.False(t, HasAnyVariantUpdateField(vEmpty))
}

func TestExternalIDFromSession(t *testing.T) {
	id := ExternalIDFromSession("prefix", "session-id")
	assert.Contains(t, id, "prefix_")
}

func TestSetInternalContext(t *testing.T) {
	ctx := context.Background()
	ctx = SetInternalContext(ctx)
	assert.True(t, IsInternalRequest(ctx))
}
