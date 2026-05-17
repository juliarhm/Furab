//go:build functional
// +build functional

// Package functional contains functional tests for the user service.
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

	"furab-backend/services/user-service/internal/model"
	"furab-backend/services/user-service/internal/repository"
	"furab-backend/services/user-service/internal/service"

	_ "github.com/lib/pq"
)

var (
	testDB   *sql.DB
	testRepo repository.UserRepository
	testSvc  service.UserService
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
	dbName := getEnvOrDefault("DB_NAME", "user_service")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	if envDsn := os.Getenv("TEST_DB_DSN"); envDsn != "" {
		dsn = envDsn
	}

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	for i := 0; i < 30; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}

	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	testRepo = repository.NewPostgresUserRepository(testDB)
	testSvc = service.NewUserService(testRepo)

	code := m.Run()

	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			user_id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			phone VARCHAR(20) NOT NULL UNIQUE,
			email VARCHAR(255) UNIQUE,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS users")
}

func cleanupUsers() {
	testDB.Exec("DELETE FROM users")
}

// --- Functional Test Cases ---

func TestFunctional_CreateAndGetUser(t *testing.T) {
	cleanupUsers()
	ctx := context.Background()

	req := &model.CreateUserRequest{
		UserID: "func-user-001",
		Name:   "Andi",
		Phone:  "081234567890",
		Email:  "andi@example.com",
	}

	resp, err := testSvc.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if resp.Message == "" {
		t.Errorf("Expected message, got empty")
	}
	if resp.UserID == "" {
		t.Error("Expected non-empty UserID")
	}

	user, err := testSvc.GetUser(ctx, resp.UserID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, user.Name)
	}
	if user.Phone != req.Phone {
		t.Errorf("Expected phone %s, got %s", req.Phone, user.Phone)
	}
}

func TestFunctional_DuplicateUser(t *testing.T) {
	cleanupUsers()
	ctx := context.Background()

	req := &model.CreateUserRequest{
		UserID: "func-user-002",
		Name:   "Budi",
		Phone:  "0899999999",
		Email:  "budi@example.com",
	}

	_, err := testSvc.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("First create should succeed, got: %v", err)
	}

	_, err = testSvc.CreateUser(ctx, req)
	if err == nil {
		t.Fatal("Second create with same phone/email should fail")
	}
}

func TestFunctional_UpdateUser(t *testing.T) {
	cleanupUsers()
	ctx := context.Background()

	// 1. Create User
	req := &model.CreateUserRequest{
		UserID: "func-user-003",
		Name:   "Citra",
		Phone:  "0877777777",
		Email:  "citra1@example.com",
	}
	resp, err := testSvc.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// 2. Update User
	updateReq := &model.UpdateUserRequest{
		Name:  "Citra Lestari",
		Phone: "0877777778",
		Email: "citra@example.com",
	}
	updateResp, err := testSvc.UpdateUser(ctx, resp.UserID, updateReq)
	if err != nil {
		t.Fatalf("Update user failed: %v", err)
	}
	if updateResp.Message == "" {
		t.Errorf("Expected message, got empty")
	}

	// 3. Verify in DB
	user, err := testSvc.GetUser(ctx, resp.UserID)
	if err != nil {
		t.Fatalf("Get user failed: %v", err)
	}
	if user.Name != updateReq.Name {
		t.Errorf("Expected updated name %s, got %s", updateReq.Name, user.Name)
	}
	if user.Email != updateReq.Email {
		t.Errorf("Expected updated email %s, got %s", updateReq.Email, user.Email)
	}
	if user.Phone != updateReq.Phone {
		t.Errorf("Expected updated phone %s, got %s", updateReq.Phone, user.Phone)
	}
}
