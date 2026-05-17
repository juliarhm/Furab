// Package main is the entry point for the Furab API Gateway.
// The API Gateway acts as a reverse proxy, routing requests to the appropriate microservice.
package main

import (
	"log"
	"net/http"
	"time"

	"furab-backend/gateway/api-gateway/internal/router"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load("api-gateway")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting api-gateway", "port", cfg.ServerPort)

	r := router.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	logger.Info("api-gateway listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("api-gateway error: %v", err)
	}
}
