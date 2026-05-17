// Package main is the entry point for settlement-service.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/settlement-service/internal/handler"
	"furab-backend/services/settlement-service/internal/repository"
	"furab-backend/services/settlement-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type noopWalletClient struct{}
type noopDriverClient struct{}
type noopMerchantClient struct{}

func (noopWalletClient) CreditBalance(ctx context.Context, walletID string, amount float64, referenceID string) error {
	return nil
}

func (noopDriverClient) GetDriverWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	return "driver-wallet-noop", nil
}

func (noopMerchantClient) GetMerchantWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	return "merchant-wallet-noop", nil
}

func main() {
	cfg := config.Load("settlement-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting settlement-service", "port", cfg.ServerPort)

	db, err := sql.Open("pgx", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	logger.Info("connected to database")

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	repo := repository.NewPostgresSettlementRepository(db)
	svc := service.NewSettlementService(repo, noopWalletClient{}, noopDriverClient{}, noopMerchantClient{})
	h := handler.NewSettlementHandler(svc)
	h.RegisterRoutes(r)

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
