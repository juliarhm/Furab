//go:build functional
// +build functional

// Package functional contains functional tests for location-service.
// Functional tests access a REAL Redis instance (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d redis
//   - Redis available on 127.0.0.1:6379
package functional

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/location-service/internal/model"
	"furab-backend/services/location-service/internal/repository"
	"furab-backend/services/location-service/internal/service"

	"github.com/redis/go-redis/v9"
)

var (
	testRDB  *redis.Client
	testRepo repository.LocationRepository
	testSvc  service.LocationService
)

// stubDriverClient is a test stub that always validates the driver.
type stubDriverClient struct{}

func (s *stubDriverClient) ValidateDriver(ctx context.Context, driverID string) (bool, error) {
	return true, nil
}

// TestMain sets up the Redis connection, runs tests, and cleans up.
func TestMain(m *testing.M) {
	redisHost := getEnvOrDefault("REDIS_HOST", "127.0.0.1")
	redisPort := getEnvOrDefault("REDIS_PORT", "6379")
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "")

	testRDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       1, // Use DB 1 for testing to avoid conflict with production
	})

	// Wait for Redis to be ready (max 30 seconds)
	ctx := context.Background()
	var err error
	for i := 0; i < 30; i++ {
		err = testRDB.Ping(ctx).Err()
		if err == nil {
			break
		}
		log.Printf("Waiting for Redis... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Redis is not ready: %v", err)
	}
	log.Println("Redis connected!")

	// Flush test DB before tests
	testRDB.FlushDB(ctx)

	// Initialize repository and service
	testRepo = repository.NewRedisLocationRepository(testRDB)
	testSvc = service.NewLocationService(testRepo, &stubDriverClient{})

	// Run tests
	code := m.Run()

	// Cleanup
	testRDB.FlushDB(ctx)
	testRDB.Close()
	os.Exit(code)
}

func cleanupRedis() {
	testRDB.FlushDB(context.Background())
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_UpdateAndTrackDriver tests updating a driver's location and tracking it from real Redis.
func TestFunctional_UpdateAndTrackDriver(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()

	driverID := "func-driver-001"
	now := time.Now()

	// Update driver location → GEOADD ke Redis beneran
	req := model.UpdateLocationRequest{
		DriverID:  driverID,
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: now,
	}
	err := testSvc.UpdateDriverLocation(ctx, req)
	if err != nil {
		t.Fatalf("failed to update location: %v", err)
	}
	t.Logf("Updated location for driver: %s", driverID)

	// Track driver → GEOPOS dari Redis beneran
	loc, err := testSvc.GetDriverLocation(ctx, driverID, "")
	if err != nil {
		t.Fatalf("failed to track driver: %v", err)
	}

	if loc.DriverID != driverID {
		t.Errorf("expected driver_id %s, got: %s", driverID, loc.DriverID)
	}

	// Redis GEO has precision loss, check within ~0.001 degree tolerance
	if abs(loc.Latitude-req.Latitude) > 0.001 {
		t.Errorf("latitude mismatch: expected ~%.4f, got: %.4f", req.Latitude, loc.Latitude)
	}
	if abs(loc.Longitude-req.Longitude) > 0.001 {
		t.Errorf("longitude mismatch: expected ~%.4f, got: %.4f", req.Longitude, loc.Longitude)
	}
	t.Logf("Tracked driver at: lat=%.4f, lng=%.4f", loc.Latitude, loc.Longitude)
}

// TestFunctional_SearchNearbyDrivers tests geo-search for nearby drivers in real Redis.
// Registers 3 drivers at different locations and searches from an origin point.
func TestFunctional_SearchNearbyDrivers(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()
	now := time.Now()

	// Register 3 drivers at different distances from Monas, Jakarta
	drivers := []struct {
		id  string
		lat float64
		lng float64
	}{
		{"driver-near", -6.1760, 106.8270},
		{"driver-medium", -6.2300, 106.8500},
		{"driver-far", -6.2600, 106.7810},
	}

	for _, d := range drivers {
		err := testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
			DriverID:  d.id,
			Latitude:  d.lat,
			Longitude: d.lng,
			Timestamp: now,
		})
		if err != nil {
			t.Fatalf("failed to register driver %s: %v", d.id, err)
		}

		// Set all drivers as available
		err = testSvc.UpdateDriverStatus(ctx, model.UpdateStatusRequest{
			DriverID:     d.id,
			DriverStatus: "available",
		})
		if err != nil {
			t.Fatalf("failed to set status for %s: %v", d.id, err)
		}
	}
	t.Log("Registered 3 drivers with available status")

	// Search from Monas with 5km radius
	results, err := testSvc.FindNearbyDrivers(ctx, model.SearchDriverRequest{
		LatitudeOrigin:  -6.1754,
		LongitudeOrigin: 106.8272,
		Radius:          5.0,
	})
	if err != nil {
		t.Fatalf("failed to search nearby drivers: %v", err)
	}

	if len(results) < 1 {
		t.Fatalf("expected at least 1 driver within 5km, got: %d", len(results))
	}
	t.Logf("Found %d drivers within 5km radius", len(results))

	// Verify results are sorted by distance (ASC)
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("results not sorted by distance: %.2f < %.2f", results[i].Distance, results[i-1].Distance)
		}
	}

	// Verify each result has valid fields
	for _, r := range results {
		if r.DriverID == "" {
			t.Error("expected non-empty driver_id")
		}
		if r.DriverStatus != "available" {
			t.Errorf("expected status available, got: %s", r.DriverStatus)
		}
		t.Logf("  Driver: %s, distance: %.2f km, status: %s", r.DriverID, r.Distance, r.DriverStatus)
	}

	// Search with 15km radius → should find all 3
	allResults, err := testSvc.FindNearbyDrivers(ctx, model.SearchDriverRequest{
		LatitudeOrigin:  -6.1754,
		LongitudeOrigin: 106.8272,
		Radius:          15.0,
	})
	if err != nil {
		t.Fatalf("failed to search all drivers: %v", err)
	}
	if len(allResults) != 3 {
		t.Errorf("expected 3 drivers within 15km, got: %d", len(allResults))
	}
	t.Logf("Found %d drivers within 10km radius", len(allResults))
}

