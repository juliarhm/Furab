//go:build functional
// +build functional

// Package functional contains functional tests for menu-service.
// These tests access a real PostgreSQL database running in Docker.
// Run with: go test ./test/functional/... -v -tags=functional
package functional

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"furab-backend/services/menu-service/internal/model"
	"furab-backend/services/menu-service/internal/repository"
	"furab-backend/services/menu-service/internal/service"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB

// TestMain sets up the database environment for functional testing.
func TestMain(m *testing.M) {
	// =========================================================================
	// Step 1: Ensure the menu_service database exists.
	// =========================================================================
	defaultConn := "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=postgres sslmode=disable"
	adminDB, err := sql.Open("postgres", defaultConn)
	if err != nil {
		log.Fatalf("Failed to open connection to default database: %v", err)
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

	_, err = adminDB.Exec("CREATE DATABASE menu_service")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Fatalf("Failed to create database menu_service: %v", err)
		}
		log.Println("Database menu_service already exists, skipping creation.")
	} else {
		log.Println("Database menu_service created successfully.")
	}
	adminDB.Close()

	// =========================================================================
	// Step 2: Connect to the menu_service database.
	// =========================================================================
	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=menu_service sslmode=disable"
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
		log.Printf("Waiting for menu_service database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to menu_service database after 30 attempts: %v", err)
	}

	log.Println("Database connection to menu_service established successfully.")

	// =========================================================================
	// Step 3: Run tests with schema setup/teardown.
	// =========================================================================
	setupSchema()
	code := m.Run()
	teardownSchema()
	testDB.Close()
	os.Exit(code)
}

func setupSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS menus (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		price DOUBLE PRECISION NOT NULL DEFAULT 0,
		category TEXT NOT NULL DEFAULT '',
		image_url TEXT NOT NULL DEFAULT '',
		is_available BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL
	);
	`
	_, err := testDB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}
}

func teardownSchema() {
	_, err := testDB.Exec("DROP TABLE IF EXISTS menus")
	if err != nil {
		log.Printf("Failed to teardown schema: %v", err)
	}
}

func cleanupMenus() {
	_, err := testDB.Exec("DELETE FROM menus")
	if err != nil {
		log.Fatalf("Failed to cleanup menus: %v", err)
	}
}

func menuRequest() *model.Menu {
	return &model.Menu{
		MerchantID:  "merchant-001",
		Name:        "Nasi Goreng Spesial",
		Description: "Nasi goreng dengan telur dan ayam",
		Price:       25000,
		Category:    "Makanan",
		ImageURL:    "https://example.com/nasi-goreng.jpg",
		IsAvailable: true,
	}
}

// =============================================================================
// Test Cases
// =============================================================================

// TestFunctional_Menu_CreateAndSearch tests creating a menu and searching for it.
func TestFunctional_Menu_CreateAndSearch(t *testing.T) {
	cleanupMenus()
	repo := repository.NewMenuRepository(testDB)
	svc := service.NewMenuService(repo)
	ctx := context.Background()

	// Create
	req := menuRequest()
	err := svc.CreateMenu(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID, "ID should be auto-generated after create")
	assert.False(t, req.CreatedAt.IsZero())
	assert.False(t, req.UpdatedAt.IsZero())

	// Search by MerchantID
	t.Run("SearchByMerchantID", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			MerchantID: "merchant-001",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
		assert.Equal(t, req.MerchantID, results[0].MerchantID)
		assert.Equal(t, req.Name, results[0].Name)
		assert.Equal(t, req.Description, results[0].Description)
		assert.Equal(t, req.Price, results[0].Price)
		assert.Equal(t, req.Category, results[0].Category)
		assert.Equal(t, req.ImageURL, results[0].ImageURL)
		assert.True(t, results[0].IsAvailable)
	})

	// Search by Query (name ILIKE)
	t.Run("SearchByQuery", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			Query: "Goreng",
			Limit: 10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})

	// Search by Category
	t.Run("SearchByCategory", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			Category: "Makanan",
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})
}

// TestFunctional_Menu_SearchNoMatch verifies empty results for non-matching criteria.
func TestFunctional_Menu_SearchNoMatch(t *testing.T) {
	cleanupMenus()
	repo := repository.NewMenuRepository(testDB)
	svc := service.NewMenuService(repo)
	ctx := context.Background()

	// Seed one entry so the table is not empty.
	req := menuRequest()
	err := svc.CreateMenu(ctx, req)
	assert.NoError(t, err)

	t.Run("NoMatchByMerchantID", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			MerchantID: "non-existent-merchant",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByQuery", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			Query: "Sushi",
			Limit: 10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByCategory", func(t *testing.T) {
		results, total, err := svc.SearchMenus(ctx, model.SearchMenuRequest{
			Category: "Dessert",
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})
}

// TestFunctional_Menu_EdgeCases tests validation errors and edge cases.
func TestFunctional_Menu_EdgeCases(t *testing.T) {
	cleanupMenus()
	repo := repository.NewMenuRepository(testDB)
	svc := service.NewMenuService(repo)
	ctx := context.Background()

	t.Run("CreateWithEmptyMerchantID", func(t *testing.T) {
		req := menuRequest()
		req.MerchantID = ""
		err := svc.CreateMenu(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant id is required")
	})

	t.Run("CreateWithEmptyName", func(t *testing.T) {
		req := menuRequest()
		req.Name = ""
		err := svc.CreateMenu(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "menu name is required")
	})

	t.Run("CreateWithZeroPrice", func(t *testing.T) {
		req := menuRequest()
		req.Price = 0
		err := svc.CreateMenu(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "price must be greater than zero")
	})

	t.Run("CreateWithNegativePrice", func(t *testing.T) {
		req := menuRequest()
		req.Price = -5000
		err := svc.CreateMenu(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "price must be greater than zero")
	})

	t.Run("GetWithEmptyID", func(t *testing.T) {
		_, err := svc.GetMenu(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "menu id is required")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := svc.GetMenu(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "menu not found")
	})

	t.Run("DeleteWithEmptyID", func(t *testing.T) {
		err := svc.DeleteMenu(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "menu id is required")
	})
}

// TestFunctional_Menu_FullLifecycle tests the complete Create → Get → Update → Delete flow.
func TestFunctional_Menu_FullLifecycle(t *testing.T) {
	cleanupMenus()
	repo := repository.NewMenuRepository(testDB)
	svc := service.NewMenuService(repo)
	ctx := context.Background()

	// 1. Create
	req := menuRequest()
	err := svc.CreateMenu(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID)

	// 2. Get
	created, err := svc.GetMenu(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Nasi Goreng Spesial", created.Name)
	assert.Equal(t, float64(25000), created.Price)

	// 3. Update
	err = svc.UpdateMenu(ctx, &model.Menu{
		ID:          req.ID,
		Name:        "Nasi Goreng Premium",
		Price:       35000,
		Category:    "Makanan Spesial",
		IsAvailable: false,
	})
	assert.NoError(t, err)

	// 4. Verify Update
	updated, err := svc.GetMenu(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Nasi Goreng Premium", updated.Name)
	assert.Equal(t, float64(35000), updated.Price)
	assert.Equal(t, "Makanan Spesial", updated.Category)
	assert.False(t, updated.IsAvailable)

	// 5. Delete
	err = svc.DeleteMenu(ctx, req.ID)
	assert.NoError(t, err)

	// 6. Verify Deleted
	_, err = svc.GetMenu(ctx, req.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "menu not found")
}
