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

// Context key for user ID
type contextKey string

const (
	UserIDKey      contextKey = "userID"
	TokenClaimsKey contextKey = "jwtClaims"
)

// Replace with your secret key from user/auth.go
var jwtKey = []byte(os.Getenv("SECRET_KEY"))

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		claims := &user.CustomClaims{}
		token, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			utils.WriteJSONError(w, "token expired", http.StatusUnauthorized)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, utils.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, utils.UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, utils.UserRoleKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
