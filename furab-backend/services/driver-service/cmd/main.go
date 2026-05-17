// Package main is the entry point for driver-service.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"furab-backend/services/driver-service/internal/handler"
	"furab-backend/services/driver-service/internal/repository"
	"furab-backend/services/driver-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load("driver-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting driver-service", "port", cfg.ServerPort)

	db, err := sql.Open("pgx", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Wire dependencies
	repo := repository.NewPostgresDriverRepository(db)
	svc := service.NewDriverService(repo)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	r.Get("/health", handler.HealthCheck)
	h := handler.NewDriverHandler(svc)
	h.RegisterRoutes(r)

	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
