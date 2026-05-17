// Package main is the entry point for location-service.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"furab-backend/services/location-service/internal/handler"
	"furab-backend/services/location-service/internal/repository"
	"furab-backend/services/location-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

// dummyDriverClient implements service.DriverServiceClient for standalone testing/running
type dummyDriverClient struct{}

func (d *dummyDriverClient) ValidateDriver(ctx context.Context, driverID string) (bool, error) {
	return true, nil // Always valid for now
}

func main() {
	cfg := config.Load("location-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting location-service", "port", cfg.ServerPort)

	// Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	logger.Info("connected to redis", "addr", cfg.RedisAddr())

	// Setup dependencies
	repo := repository.NewRedisLocationRepository(rdb)
	driverClient := &dummyDriverClient{}
	svc := service.NewLocationService(repo, driverClient)
	h := handler.NewLocationHandler(svc)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h.RegisterRoutes(r)

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
