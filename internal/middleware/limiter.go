package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
	"warimas-be/internal/utils"

	"golang.org/x/time/rate"
)

// Rate Limit Tiers
const (
	// Auth / login / OTP / payment (Strict)
	limitStrict = rate.Limit(2)
	burstStrict = 5

	// General (Default)
	limitGeneral = rate.Limit(10)
	burstGeneral = 20

	// Frontend-heavy apps
	limitFrontend = rate.Limit(20)
	burstFrontend = 40

	// Internal / trusted services
	limitInternal = rate.Limit(100)
	burstInternal = 200
)

// visitor holds the rate limiter and the last time it was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

// init starts the background cleanup routine.
func init() {
	go cleanupVisitors()
}

// getVisitor retrieves or creates a rate limiter for the given IP address.
func getVisitor(key string, r rate.Limit, b int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[key]
	if !exists {
		limiter := rate.NewLimiter(r, b)
		visitors[key] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old entries from the visitors map to prevent memory leaks.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// RateLimitMiddleware checks if the request is allowed by the rate limiter.
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Determine Rate Tier
		limit, burst, tier := resolveRateTier(r)

		// 2. Determine Identity Key
		var identity string

		// Prefer User ID if authenticated
		if userID, ok := utils.GetUserIDFromContext(r.Context()); ok {
			identity = fmt.Sprintf("user:%d", userID)
		} else if deviceID := r.Header.Get("X-Device-ID"); deviceID != "" {
			// Use Device ID if provided by the client
			identity = "device:" + deviceID
		} else {
			// Fallback to IP for anonymous requests
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			identity = "ip:" + ip
		}

		// 3. Combine for final bucket key (e.g., "user:1:strict")
		// This ensures the same user has separate quotas for strict vs general actions.
		key := fmt.Sprintf("%s:%s", identity, tier)

		limiter := getVisitor(key, limit, burst)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveRateTier determines which rate limit policy applies to the request.
func resolveRateTier(r *http.Request) (rate.Limit, int, string) {
	// 1. Internal / Trusted Services (Check for a secret header)
	internalKey := os.Getenv("INTERNAL_SECRET_KEY")
	if internalKey != "" && r.Header.Get("X-Service-Auth") == internalKey {
		return limitInternal, burstInternal, "internal"
	}

	// 2. Auth / Payment (Strict)
	// Apply to payment webhooks OR if the client explicitly signals an auth action
	if r.URL.Path == "/webhook/payment" || r.Header.Get("X-Action") == "auth" {
		return limitStrict, burstStrict, "strict"
	}

	// 3. Frontend-heavy apps (High volume)
	if r.Header.Get("X-Client-Type") == "frontend-heavy" {
		return limitFrontend, burstFrontend, "frontend"
	}

	// 4. General (Default)
	return limitGeneral, burstGeneral, "general"
}
