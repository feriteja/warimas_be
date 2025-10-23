package graph

import (
	"context"
	"errors"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/middleware"

	"github.com/99designs/gqlgen/graphql"
	"github.com/golang-jwt/jwt/v5"
)

func AuthDirective(ctx context.Context, obj interface{}, next graphql.Resolver, role *model.Role) (res interface{}, err error) {

	token := ctx.Value(middleware.TokenClaimsKey)
	if token == nil {
		return nil, errors.New("unauthorized")
	}

	claims, ok := token.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token")
	}

	userRole, _ := claims["role"].(string)
	if userRole == "" {
		return nil, errors.New("unauthorized")

	}

	// Convert GraphQL enum to string (if provided)
	requiredRole := "USER"
	if role != nil {
		requiredRole = string(*role)
	}
	// Authorization check
	if requiredRole == "ADMIN" && userRole != "ADMIN" {
		return nil, errors.New("forbidden: admin only")
	}
	return next(ctx)
}
