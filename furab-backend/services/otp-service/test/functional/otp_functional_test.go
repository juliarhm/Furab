//go:build functional
// +build functional

// Package functional contains functional tests for the otp service.
// Functional tests access a REAL PostgreSQL database.
//
// Run with: go test ./test/functional/... -v -tags=functional
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/otp-service/internal/model"
	"furab-backend/services/otp-service/internal/repository"
	"furab-backend/services/otp-service/internal/service"

	_ "github.com/lib/pq"
)

var (
	testDB   *sql.DB
	testRepo repository.OTPRepository
	testSvc  service.OTPService
)

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "otp_service")

	// Step 1: Auto-create database
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)
	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		log.Fatalf("Failed to connect to admin database: %v", err)
	}
	for i := 0; i < 30; i++ {
		if err = adminDB.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}
	_, _ = adminDB.Exec("CREATE DATABASE " + dbName)
	adminDB.Close()

	// Step 2: Connect to target database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	if envDsn := os.Getenv("TEST_DB_DSN"); envDsn != "" {
		dsn = envDsn
	}

	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	for i := 0; i < 30; i++ {
		if err = testDB.Ping(); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}

	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	testRepo = repository.NewPostgresOTPRepository(testDB)
	testSvc = service.NewOTPService(testRepo)

	code := m.Run()

	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS otps (
			otp_id VARCHAR(36) PRIMARY KEY,
			target VARCHAR(255) NOT NULL,
			otp_code VARCHAR(10) NOT NULL,
			expired_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_otps_target ON otps(target);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS otps")
}

func cleanupOTPs() {
	testDB.Exec("DELETE FROM otps")
}

// --- Functional Test Cases ---

func TestFunctional_OTPLifecycle(t *testing.T) {
	cleanupOTPs()
	ctx := context.Background()
	target := "081234567890"

	// 1. Generate OTP
	_, err := testSvc.GenerateOTP(ctx, &model.GenerateOTPRequest{Target: target})
	if err != nil {
		t.Fatalf("Failed to generate OTP: %v", err)
	}

	// Verify in DB that it exists
	var otpCode string
	err = testDB.QueryRow("SELECT otp_code FROM otps WHERE target = $1", target).Scan(&otpCode)
	if err != nil {
		t.Fatalf("OTP not found in DB: %v", err)
	}
	if len(otpCode) != 6 {
		t.Errorf("Expected 6-digit OTP, got %s", otpCode)
	}

	// 2. Verify OTP with wrong code
	_, err = testSvc.VerifyOTP(ctx, &model.VerifyOTPRequest{Target: target, OTPCode: "000000"})
	if err == nil {
		t.Fatalf("Expected error for wrong OTP code, but got nil")
	}
	t.Logf("Correctly got error for wrong OTP code: %v", err)

	// Verify DB record still exists after wrong attempt
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM otps WHERE target = $1", target).Scan(&count)
	if count == 0 {
		t.Error("OTP should not be deleted after failed attempt")
	}

	// 3. Verify OTP with correct code
	respVer, err := testSvc.VerifyOTP(ctx, &model.VerifyOTPRequest{Target: target, OTPCode: otpCode})
	if err != nil {
		t.Fatalf("VerifyOTP with correct code failed: %v", err)
	}
	if respVer.Status != "valid" {
		t.Error("Expected valid=true for correct OTP code")
	}

	// 4. Ensure OTP is deleted after successful verification
	testDB.QueryRow("SELECT COUNT(*) FROM otps WHERE target = $1", target).Scan(&count)
	if count != 0 {
		t.Error("OTP must be deleted after successful verification")
	}
}
