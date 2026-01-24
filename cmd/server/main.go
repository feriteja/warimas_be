package main

import (
	"database/sql"
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
	"warimas-be/internal/packages"
	"warimas-be/internal/payment"
	"warimas-be/internal/payment/webhook"
	"warimas-be/internal/product"
	"warimas-be/internal/transport"
	"warimas-be/internal/user"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"go.uber.org/zap"
)

var (
	initDBFunc      = db.InitDB
	startServerFunc = http.ListenAndServe
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	env := os.Getenv("APP_ENV") // "development" or "production"
	logger.Init(env)
	defer logger.Sync()

	cfg := config.LoadConfig()

	logger.L().Info("Connecting to database...")

	// Init DB
	database := initDBFunc(cfg)
	defer database.Close()

	router := newServer(cfg, database)

	logger.L().Info("🚀 Warimas Backend Started",
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.AppPort),
	)

	return startServerFunc(":"+cfg.AppPort, router)
}

func newServer(cfg *config.Config, database *sql.DB) *http.ServeMux {
	// -------------------------------------------------------------------------
	// Init Repositories
	// -------------------------------------------------------------------------
	productRepo := product.NewRepository(database)
	userRepo := user.NewRepository(database)
	cartRepo := cart.NewRepository(database)
	orderRepo := order.NewRepository(database)
	paymentRepo := payment.NewRepository(database)
	categoryRepo := category.NewRepository(database)
	addressRepo := address.NewRepository(database)
	packagesRepo := packages.NewRepository(database)

	// -------------------------------------------------------------------------
	// Init Services
	// -------------------------------------------------------------------------
	productSvc := product.NewService(productRepo)
	userSvc := user.NewService(userRepo)
	cartSvc := cart.NewService(cartRepo, productRepo)
	categorySvc := category.NewService(categoryRepo)
	addressSvc := address.NewService(addressRepo)
	packagesSvc := packages.NewService(packagesRepo)

	paymentGateway := payment.NewXenditGateway(cfg.XenditSecretKey)
	orderSvc := order.NewService(orderRepo, paymentRepo, paymentGateway, addressRepo, userRepo)
	webhookHandler := webhook.NewWebhookHandler(orderSvc, paymentGateway, paymentRepo)

	// -------------------------------------------------------------------------
	// GraphQL Resolver & Server
	// -------------------------------------------------------------------------
	resolver := &graph.Resolver{
		DB:          database,
		ProductSvc:  productSvc,
		UserSvc:     userSvc,
		CartSvc:     cartSvc,
		OrderSvc:    orderSvc,
		CategorySvc: categorySvc,
		AddressSvc:  addressSvc,
		PackageSvc:  packagesSvc,
	}

	srv := handler.NewDefaultServer(graph.NewSchema(resolver))

	return setupRouter(srv, webhookHandler.PaymentWebhookHandler)
}

func setupRouter(srv *handler.Server, paymentWebhookHandler http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	graphqlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := transport.WithHTTP(r.Context(), r, w)
		srv.ServeHTTP(w, r.WithContext(ctx))
	})

	mux.Handle("/query",
		middleware.CORS(
			middleware.LoggingMiddleware(
				middleware.AuthMiddleware(
					middleware.RateLimitMiddleware(graphqlHandler),
				),
			),
		),
	)

	// Apply RateLimitMiddleware to webhook (will use "strict" tier based on path)
	mux.Handle("/webhook/payment", middleware.RateLimitMiddleware(paymentWebhookHandler))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	return mux
}
