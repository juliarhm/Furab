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

	"furab-backend/services/rating-service/internal/model"
	"furab-backend/services/rating-service/internal/repository"
	"furab-backend/services/rating-service/internal/service"

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
	_, err = adminDB.Exec("CREATE DATABASE rating_service")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("Failed to create database: %v", err)
	}
	adminDB.Close()

	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=rating_service sslmode=disable"
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
		log.Printf("Waiting for rating_service database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to rating_service database after 30 attempts: %v", err)
	}
	log.Println("Database connection to rating_service established successfully.")

	setupSchema()
	code := m.Run()
	teardownSchema()
	testDB.Close()
	os.Exit(code)
}

func setupSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS ratings (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		target_type TEXT NOT NULL,
		score INTEGER NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL
	);`
	if _, err := testDB.Exec(query); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}
}

func teardownSchema() {
	if _, err := testDB.Exec("DROP TABLE IF EXISTS ratings"); err != nil {
		log.Printf("Failed to teardown schema: %v", err)
	}
}

func cleanupRatings() {
	if _, err := testDB.Exec("DELETE FROM ratings"); err != nil {
		log.Fatalf("Failed to cleanup ratings: %v", err)
	}
}

func ratingRequest() *model.Rating {
	return &model.Rating{
		UserID:     "user-123",
		TargetID:   "merchant-456",
		TargetType: "merchant",
		Score:      5,
		Comment:    "Great food and service!",
	}
}

func TestFunctional_Rating_CreateAndSearch(t *testing.T) {
	cleanupRatings()
	repo := repository.NewRatingRepository(testDB)
	svc := service.NewRatingService(repo)
	ctx := context.Background()

	req := ratingRequest()
	err := svc.CreateRating(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID)
	assert.False(t, req.CreatedAt.IsZero())

	t.Run("SearchByTargetID", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{TargetID: "merchant-456", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
		assert.Equal(t, req.UserID, results[0].UserID)
		assert.Equal(t, req.TargetID, results[0].TargetID)
		assert.Equal(t, req.TargetType, results[0].TargetType)
		assert.Equal(t, req.Score, results[0].Score)
		assert.Equal(t, req.Comment, results[0].Comment)
	})

	t.Run("SearchByTargetType", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{TargetType: "merchant", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})

	t.Run("SearchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{UserID: "user-123", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})
}

func TestFunctional_Rating_SearchNoMatch(t *testing.T) {
	cleanupRatings()
	repo := repository.NewRatingRepository(testDB)
	svc := service.NewRatingService(repo)
	ctx := context.Background()

	err := svc.CreateRating(ctx, ratingRequest())
	assert.NoError(t, err)

	t.Run("NoMatchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{UserID: "non-existent", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByTargetID", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{TargetID: "non-existent", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByTargetType", func(t *testing.T) {
		results, total, err := svc.SearchRatings(ctx, model.SearchRatingRequest{TargetType: "driver", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})
}

func TestFunctional_Rating_EdgeCases(t *testing.T) {
	cleanupRatings()
	repo := repository.NewRatingRepository(testDB)
	svc := service.NewRatingService(repo)
	ctx := context.Background()

	t.Run("CreateWithEmptyUserID", func(t *testing.T) {
		req := ratingRequest()
		req.UserID = ""
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user id is required")
	})

	t.Run("CreateWithEmptyTargetID", func(t *testing.T) {
		req := ratingRequest()
		req.TargetID = ""
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target id is required")
	})

	t.Run("CreateWithEmptyTargetType", func(t *testing.T) {
		req := ratingRequest()
		req.TargetType = ""
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target type is required")
	})

	t.Run("CreateWithScoreTooLow", func(t *testing.T) {
		req := ratingRequest()
		req.Score = 0
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be between 1 and 5")
	})

	t.Run("CreateWithScoreTooHigh", func(t *testing.T) {
		req := ratingRequest()
		req.Score = 6
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be between 1 and 5")
	})

	t.Run("CreateWithNegativeScore", func(t *testing.T) {
		req := ratingRequest()
		req.Score = -1
		err := svc.CreateRating(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be between 1 and 5")
	})

	t.Run("GetWithEmptyID", func(t *testing.T) {
		_, err := svc.GetRating(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating id is required")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := svc.GetRating(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating not found")
	})

	t.Run("DeleteWithEmptyID", func(t *testing.T) {
		err := svc.DeleteRating(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating id is required")
	})
}

func TestFunctional_Rating_FullLifecycle(t *testing.T) {
	cleanupRatings()
	repo := repository.NewRatingRepository(testDB)
	svc := service.NewRatingService(repo)
	ctx := context.Background()

	req := ratingRequest()
	err := svc.CreateRating(ctx, req)
	assert.NoError(t, err)

	created, err := svc.GetRating(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, created.Score)

	err = svc.UpdateRating(ctx, &model.Rating{ID: req.ID, Score: 3, Comment: "Updated comment"})
	assert.NoError(t, err)

	updated, err := svc.GetRating(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, updated.Score)
	assert.Equal(t, "Updated comment", updated.Comment)

	err = svc.DeleteRating(ctx, req.ID)
	assert.NoError(t, err)

	_, err = svc.GetRating(ctx, req.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rating not found")
}
