//go:build functional
// +build functional

// Package functional contains functional tests for review-service.
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

	"furab-backend/services/review-service/internal/model"
	"furab-backend/services/review-service/internal/repository"
	"furab-backend/services/review-service/internal/service"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB

// TestMain sets up the database environment for functional testing.
func TestMain(m *testing.M) {
	// =========================================================================
	// Step 1: Ensure the review_service database exists.
	// Connect to the default 'postgres' database first to create it if needed.
	// =========================================================================
	defaultConn := "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=postgres sslmode=disable"
	adminDB, err := sql.Open("postgres", defaultConn)
	if err != nil {
		log.Fatalf("Failed to open connection to default database: %v", err)
	}

	// Retry ping to default database (Docker container might still be starting).
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

	// Attempt to create the review_service database (ignore if already exists).
	_, err = adminDB.Exec("CREATE DATABASE review_service")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Fatalf("Failed to create database review_service: %v", err)
		}
		log.Println("Database review_service already exists, skipping creation.")
	} else {
		log.Println("Database review_service created successfully.")
	}
	adminDB.Close()

	// =========================================================================
	// Step 2: Connect to the review_service database.
	// =========================================================================
	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=review_service sslmode=disable"
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to initialize database driver: %v", err)
	}

	// Retry Ping to wait for the target database to become available.
	for i := 0; i < 30; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for review_service database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to review_service database after 30 attempts: %v", err)
	}

	log.Println("Database connection to review_service established successfully.")

	// =========================================================================
	// Step 3: Run tests with schema setup/teardown.
	// =========================================================================
	setupSchema()
	code := m.Run()
	teardownSchema()
	testDB.Close()
	os.Exit(code)
}

