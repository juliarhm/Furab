// Package main is the entry point for review-service.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/review-service/internal/handler"
	"furab-backend/services/review-service/internal/repository"
	"furab-backend/services/review-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load("review-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting review-service", "port", cfg.ServerPort)

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize dependencies
	repo := repository.NewReviewRepository(db)
	svc := service.NewReviewService(repo)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.NewReviewHandler(svc)
	r.Mount("/reviews", h.Routes())

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
