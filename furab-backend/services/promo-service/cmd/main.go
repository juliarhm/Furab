// Package main is the entry point for promo-service.
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

	"furab-backend/services/promo-service/internal/client"
	"furab-backend/services/promo-service/internal/handler"
	"furab-backend/services/promo-service/internal/repository"
	"furab-backend/services/promo-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load("promo-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting promo-service",
		"port", cfg.ServerPort,
		"environment", cfg.Environment,
	)

	// Connect to PostgreSQL
	db, err := sql.Open("pgx", cfg.DatabaseURL())
	if err != nil {
		logger.Error("failed to open database connection, falling back to in-memory", "error", err)
	}

	var repo repository.PromoRepository

	if db != nil {
		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Verify database connection
		if err := db.Ping(); err != nil {
			logger.Error("failed to ping database, falling back to in-memory", "error", err)
			repo = repository.NewInMemoryPromoRepository()
		} else {
			logger.Info("connected to database")
			repo = repository.NewPostgresPromoRepository(db)
			defer db.Close()
		}
	} else {
		repo = repository.NewInMemoryPromoRepository()
	}
	orderClient := client.NewDummyOrderClient()
	userClient := client.NewDummyUserClient()
	promoService := service.NewPromoService(repo, orderClient, userClient)

	// Setup router
	r := chi.NewRouter()
	
	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.NewPromoHandler(promoService)
	h.RegisterRoutes(r)

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
