package functional

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"furab-backend/services/auth-service/internal/model"
	"furab-backend/services/auth-service/internal/service"
)

var testDB *sql.DB

// ---------------------------------------------------------
// REAL DEPENDENCY IMPLEMENTATIONS FOR TEST
// ---------------------------------------------------------

type realUserService struct {
	db *sql.DB
}

func (s *realUserService) CreateUser(ctx context.Context, contact string) error {
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, "INSERT INTO users (id, contact) VALUES ($1, $2) ON CONFLICT (contact) DO NOTHING", id, contact)
	return err
}

func (s *realUserService) GetUser(ctx context.Context, contact string) (*model.User, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE contact = $1", contact).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // AuthService expects nil user if not found
		}
		return nil, err
	}
	return &model.User{ID: id}, nil
}

type realOTPService struct {
	db *sql.DB
}

func (s *realOTPService) GenerateOTP(ctx context.Context, contact string) error {
	code := "123456" // Default fixed code for test convenience
	_, err := s.db.ExecContext(ctx, "INSERT INTO otp (contact, code) VALUES ($1, $2) ON CONFLICT (contact) DO UPDATE SET code = EXCLUDED.code", contact, code)
	return err
}

func (s *realOTPService) VerifyOTP(ctx context.Context, contact, otpCode string) (bool, error) {
	var storedCode string
	err := s.db.QueryRowContext(ctx, "SELECT code FROM otp WHERE contact = $1", contact).Scan(&storedCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if storedCode == otpCode {
		return true, nil
	}
	return false, nil
}

type realTokenGenerator struct {
	db *sql.DB
}

func (s *realTokenGenerator) GenerateToken(userID string) (string, error) {
	token := "token-" + uuid.New().String()
	_, err := s.db.Exec("INSERT INTO sessions (user_id, token) VALUES ($1, $2)", userID, token)
	return token, err
}

func (s *realTokenGenerator) ValidateToken(token string) (bool, error) {
	var userID string
	err := s.db.QueryRow("SELECT user_id FROM sessions WHERE token = $1", token).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------
// TEST MAIN
// ---------------------------------------------------------

func TestMain(m *testing.M) {
	adminConn := "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=postgres sslmode=disable"
	if envDsn := os.Getenv("TEST_DB_DSN"); envDsn != "" {
		adminConn = envDsn
	}

	adminDB, err := sql.Open("postgres", adminConn)
	if err != nil {
		log.Fatalf("Could not connect to admin database: %v", err)
	}
	for i := 0; i < 30; i++ {
		if err = adminDB.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to database after 30 attempts: %v", err)
	}
	_, err = adminDB.Exec("CREATE DATABASE auth_test")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("Failed to create database: %v", err)
	}
	adminDB.Close()

	connStr := "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=auth_test sslmode=disable"
	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	for i := 0; i < 30; i++ {
		if err = testDB.Ping(); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	// Setup Schema
	_, err = testDB.Exec(`
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS otp;
		DROP TABLE IF EXISTS users;
		
		CREATE TABLE users (
			id VARCHAR(36) PRIMARY KEY,
			contact VARCHAR(255) UNIQUE NOT NULL
		);
		CREATE TABLE otp (
			contact VARCHAR(255) PRIMARY KEY,
			code VARCHAR(10) NOT NULL
		);
		CREATE TABLE sessions (
			user_id VARCHAR(36) NOT NULL,
			token VARCHAR(255) PRIMARY KEY
		);
	`)
	if err != nil {
		log.Fatalf("Could not create schema: %v", err)
	}

	code := m.Run()

	// Cleanup
	_, _ = testDB.Exec(`
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS otp;
		DROP TABLE IF EXISTS users;
	`)
	testDB.Close()
	os.Exit(code)
}

func clearDB() {
	_, _ = testDB.Exec("TRUNCATE TABLE sessions, otp, users")
}

// ---------------------------------------------------------
// TESTS
// ---------------------------------------------------------

func TestFunctional_FullFlow(t *testing.T) {
	clearDB()
	ctx := context.Background()

	userService := &realUserService{db: testDB}
	otpService := &realOTPService{db: testDB}
	tokenGenerator := &realTokenGenerator{db: testDB}
	svc := service.NewAuthService(userService, otpService, tokenGenerator)

	contact := "08123456789"
	expectedOTP := "123456"

	t.Run("Register", func(t *testing.T) {
		resReg, err := svc.Register(ctx, contact)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if resReg.Status != "success" {
			t.Fatalf("Register expected status success, got %s", resReg.Status)
		}

		// Verify User in DB
		var dbContact string
		err = testDB.QueryRow("SELECT contact FROM users WHERE contact = $1", contact).Scan(&dbContact)
		if err != nil {
			t.Fatalf("User not found in DB: %v", err)
		}

		// Verify OTP in DB
		var dbCode string
		err = testDB.QueryRow("SELECT code FROM otp WHERE contact = $1", contact).Scan(&dbCode)
		if err != nil {
			t.Fatalf("OTP not found in DB: %v", err)
		}
		if dbCode != expectedOTP {
			t.Fatalf("Expected OTP %s in DB, got %s", expectedOTP, dbCode)
		}
	})

	var generatedToken string
	t.Run("VerifyOTP", func(t *testing.T) {
		resLogin, err := svc.VerifyOTP(ctx, contact, expectedOTP)
		if err != nil {
			t.Fatalf("VerifyOTP failed: %v", err)
		}
		if resLogin.Status != "success" {
			t.Fatalf("VerifyOTP expected status success, got %s", resLogin.Status)
		}
		if resLogin.AccessToken == "" {
			t.Fatal("VerifyOTP expected token, got empty string")
		}
		generatedToken = resLogin.AccessToken

		// Verify token was stored in DB
		var dbUserID string
		err = testDB.QueryRow("SELECT user_id FROM sessions WHERE token = $1", generatedToken).Scan(&dbUserID)
		if err != nil {
			t.Fatalf("Token not found in DB: %v", err)
		}
	})

	t.Run("ValidateToken", func(t *testing.T) {
		resToken, err := svc.ValidateToken(ctx, generatedToken)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}
		if resToken.Status != "valid" {
			t.Fatalf("ValidateToken expected status valid, got %s", resToken.Status)
		}
	})
}

func TestFunctional_InvalidOTP(t *testing.T) {
	clearDB()
	ctx := context.Background()

	userService := &realUserService{db: testDB}
	otpService := &realOTPService{db: testDB}
	tokenGenerator := &realTokenGenerator{db: testDB}
	svc := service.NewAuthService(userService, otpService, tokenGenerator)

	contact := "08123456789"
	wrongOtp := "000000"

	// Arrange: need to register first to have the user and OTP in DB
	_, err := svc.Register(ctx, contact)
	if err != nil {
		t.Fatalf("Setup Register failed: %v", err)
	}

	t.Run("VerifyOTP with invalid OTP", func(t *testing.T) {
		res, err := svc.VerifyOTP(ctx, contact, wrongOtp)
		if res != nil {
			t.Fatalf("VerifyOTP expected nil response, got %v", res)
		}
		if !errors.Is(err, service.ErrOTPInvalid) {
			t.Fatalf("VerifyOTP expected error %v, got %v", service.ErrOTPInvalid, err)
		}
	})
}

func TestFunctional_InvalidToken(t *testing.T) {
	clearDB()
	ctx := context.Background()

	userService := &realUserService{db: testDB}
	otpService := &realOTPService{db: testDB}
	tokenGenerator := &realTokenGenerator{db: testDB}
	svc := service.NewAuthService(userService, otpService, tokenGenerator)

	invalidToken := "invalid-token-123"

	t.Run("ValidateToken with invalid token", func(t *testing.T) {
		res, err := svc.ValidateToken(ctx, invalidToken)
		if err != nil {
			t.Fatalf("ValidateToken unexpected error: %v", err)
		}
		if res.Status != "invalid" {
			t.Fatalf("ValidateToken expected status invalid, got %s", res.Status)
		}
	})
}

func TestFunctional_InvalidInput(t *testing.T) {
	clearDB()
	ctx := context.Background()

	userService := &realUserService{db: testDB}
	otpService := &realOTPService{db: testDB}
	tokenGenerator := &realTokenGenerator{db: testDB}
	svc := service.NewAuthService(userService, otpService, tokenGenerator)

	t.Run("Register with empty input", func(t *testing.T) {
		res, err := svc.Register(ctx, "")
		if res != nil {
			t.Fatalf("Register expected nil response, got %v", res)
		}
		if !errors.Is(err, service.ErrContactRequired) {
			t.Fatalf("Register expected error %v, got %v", service.ErrContactRequired, err)
		}
	})
}
