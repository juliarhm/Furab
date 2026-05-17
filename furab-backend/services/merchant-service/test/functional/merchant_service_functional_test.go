//go:build functional
// +build functional

package functional

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"furab-backend/services/merchant-service/internal/model"
	"furab-backend/services/merchant-service/internal/repository"
	"furab-backend/services/merchant-service/internal/service"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	defaultConn := "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=postgres sslmode=disable"
	adminDB, err := sql.Open("postgres", defaultConn)
	if err != nil {
		log.Fatalf("Failed to open default database: %v", err)
	}
	for i := 0; i < 30; i++ {
		err = adminDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for default database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to default database after 30 attempts: %v", err)
	}
	_, err = adminDB.Exec("CREATE DATABASE merchant_service")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("Failed to create database: %v", err)
	}
	adminDB.Close()

	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=merchant_service sslmode=disable"
	}
	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to initialize database driver: %v", err)
	}
	for i := 0; i < 30; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for merchant_service database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to merchant_service database after 30 attempts: %v", err)
	}
	log.Println("Database connection to merchant_service established successfully.")

	setupSchema()
	code := m.Run()
	teardownSchema()
	testDB.Close()
	os.Exit(code)
}

func setupSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS merchants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		address TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		rating DOUBLE PRECISION NOT NULL DEFAULT 0,
		is_open BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL
	);`
	if _, err := testDB.Exec(query); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}
}

func teardownSchema() {
	if _, err := testDB.Exec("DROP TABLE IF EXISTS merchants"); err != nil {
		log.Printf("Failed to teardown schema: %v", err)
	}
}

func cleanupMerchants() {
	if _, err := testDB.Exec("DELETE FROM merchants"); err != nil {
		log.Fatalf("Failed to cleanup merchants: %v", err)
	}
}

func merchantRequest() *model.Merchant {
	return &model.Merchant{
		Name:        "Warung Makan Sederhana",
		Address:     "Jl. Sudirman No. 1, Jakarta",
		Description: "Warung makan dengan menu nusantara",
		IsOpen:      true,
	}
}

func TestFunctional_Merchant_CreateAndSearch(t *testing.T) {
	cleanupMerchants()
	repo := repository.NewMerchantRepository(testDB)
	svc := service.NewMerchantService(repo)
	ctx := context.Background()

	req := merchantRequest()
	err := svc.CreateMerchant(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID)

	t.Run("SearchByName", func(t *testing.T) {
		results, total, err := svc.SearchMerchants(ctx, model.SearchMerchantRequest{Query: "Warung", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
		assert.Equal(t, req.Name, results[0].Name)
		assert.Equal(t, req.Address, results[0].Address)
	})

	t.Run("SearchByDescription", func(t *testing.T) {
		results, total, err := svc.SearchMerchants(ctx, model.SearchMerchantRequest{Query: "nusantara", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})
}

func TestFunctional_Merchant_SearchNoMatch(t *testing.T) {
	cleanupMerchants()
	repo := repository.NewMerchantRepository(testDB)
	svc := service.NewMerchantService(repo)
	ctx := context.Background()

	err := svc.CreateMerchant(ctx, merchantRequest())
	assert.NoError(t, err)

	results, total, err := svc.SearchMerchants(ctx, model.SearchMerchantRequest{Query: "Sushi Restaurant", Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, results)
}

func TestFunctional_Merchant_EdgeCases(t *testing.T) {
	cleanupMerchants()
	repo := repository.NewMerchantRepository(testDB)
	svc := service.NewMerchantService(repo)
	ctx := context.Background()

	t.Run("CreateWithEmptyName", func(t *testing.T) {
		req := merchantRequest()
		req.Name = ""
		err := svc.CreateMerchant(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant name is required")
	})

	t.Run("CreateWithEmptyAddress", func(t *testing.T) {
		req := merchantRequest()
		req.Address = ""
		err := svc.CreateMerchant(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant address is required")
	})

	t.Run("GetWithEmptyID", func(t *testing.T) {
		_, err := svc.GetMerchant(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant id is required")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := svc.GetMerchant(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant not found")
	})

	t.Run("DeleteWithEmptyID", func(t *testing.T) {
		err := svc.DeleteMerchant(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant id is required")
	})
}

func TestFunctional_Merchant_FullLifecycle(t *testing.T) {
	cleanupMerchants()
	repo := repository.NewMerchantRepository(testDB)
	svc := service.NewMerchantService(repo)
	ctx := context.Background()

	req := merchantRequest()
	err := svc.CreateMerchant(ctx, req)
	assert.NoError(t, err)

	created, err := svc.GetMerchant(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Warung Makan Sederhana", created.Name)

	err = svc.UpdateMerchant(ctx, &model.Merchant{ID: req.ID, Name: "Warung Premium", Address: "Jl. Gatot Subroto", IsOpen: false})
	assert.NoError(t, err)

	updated, err := svc.GetMerchant(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Warung Premium", updated.Name)
	assert.False(t, updated.IsOpen)

	err = svc.DeleteMerchant(ctx, req.ID)
	assert.NoError(t, err)

	_, err = svc.GetMerchant(ctx, req.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merchant not found")
}