// TestFunctional_UpdateDriverStatus tests that only available drivers appear in search.
func TestFunctional_UpdateDriverStatus(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()
	now := time.Now()

	// Register 2 drivers near same location
	err := testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
		DriverID: "driver-avail", Latitude: -6.2088, Longitude: 106.8456, Timestamp: now,
	})
	if err != nil {
		t.Fatalf("register driver-avail: %v", err)
	}
	err = testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
		DriverID: "driver-busy", Latitude: -6.2090, Longitude: 106.8460, Timestamp: now,
	})
	if err != nil {
		t.Fatalf("register driver-busy: %v", err)
	}

	// Set statuses
	err = testSvc.UpdateDriverStatus(ctx, model.UpdateStatusRequest{
		DriverID: "driver-avail", DriverStatus: "available",
	})
	if err != nil {
		t.Fatalf("set available: %v", err)
	}
	err = testSvc.UpdateDriverStatus(ctx, model.UpdateStatusRequest{
		DriverID: "driver-busy", DriverStatus: "busy",
	})
	if err != nil {
		t.Fatalf("set busy: %v", err)
	}

	// Search → should only find available driver
	results, err := testSvc.FindNearbyDrivers(ctx, model.SearchDriverRequest{
		LatitudeOrigin: -6.2088, LongitudeOrigin: 106.8456, Radius: 5.0,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 available driver, got: %d", len(results))
	}
	if results[0].DriverID != "driver-avail" {
		t.Errorf("expected driver-avail, got: %s", results[0].DriverID)
	}
	if results[0].DriverStatus != "available" {
		t.Errorf("expected available, got: %s", results[0].DriverStatus)
	}
	t.Logf("Correctly filtered: only %s (status: %s)", results[0].DriverID, results[0].DriverStatus)
}

// TestFunctional_InvalidDriverID tests that empty driver_id is rejected.
func TestFunctional_InvalidDriverID(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()

	// Update with empty driver_id
	err := testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
		DriverID:  "",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for empty driver_id")
	}
	t.Logf("Correctly rejected empty driver_id: %v", err)

	// Track non-existent driver
	_, err = testSvc.GetDriverLocation(ctx, "", "")
	if err == nil {
		t.Fatal("expected error for empty driver_id in GetDriverLocation")
	}
	t.Logf("Correctly rejected empty driver_id in track: %v", err)
}

// TestFunctional_UpdateLocationOverwrite tests that updating location overwrites the previous one.
func TestFunctional_UpdateLocationOverwrite(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()
	driverID := "func-driver-overwrite"

	// First location: Monas
	err := testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
		DriverID: driverID, Latitude: -6.1754, Longitude: 106.8272, Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("update 1: %v", err)
	}

	// Second location: Sudirman (overwrite)
	err = testSvc.UpdateDriverLocation(ctx, model.UpdateLocationRequest{
		DriverID: driverID, Latitude: -6.2088, Longitude: 106.8456, Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("update 2: %v", err)
	}

	// Track → should return second location
	loc, err := testSvc.GetDriverLocation(ctx, driverID, "")
	if err != nil {
		t.Fatalf("track: %v", err)
	}

	// Should be near Sudirman, not Monas
	if abs(loc.Latitude-(-6.2088)) > 0.001 {
		t.Errorf("expected latitude ~-6.2088, got: %.4f (location not overwritten?)", loc.Latitude)
	}
	t.Logf("Location correctly overwritten to lat=%.4f, lng=%.4f", loc.Latitude, loc.Longitude)
}

// TestFunctional_InvalidRadius tests that search with invalid radius returns error.
func TestFunctional_InvalidRadius(t *testing.T) {
	cleanupRedis()
	ctx := context.Background()

	// Search with radius 0
	_, err := testSvc.FindNearbyDrivers(ctx, model.SearchDriverRequest{
		LatitudeOrigin: -6.2088, LongitudeOrigin: 106.8456, Radius: 0,
	})
	if err == nil {
		t.Fatal("expected error for radius 0")
	}
	t.Logf("Correctly rejected radius 0: %v", err)

	// Search with negative radius
	_, err = testSvc.FindNearbyDrivers(ctx, model.SearchDriverRequest{
		LatitudeOrigin: -6.2088, LongitudeOrigin: 106.8456, Radius: -5.0,
	})
	if err == nil {
		t.Fatal("expected error for negative radius")
	}
	t.Logf("Correctly rejected negative radius: %v", err)
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
