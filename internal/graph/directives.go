package graph

import (
	"context"
	"errors"
	"fmt"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/utils"

	"github.com/99designs/gqlgen/graphql"
)

func AuthDirective(ctx context.Context, obj interface{}, next graphql.Resolver, role *model.Role) (res interface{}, err error) {
	fmt.Printf("context role: %#v\n", ctx.Value(utils.UserRoleKey))

	userRole, ok := ctx.Value(utils.UserRoleKey).(string)
	if !ok || userRole == "" {
		return nil, errors.New("unauthorizeda")
	}

	// Convert GraphQL enum to string
	requiredRole := "USER"
	if role != nil {
		requiredRole = string(*role)
	}

	// Role-based access control
	if requiredRole == "ADMIN" && userRole != "ADMIN" {
		return nil, errors.New("forbidden: admin only")
	}

	return next(ctx)
}
