package main

import (
	"log"
	"net/http"

	"warimas-be/internal/cart"
	"warimas-be/internal/config"
	"warimas-be/internal/db"
	"warimas-be/internal/graph"
	"warimas-be/internal/middleware"
	"warimas-be/internal/product"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	cfg := config.LoadConfig()
	database := db.InitDB(cfg)
	defer database.Close()

	productRepo := product.NewRepository(database)
	productSvc := product.NewService(productRepo)

	userRepo := user.NewRepository(database)
	userSvc := user.NewService(userRepo)

	cartRepo := cart.NewRepository(database)
	cartSvc := cart.NewService(cartRepo)

	resolver := &graph.Resolver{
		DB:         database,
		ProductSvc: productSvc,
		UserSvc:    userSvc,
		CartSvc:    cartSvc,
	}

	srv := handler.NewDefaultServer(graph.NewSchema(resolver))

	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	http.Handle("/query", middleware.AuthMiddleware(srv))

	log.Printf("🚀 GraphQL server running at http://localhost:%s/", cfg.AppPort)
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, nil))
}
