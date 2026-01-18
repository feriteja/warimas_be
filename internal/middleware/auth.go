package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"warimas-be/internal/logger"
	"warimas-be/internal/user"
	"warimas-be/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Context key types
type contextKey string

const (
	UserIDKey      contextKey = "userID"
	TokenClaimsKey contextKey = "jwtClaims"
)

// JWT secret

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jwtKey = []byte(os.Getenv("JWT_SECRET"))

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

		// Use logger from context to ensure Request ID is present in logs
		log := logger.FromCtx(r.Context())

		if err != nil || !token.Valid {
			if errors.Is(err, jwt.ErrTokenExpired) {
				log.Warn("auth failed: token expired", zap.Error(err))
			} else {
				log.Warn("auth failed: invalid token", zap.Error(err))
			}
			utils.WriteJSONError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// 4️⃣ Inject user data into context
		ctx := r.Context()
		ctx = context.WithValue(ctx, utils.UserIDKey, claims.UserID)

		if claims.SellerID != nil {
			ctx = context.WithValue(ctx, utils.SellerIDKey, *claims.SellerID)
		}
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
