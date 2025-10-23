package graph

import (
	"context"

	"warimas-be/internal/middleware"
)

func GetUserIDFromContext(ctx context.Context) (int, bool) {
	uid, ok := ctx.Value(middleware.UserIDKey).(int)
	return uid, ok
}
