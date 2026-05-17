// Package main is the entry point for pricing-service.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/pricing-service/internal/client"
	"furab-backend/services/pricing-service/internal/handler"
	"furab-backend/services/pricing-service/internal/repository"
	"furab-backend/services/pricing-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load("pricing-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting pricing-service", "port", cfg.ServerPort)

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

	// Compose dependencies
	repo := repository.NewPostgresPriceRepository(db)
	orderClient := client.NewDummyOrderClient()
	locationClient := client.NewDummyLocationClient()
	priceService := service.NewPriceService(repo, orderClient, locationClient)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.NewPriceHandler(priceService)
	h.RegisterRoutes(r)

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
