package graph

import (
	"database/sql"
	"warimas-be/internal/cart"
	"warimas-be/internal/product"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct {
	DB         *sql.DB
	ProductSvc product.Service
	UserSvc    user.Service
	CartSvc    cart.Service
}

func NewSchema(r *Resolver) graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers: r,
		Directives: DirectiveRoot{
			Auth: AuthDirective,
		},
	})
}
