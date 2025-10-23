package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

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

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ctx := context.WithValue(r.Context(), TokenClaimsKey, claims)
			if uid, ok := claims["user_id"].(float64); ok {
				ctx = context.WithValue(ctx, UserIDKey, int(uid))
			}
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
