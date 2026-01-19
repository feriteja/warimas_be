package user

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "secret"
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secret"
	hash, _ := HashPassword(password)

	assert.True(t, CheckPasswordHash(password, hash))
	assert.False(t, CheckPasswordHash("wrong", hash))
}

func TestGenerateJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")

	token, err := GenerateJWT(1, "USER", "test@example.com", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Test with SellerID
	sellerID := "seller-123"
	tokenSeller, err := GenerateJWT(1, "USER", "test@example.com", &sellerID)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenSeller)
}

func TestGenerateJWT_NoSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	_, err := GenerateJWT(1, "USER", "test@example.com", nil)
	assert.Error(t, err)
	assert.Equal(t, "JWT_SECRET is not set", err.Error())
}

func TestParseJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "testsecret")

	// Generate a valid token first
	tokenStr, _ := GenerateJWT(1, "USER", "test@example.com", nil)

	t.Run("Success", func(t *testing.T) {
		claims, err := ParseJWT(tokenStr)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, uint(1), claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, err := ParseJWT("invalid-token-string")
		assert.Error(t, err)
	})

	t.Run("NoSecret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "")
		_, err := ParseJWT(tokenStr)
		assert.Error(t, err)
		assert.Equal(t, "JWT_SECRET is not set", err.Error())
	})

	t.Run("WrongSecret", func(t *testing.T) {
		// Generate token with one secret
		t.Setenv("JWT_SECRET", "secret1")
		token, _ := GenerateJWT(1, "USER", "test@example.com", nil)

		// Try to parse with another
		t.Setenv("JWT_SECRET", "secret2")
		_, err := ParseJWT(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature is invalid")
	})

	t.Run("InvalidSigningMethod", func(t *testing.T) {
		// Create a token with "none" method
		token := jwt.New(jwt.SigningMethodNone)
		tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

		_, err := ParseJWT(tokenStr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected signing method")
	})

	t.Run("InvalidClaimsType", func(t *testing.T) {
		// Generate token with MapClaims instead of CustomClaims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"foo": "bar"})
		tokenStr, _ := token.SignedString([]byte("testsecret"))

		_, err := ParseJWT(tokenStr)
		if assert.Error(t, err) {
			assert.Equal(t, "invalid token", err.Error())
		}
	})
}
