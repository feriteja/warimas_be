package graph

import (
	"database/sql"
	"warimas-be/internal/address"
	"warimas-be/internal/cart"
	"warimas-be/internal/category"
	"warimas-be/internal/order"
	"warimas-be/internal/product"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct {
	DB          *sql.DB
	ProductSvc  product.Service
	UserSvc     user.Service
	CartSvc     cart.Service
	OrderSvc    order.Service
	CategorySvc category.Service
	AddressSvc  address.Service
}

func NewSchema(r *Resolver) graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers: r,
		Directives: DirectiveRoot{
			Auth: AuthDirective,
		},
	})
}
