// Package main is the entry point for rating-service.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/rating-service/internal/handler"
	"furab-backend/services/rating-service/internal/repository"
	"furab-backend/services/rating-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load("rating-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting rating-service", "port", cfg.ServerPort)

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize dependencies
	repo := repository.NewRatingRepository(db)
	svc := service.NewRatingService(repo)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.NewRatingHandler(svc)
	r.Mount("/ratings", h.Routes())

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
