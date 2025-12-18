package auth

import (
	"net/http"
	"strings"
)

func ExtractAccessToken(r *http.Request) string {
	// 1️⃣ Cookie (preferred)
	if cookie, err := r.Cookie("access_token"); err == nil {
		if cookie.Value != "" {
			return cookie.Value
		}
	}

	// 2️⃣ Authorization header (fallback)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
