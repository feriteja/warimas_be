package utils

import "context"

const (
	UserIDKey    contextKey = "user_id"
	SellerIDKey  contextKey = "seller_id"
	UserEmailKey contextKey = "email"
	UserRoleKey  contextKey = "role"
)

const (
	ProductStatusActive  = "active"
	ProductStatusDisable = "disable"
)

type ctxKey string

const internalRequestKey ctxKey = "internal_request"

func WithInternalRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, internalRequestKey, true)
}

func SetInternalContext(ctx context.Context) context.Context {
	return WithInternalRequest(ctx)
}
