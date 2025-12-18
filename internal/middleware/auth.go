package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"warimas-be/internal/user"
	"warimas-be/internal/utils"

	"github.com/golang-jwt/jwt/v5"
)

// Context key types
type contextKey string

const (
	UserIDKey      contextKey = "userID"
	TokenClaimsKey contextKey = "jwtClaims"
)

// JWT secret
var jwtKey = []byte(os.Getenv("SECRET_KEY"))

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1️⃣ Extract token (cookie first, header fallback)
		tokenStr := extractAccessToken(r)
		if tokenStr == "" {
			// No token → continue as anonymous
			next.ServeHTTP(w, r)
			return
		}

		// 2️⃣ Parse & validate token
		claims := &user.CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			utils.WriteJSONError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// 3️⃣ Expiration check
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			utils.WriteJSONError(w, "token expired", http.StatusUnauthorized)
			return
		}

		// 4️⃣ Inject user data into context
		ctx := r.Context()
		ctx = context.WithValue(ctx, utils.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, utils.UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, utils.UserRoleKey, claims.Role)
		ctx = context.WithValue(ctx, TokenClaimsKey, claims)

		// 5️⃣ Continue request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// 🔐 Token extractor (cookie → header)
func extractAccessToken(r *http.Request) string {
	// Cookie (preferred)
	if cookie, err := r.Cookie("access_token"); err == nil {
		if cookie.Value != "" {
			return cookie.Value
		}
	}

	// Authorization header (fallback)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
