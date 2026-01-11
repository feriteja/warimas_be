package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"warimas-be/internal/address"
	"warimas-be/internal/cart"
	"warimas-be/internal/category"
	"warimas-be/internal/config"
	"warimas-be/internal/db"
	"warimas-be/internal/graph"
	"warimas-be/internal/logger"
	"warimas-be/internal/middleware"
	"warimas-be/internal/order"
	"warimas-be/internal/payment"
	"warimas-be/internal/payment/webhook"
	"warimas-be/internal/product"
	"warimas-be/internal/transport"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"go.uber.org/zap"
)

func main() {
	env := os.Getenv("APP_ENV") // "development" or "production"
	logger.Init(env)
	defer logger.Sync()

	cfg := config.LoadConfig()

	logger.L().Info("Connecting to database...")

	// Init DB
	database := db.InitDB(cfg)
	defer database.Close()

	// Init repositories
	productRepo := product.NewRepository(database)
	userRepo := user.NewRepository(database)
	cartRepo := cart.NewRepository(database)
	orderRepo := order.NewRepository(database)
	paymentRepo := payment.NewRepository(database)
	categoryRepo := category.NewRepository(database)
	addressRepo := address.NewRepository(database)

	// Init services
	productSvc := product.NewService(productRepo)
	userSvc := user.NewService(userRepo)
	cartSvc := cart.NewService(cartRepo, productRepo)
	categorySvc := category.NewService(categoryRepo)
	addressSvc := address.NewService(addressRepo)

	paymentGateway := payment.NewXenditGateway(cfg.XenditSecretKey)
	orderSvc := order.NewService(orderRepo, paymentRepo, paymentGateway, addressRepo)
	webhookHandler := webhook.NewWebhookHandler(orderSvc, paymentGateway, paymentRepo)

	// GraphQL resolver
	resolver := &graph.Resolver{
		DB:          database,
		ProductSvc:  productSvc,
		UserSvc:     userSvc,
		CartSvc:     cartSvc,
		OrderSvc:    orderSvc,
		CategorySvc: categorySvc,
		AddressSvc:  addressSvc,
	}

	srv := handler.NewDefaultServer(graph.NewSchema(resolver))

	// Routes
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	graphqlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := transport.WithHTTP(r.Context(), r, w)
		srv.ServeHTTP(w, r.WithContext(ctx))
	})

	http.Handle("/query",
		middleware.CORS(
			middleware.LoggingMiddleware(
				middleware.AuthMiddleware(graphqlHandler),
			),
		),
	)

	http.HandleFunc("/webhook/payment", webhookHandler.PaymentWebhookHandler)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	logger.L().Info("🚀 Warimas Backend Started",
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.AppPort),
	)

	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, nil))
}
