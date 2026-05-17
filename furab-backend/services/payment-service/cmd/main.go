// Package main is the entry point for payment-service.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/payment-service/internal/handler"
	"furab-backend/services/payment-service/internal/repository"
	"furab-backend/services/payment-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type noopPricingClient struct{}

type noopPromoClient struct{}

type noopWalletClient struct{}

type noopSettlementClient struct{}

func (noopPricingClient) GetTotalAmount(ctx context.Context, orderID string) (float64, error) {
	return 0, nil
}

func (noopPromoClient) ApplyPromo(ctx context.Context, promoCode string, totalAmount float64) (float64, float64, error) {
	return totalAmount, 0, nil
}

func (noopWalletClient) LockBalance(ctx context.Context, userID string, amount float64, reference string) error {
	return nil
}

func (noopWalletClient) UnlockBalance(ctx context.Context, userID string, amount float64, reference string) error {
	return nil
}

func (noopWalletClient) DeductBalance(ctx context.Context, userID string, amount float64, reference string) error {
	return nil
}

func (noopWalletClient) CreditBalance(ctx context.Context, userID string, amount float64, reference string) error {
	return nil
}

func (noopSettlementClient) TriggerSettlement(ctx context.Context, paymentID, orderID string, finalAmount float64) error {
	return nil
}

func main() {
	cfg := config.Load("payment-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting payment-service", "port", cfg.ServerPort)

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

	repo := repository.NewPostgresPaymentRepository(db)
	paymentSvc := service.NewPaymentService(repo, noopPricingClient{}, noopPromoClient{}, noopWalletClient{}, noopSettlementClient{})
	paymentHandler := handler.NewPaymentHandler(paymentSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	paymentHandler.RegisterRoutes(r)

	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
