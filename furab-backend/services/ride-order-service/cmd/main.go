// Package main is the entry point for the ride-order-service.
// It bootstraps the application by loading configuration, connecting to the database,
// initializing event publishers, wiring dependencies, and starting the HTTP server.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"furab-backend/services/ride-order-service/internal/handler"
	"furab-backend/services/ride-order-service/internal/repository"
	"furab-backend/services/ride-order-service/internal/service"
	"furab-backend/shared/config"
	"furab-backend/shared/event/kafka"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Load configuration
	cfg := config.Load("ride-order-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting ride-order-service",
		"port", cfg.ServerPort,
		"environment", cfg.Environment,
	)

	// Connect to PostgreSQL
	db, err := sql.Open("pgx", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	logger.Info("connected to database")

	// Initialize Kafka publisher for ride events
	kafkaPublisher := kafka.NewPublisher(cfg.KafkaBrokers)
	defer kafkaPublisher.Close()

	// Wire dependencies
	orderRepo := repository.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepo, kafkaPublisher)
	orderHandler := handler.NewOrderHandler(orderService)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Health check (no auth required)
	r.Get("/health", handler.HealthCheck)

	// Register ride order routes
	orderHandler.RegisterRoutes(r)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server shutdown error", "error", err)
		}
	}()

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	logger.Info("server stopped")
}