// setupSchema creates the reviews table if it does not exist.
func setupSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS reviews (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		merchant_id TEXT NOT NULL,
		order_id TEXT NOT NULL,
		rating INTEGER NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		image_url TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL
	);
	`
	_, err := testDB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}
}

// teardownSchema drops the reviews table after all tests complete.
func teardownSchema() {
	query := `DROP TABLE IF EXISTS reviews`
	_, err := testDB.Exec(query)
	if err != nil {
		log.Printf("Failed to teardown schema: %v", err)
	}
}

// cleanupReviews removes all rows from reviews for test isolation.
func cleanupReviews() {
	_, err := testDB.Exec("DELETE FROM reviews")
	if err != nil {
		log.Fatalf("Failed to cleanup reviews: %v", err)
	}
}

// reviewRequest returns a sample Review for testing.
func reviewRequest() *model.Review {
	return &model.Review{
		UserID:     "user-001",
		MerchantID: "merchant-001",
		OrderID:    "order-001",
		Rating:     5,
		Comment:    "Makanan enak dan cepat!",
		ImageURL:   "https://example.com/review-photo.jpg",
	}
}

// =============================================================================
// Test Cases
// =============================================================================

// TestFunctional_Review_CreateAndSearch tests creating a review and searching for it.
func TestFunctional_Review_CreateAndSearch(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	// Create
	req := reviewRequest()
	err := svc.CreateReview(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID, "ID should be auto-generated after create")
	assert.False(t, req.CreatedAt.IsZero(), "CreatedAt should be set after create")
	assert.False(t, req.UpdatedAt.IsZero(), "UpdatedAt should be set after create")

	// Search by MerchantID
	t.Run("SearchByMerchantID", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "merchant-001",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
		assert.Equal(t, req.UserID, results[0].UserID)
		assert.Equal(t, req.MerchantID, results[0].MerchantID)
		assert.Equal(t, req.OrderID, results[0].OrderID)
		assert.Equal(t, req.Rating, results[0].Rating)
		assert.Equal(t, req.Comment, results[0].Comment)
		assert.Equal(t, req.ImageURL, results[0].ImageURL)
	})

	// Search by UserID
	t.Run("SearchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			UserID: "user-001",
			Limit:  10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
	})

	// Search by MerchantID + UserID combined
	t.Run("SearchCombined", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "merchant-001",
			UserID:     "user-001",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})
}

// TestFunctional_Review_SearchNoMatch verifies that searching with
// criteria that match no records returns empty results and zero total.
func TestFunctional_Review_SearchNoMatch(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	// Seed one review so the table is not empty.
	req := reviewRequest()
	err := svc.CreateReview(ctx, req)
	assert.NoError(t, err)

	t.Run("NoMatchByMerchantID", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "non-existent-merchant",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			UserID: "non-existent-user",
			Limit:  10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchCombined", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "merchant-001",
			UserID:     "non-existent-user",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})
}

// TestFunctional_Review_EdgeCases tests validation errors and edge cases.
func TestFunctional_Review_EdgeCases(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	t.Run("CreateWithEmptyUserID", func(t *testing.T) {
		req := reviewRequest()
		req.UserID = ""
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user id is required")
	})

	t.Run("CreateWithEmptyMerchantID", func(t *testing.T) {
		req := reviewRequest()
		req.MerchantID = ""
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant id is required")
	})

	t.Run("CreateWithEmptyOrderID", func(t *testing.T) {
		req := reviewRequest()
		req.OrderID = ""
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order id is required")
	})

	t.Run("CreateWithRatingTooLow", func(t *testing.T) {
		req := reviewRequest()
		req.Rating = 0
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating must be between 1 and 5")
	})

	t.Run("CreateWithRatingTooHigh", func(t *testing.T) {
		req := reviewRequest()
		req.Rating = 6
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating must be between 1 and 5")
	})

	t.Run("CreateWithNegativeRating", func(t *testing.T) {
		req := reviewRequest()
		req.Rating = -1
		err := svc.CreateReview(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating must be between 1 and 5")
	})

	t.Run("GetWithEmptyID", func(t *testing.T) {
		_, err := svc.GetReview(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "review id is required")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := svc.GetReview(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "review not found")
	})

	t.Run("UpdateWithEmptyID", func(t *testing.T) {
		err := svc.UpdateReview(ctx, &model.Review{ID: "", Rating: 3})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "review id is required")
	})

	t.Run("DeleteWithEmptyID", func(t *testing.T) {
		err := svc.DeleteReview(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "review id is required")
	})
}

// TestFunctional_Review_MultipleEntries tests creating multiple reviews
// and verifying search filters return the correct subset.
func TestFunctional_Review_MultipleEntries(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	// Seed diverse review entries.
	entries := []*model.Review{
		{UserID: "user-001", MerchantID: "merchant-001", OrderID: "ord-1", Rating: 5, Comment: "Excellent!", ImageURL: ""},
		{UserID: "user-001", MerchantID: "merchant-002", OrderID: "ord-2", Rating: 4, Comment: "Good", ImageURL: ""},
		{UserID: "user-002", MerchantID: "merchant-001", OrderID: "ord-3", Rating: 3, Comment: "Average", ImageURL: ""},
		{UserID: "user-003", MerchantID: "merchant-003", OrderID: "ord-4", Rating: 2, Comment: "Below average", ImageURL: ""},
	}
	for _, e := range entries {
		err := svc.CreateReview(ctx, e)
		assert.NoError(t, err)
		assert.NotEmpty(t, e.ID)
	}

	t.Run("FilterByMerchant001", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "merchant-001",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
	})

	t.Run("FilterByUser001", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			UserID: "user-001",
			Limit:  10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
	})

	t.Run("FilterCombined_User001_Merchant001", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			MerchantID: "merchant-001",
			UserID:     "user-001",
			Limit:      10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, "Excellent!", results[0].Comment)
	})

	t.Run("AllEntriesNoFilter", func(t *testing.T) {
		results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
			Limit: 10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, results, 4)
	})
}

// TestFunctional_Review_GetByID verifies direct retrieval of a single review.
func TestFunctional_Review_GetByID(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	// Create
	req := reviewRequest()
	err := svc.CreateReview(ctx, req)
	assert.NoError(t, err)

	// Get by ID
	retrieved, err := svc.GetReview(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, req.ID, retrieved.ID)
	assert.Equal(t, req.UserID, retrieved.UserID)
	assert.Equal(t, req.MerchantID, retrieved.MerchantID)
	assert.Equal(t, req.OrderID, retrieved.OrderID)
	assert.Equal(t, req.Rating, retrieved.Rating)
	assert.Equal(t, req.Comment, retrieved.Comment)
	assert.Equal(t, req.ImageURL, retrieved.ImageURL)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())

	// Verify non-existent returns error
	_, err = svc.GetReview(ctx, "does-not-exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "review not found")
}

// TestFunctional_Review_FullLifecycle tests the complete Create → Get → Update → Delete flow.
func TestFunctional_Review_FullLifecycle(t *testing.T) {
	cleanupReviews()
	repo := repository.NewReviewRepository(testDB)
	svc := service.NewReviewService(repo)
	ctx := context.Background()

	// 1. Create
	req := reviewRequest()
	err := svc.CreateReview(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID)

	// 2. Get — verify initial state
	created, err := svc.GetReview(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, created.Rating)
	assert.Equal(t, "Makanan enak dan cepat!", created.Comment)
	assert.Equal(t, "https://example.com/review-photo.jpg", created.ImageURL)

	// 3. Update
	err = svc.UpdateReview(ctx, &model.Review{
		ID:       req.ID,
		Rating:   3,
		Comment:  "Ternyata biasa saja.",
		ImageURL: "https://example.com/updated-photo.jpg",
	})
	assert.NoError(t, err)

	// 4. Verify Update
	updated, err := svc.GetReview(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, updated.Rating)
	assert.Equal(t, "Ternyata biasa saja.", updated.Comment)
	assert.Equal(t, "https://example.com/updated-photo.jpg", updated.ImageURL)
	// MerchantID and OrderID should remain unchanged
	assert.Equal(t, req.MerchantID, updated.MerchantID)
	assert.Equal(t, req.OrderID, updated.OrderID)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt) || updated.UpdatedAt.Equal(created.UpdatedAt))

	// 5. Search — should find the updated review
	results, total, err := svc.SearchReviews(ctx, model.SearchReviewRequest{
		MerchantID: req.MerchantID,
		Limit:      10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 3, results[0].Rating)

	// 6. Delete
	err = svc.DeleteReview(ctx, req.ID)
	assert.NoError(t, err)

	// 7. Verify Deleted
	_, err = svc.GetReview(ctx, req.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "review not found")

	// 8. Search — should return empty
	results, total, err = svc.SearchReviews(ctx, model.SearchReviewRequest{
		MerchantID: req.MerchantID,
		Limit:      10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, results)
}
