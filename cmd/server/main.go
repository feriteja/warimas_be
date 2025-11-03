package main

import (
	"fmt"
	"log"
	"net/http"

	"warimas-be/internal/cart"
	"warimas-be/internal/config"
	"warimas-be/internal/db"
	"warimas-be/internal/graph"
	"warimas-be/internal/logger"
	"warimas-be/internal/middleware"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"
	"warimas-be/internal/payment/webhook"
	"warimas-be/internal/product"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	logger.Debug("Connecting to database...")

	// Init DB
	database := db.InitDB(cfg)
	defer database.Close()

	// Init repositories
	productRepo := product.NewRepository(database)
	userRepo := user.NewRepository(database)
	cartRepo := cart.NewRepository(database)
	orderRepo := order.NewRepository(database)
	paymentRepo := payment.NewRepository(database)

	// Init services
	productSvc := product.NewService(productRepo)
	userSvc := user.NewService(userRepo)
	cartSvc := cart.NewService(cartRepo)

	// ✅ Add payment gateway initialization
	paymentGateway := payment.NewXenditGateway(cfg.XenditSecretKey)
	orderSvc := order.NewService(orderRepo, paymentRepo, paymentGateway)
	webhookHandler := webhook.NewWebhookHandler(orderSvc, paymentGateway)

	// GraphQL resolver
	resolver := &graph.Resolver{
		DB:         database,
		ProductSvc: productSvc,
		UserSvc:    userSvc, ///i put register & login here
		CartSvc:    cartSvc,
		OrderSvc:   orderSvc,
	}

	// GraphQL server
	srv := handler.NewDefaultServer(graph.NewSchema(resolver))

	// Routes
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	http.Handle("/query", middleware.LoggingMiddleware(middleware.AuthMiddleware(srv)))
	http.HandleFunc("/webhook/payment", webhookHandler.PaymentWebhookHandler)

	// Health or default route (optional)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	logger.Info("🚀 Starting Warimas Backend", map[string]interface{}{
		"env":  "development",
		"port": "8080",
	})
	log.Printf("🚀 GraphQL server running at http://localhost:%s/", cfg.AppPort)
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, nil))
}
