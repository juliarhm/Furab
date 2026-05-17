// Package main is the entry point for auth-service.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"furab-backend/services/auth-service/internal/handler"
	"furab-backend/services/auth-service/internal/model"
	"furab-backend/services/auth-service/internal/repository"
	"furab-backend/services/auth-service/internal/service"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ---------------------------------------------------------
// PLACEHOLDER DEPENDENCY IMPLEMENTATIONS
// In a real microservice architecture, these would be gRPC clients
// communicating with user-service and otp-service.
// ---------------------------------------------------------

type placeholderUserService struct{}

func (s *placeholderUserService) CreateUser(ctx context.Context, contact string) error {
	return nil
}

func (s *placeholderUserService) GetUser(ctx context.Context, contact string) (*model.User, error) {
	return &model.User{ID: uuid.New().String(), Contact: contact}, nil
}

type placeholderOTPService struct{}

func (s *placeholderOTPService) GenerateOTP(ctx context.Context, contact string) error {
	return nil
}

func (s *placeholderOTPService) VerifyOTP(ctx context.Context, contact, otpCode string) (bool, error) {
	return true, nil
}

type placeholderTokenGenerator struct {
	repo repository.AuthRepository
}

func (s *placeholderTokenGenerator) GenerateToken(userID string) (string, error) {
	token := "jwt-" + uuid.New().String()
	session := &model.Session{
		SessionID: uuid.New().String(),
		UserID:    userID,
		Token:     token,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}
	err := s.repo.SaveSession(context.Background(), session)
	return token, err
}

func (s *placeholderTokenGenerator) ValidateToken(token string) (bool, error) {
	_, err := s.repo.GetSession(context.Background(), token)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------
// MAIN
// ---------------------------------------------------------

func main() {
	cfg := config.Load("auth-service")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting auth-service", "port", cfg.ServerPort)

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Wire dependencies
	repo := repository.NewPostgresAuthRepository(db)

	userService := &placeholderUserService{}
	otpService := &placeholderOTPService{}
	tokenGenerator := &placeholderTokenGenerator{repo: repo}

	svc := service.NewAuthService(userService, otpService, tokenGenerator)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	r.Get("/health", handler.HealthCheck)
	h := handler.NewAuthHandler(svc)
	h.RegisterRoutes(r)

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
