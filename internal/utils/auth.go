package utils

import "context"

type contextKey string

// SetUserContext sets user info into context (called by middleware)
func SetUserContext(ctx context.Context, id uint, email string, role string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, id)
	ctx = context.WithValue(ctx, UserEmailKey, email)
	ctx = context.WithValue(ctx, UserRoleKey, role)
	return ctx
}

// GetUserIDFromContext retrieves userID safely
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(UserIDKey).(uint)
	return id, ok
}

// ✅ GetUserEmailFromContext retrieves userEmail safely
func GetUserEmailFromContext(ctx context.Context) string {
	email, _ := ctx.Value(UserEmailKey).(string)
	return email
}

// Optionally: get role (if you implement roles later)
func GetUserRoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(UserRoleKey).(string)
	return role
}
