// Package main is the entry point for merchant-service.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/merchant-service/internal/handler"
	"furab-backend/services/merchant-service/internal/repository"
	"furab-backend/services/merchant-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load("merchant-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting merchant-service", "port", cfg.ServerPort)

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize dependencies
	repo := repository.NewMerchantRepository(db)
	svc := service.NewMerchantService(repo)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.NewMerchantHandler(svc)
	r.Mount("/merchants", h.Routes())

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
