//go:build functional
// +build functional

// Package functional contains functional tests for emergency-service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "emergency_service" created automatically by init script or manually
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/emergency-service/internal/model"
	"furab-backend/services/emergency-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB  *sql.DB
	testSvc service.EmergencyService
)

// --- In-test PostgreSQL repository (implements repository.EmergencyRepository) ---

type testEmergencyRepo struct {
	db *sql.DB
}

func (r *testEmergencyRepo) SaveEmergencyEvent(ctx context.Context, event model.EmergencyEvent) error {
	query := `INSERT INTO emergency_events
		(emergency_id, actor_id, actor_type, order_id, latitude, longitude,
		 emergency_type, status, created_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query,
		event.EmergencyID, event.ActorID, event.ActorType, event.OrderID,
		event.Latitude, event.Longitude, event.EmergencyType, event.Status,
		event.CreatedAt, event.ResolvedAt)
	return err
}

// --- Stubs for external service clients ---

type stubLocationClient struct {
	shouldFail bool
}

func (s *stubLocationClient) GetLastLocation(ctx context.Context, actorID, actorType string) (*model.EmergencyLocation, error) {
	if s.shouldFail {
		return nil, fmt.Errorf("location service unavailable")
	}
	return &model.EmergencyLocation{
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: time.Now(),
		Accuracy:  10.0,
	}, nil
}

type stubActorClient struct {
	shouldFailValidate bool
	shouldFailContact  bool
	hasContact         bool
}

func (s *stubActorClient) ValidateActor(ctx context.Context, actorID, actorType string) (bool, error) {
	if s.shouldFailValidate {
		return false, fmt.Errorf("actor validation failed")
	}
	return true, nil
}

func (s *stubActorClient) ValidateOrder(ctx context.Context, orderID string) (bool, error) {
	return true, nil
}

func (s *stubActorClient) GetEmergencyContact(ctx context.Context, actorID, actorType string) (*model.EmergencyContact, error) {
	if s.shouldFailContact {
		return nil, fmt.Errorf("contact not found")
	}
	if s.hasContact {
		return &model.EmergencyContact{
			ReceiverID: "emergency-contact-001",
			Phone:      "+6281234567890",
			Email:      "emergency@example.com",
		}, nil
	}
	return nil, nil
}

type stubNotificationClient struct {
	notifSent    bool
	contactSent  bool
	shouldFail   bool
}

func (s *stubNotificationClient) SendNotification(ctx context.Context, notification model.EmergencyNotification) error {
	if s.shouldFail {
		return fmt.Errorf("notification failed")
	}
	s.notifSent = true
	return nil
}

func (s *stubNotificationClient) SendEmergencyContact(ctx context.Context, contact model.EmergencyContact, notification model.EmergencyNotification) error {
	if s.shouldFail {
		return fmt.Errorf("contact notification failed")
	}
	s.contactSent = true
	return nil
}

// --- TestMain setup ---

// Default stubs (will be overridden per test if needed)
var (
	defaultLocationClient *stubLocationClient
	defaultActorClient    *stubActorClient
	defaultNotifClient    *stubNotificationClient
)

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "emergency_service")

	// Step 1: Auto-create database
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)
	adminDB, err := sql.Open("pgx", adminDSN)
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

	testDB, err = sql.Open("pgx", dsn)
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
	log.Println("Database connected!")

	// Setup schema
	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	// Run tests (service is initialized per test to allow different stubs)
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// newTestService creates a new emergency service with given stubs and real DB repo.
func newTestService(
	locClient *stubLocationClient,
	actorClient *stubActorClient,
	notifClient *stubNotificationClient,
) service.EmergencyService {
	repo := &testEmergencyRepo{db: testDB}
	return service.NewEmergencyServiceWithDependencies(
		repo, locClient, actorClient, notifClient,
	)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS emergency_events (
			emergency_id VARCHAR(36) PRIMARY KEY,
			actor_id VARCHAR(36) NOT NULL,
			actor_type VARCHAR(20) NOT NULL,
			order_id VARCHAR(36),
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			emergency_type VARCHAR(30) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			resolved_at TIMESTAMP WITH TIME ZONE
		);

		CREATE INDEX IF NOT EXISTS idx_emergency_events_actor ON emergency_events(actor_id);
		CREATE INDEX IF NOT EXISTS idx_emergency_events_order ON emergency_events(order_id);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS emergency_events")
}

func cleanupEvents() {
	testDB.Exec("DELETE FROM emergency_events")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_TriggerEmergencyFull tests the full emergency trigger flow with real DB.
func TestFunctional_TriggerEmergencyFull(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	locClient := &stubLocationClient{}
	actorClient := &stubActorClient{hasContact: true}
	notifClient := &stubNotificationClient{}
	svc := newTestService(locClient, actorClient, notifClient)

	req := model.TriggerEmergencyRequest{
		ActorID:       "user-emg-001",
		ActorType:     "user",
		OrderID:       "order-emg-001",
		Latitude:      -6.2088,
		Longitude:     106.8456,
		EmergencyType: "accident",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("failed to trigger emergency: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status success, got: %s", resp.Status)
	}
	if resp.EmergencyID == "" {
		t.Fatal("expected non-empty emergency_id")
	}
	if resp.Message != "emergency created" {
		t.Errorf("expected message 'emergency created', got: %s", resp.Message)
	}
	t.Logf("Emergency triggered: %s (status: %s)", resp.EmergencyID, resp.Status)

	// Verify event persisted in DB
	var actorID, actorType, emergencyType, status string
	var lat, lng float64
	err = testDB.QueryRow(
		"SELECT actor_id, actor_type, emergency_type, status, latitude, longitude FROM emergency_events WHERE emergency_id = $1",
		resp.EmergencyID).Scan(&actorID, &actorType, &emergencyType, &status, &lat, &lng)
	if err != nil {
		t.Fatalf("failed to verify event in DB: %v", err)
	}
	if actorID != req.ActorID {
		t.Errorf("DB actor_id: expected %s, got: %s", req.ActorID, actorID)
	}
	if actorType != req.ActorType {
		t.Errorf("DB actor_type: expected %s, got: %s", req.ActorType, actorType)
	}
	if emergencyType != req.EmergencyType {
		t.Errorf("DB emergency_type: expected %s, got: %s", req.EmergencyType, emergencyType)
	}
	if status != "active" {
		t.Errorf("DB status: expected active, got: %s", status)
	}
	t.Logf("DB verified: actor=%s, type=%s, emergency=%s, status=%s", actorID, actorType, emergencyType, status)

	// Verify notification was sent
	if !notifClient.notifSent {
		t.Error("expected notification to be sent")
	}
	if !notifClient.contactSent {
		t.Error("expected emergency contact notification to be sent")
	}
	t.Log("Notifications verified: both main + contact notified")
}

// TestFunctional_DriverEmergency tests emergency triggered by a driver.
func TestFunctional_DriverEmergency(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	svc := newTestService(&stubLocationClient{}, &stubActorClient{}, &stubNotificationClient{})

	req := model.TriggerEmergencyRequest{
		ActorID:       "driver-emg-001",
		ActorType:     "driver",
		OrderID:       "order-emg-002",
		Latitude:      -6.1751,
		Longitude:     106.8650,
		EmergencyType: "unsafe",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("failed to trigger driver emergency: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success, got: %s", resp.Status)
	}
	t.Logf("Driver emergency triggered: %s", resp.EmergencyID)

	// Verify in DB
	var actorType string
	err = testDB.QueryRow("SELECT actor_type FROM emergency_events WHERE emergency_id = $1",
		resp.EmergencyID).Scan(&actorType)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if actorType != "driver" {
		t.Errorf("expected actor_type driver, got: %s", actorType)
	}
}

// TestFunctional_LocationFallback tests that emergency still works when location service fails.
func TestFunctional_LocationFallback(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	locClient := &stubLocationClient{shouldFail: true}
	svc := newTestService(locClient, &stubActorClient{}, &stubNotificationClient{})

	reqLat := -6.3000
	reqLng := 106.8500

	req := model.TriggerEmergencyRequest{
		ActorID:       "user-emg-fallback",
		ActorType:     "user",
		OrderID:       "order-emg-fallback",
		Latitude:      reqLat,
		Longitude:     reqLng,
		EmergencyType: "other",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("emergency should succeed even with location failure: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success, got: %s", resp.Status)
	}
	t.Logf("Emergency succeeded with location fallback: %s", resp.EmergencyID)

	// Verify coordinates fallback to request payload
	var lat, lng float64
	err = testDB.QueryRow("SELECT latitude, longitude FROM emergency_events WHERE emergency_id = $1",
		resp.EmergencyID).Scan(&lat, &lng)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if lat != reqLat {
		t.Errorf("expected fallback latitude %.4f, got: %.4f", reqLat, lat)
	}
	if lng != reqLng {
		t.Errorf("expected fallback longitude %.4f, got: %.4f", reqLng, lng)
	}
	t.Logf("Location fallback verified: lat=%.4f, lng=%.4f", lat, lng)
}

// TestFunctional_InvalidActor tests that invalid actor types are rejected.
func TestFunctional_InvalidActor(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	svc := newTestService(&stubLocationClient{}, &stubActorClient{}, &stubNotificationClient{})

	// Empty actor_id
	resp, err := svc.TriggerEmergency(ctx, model.TriggerEmergencyRequest{
		ActorID: "", ActorType: "user", OrderID: "order-001",
		EmergencyType: "accident", Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for empty actor_id")
	}
	if resp != nil && resp.Status != "failed" {
		t.Errorf("expected status failed, got: %s", resp.Status)
	}
	t.Logf("Correctly rejected empty actor_id: %v", err)

	// Invalid actor_type
	resp, err = svc.TriggerEmergency(ctx, model.TriggerEmergencyRequest{
		ActorID: "user-001", ActorType: "admin", OrderID: "order-001",
		EmergencyType: "accident", Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for invalid actor_type")
	}
	if resp != nil && resp.Status != "failed" {
		t.Errorf("expected status failed, got: %s", resp.Status)
	}
	t.Logf("Correctly rejected invalid actor_type: %v", err)
}

// TestFunctional_InvalidActorValidation tests that actor validation failure is handled.
func TestFunctional_InvalidActorValidation(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	actorClient := &stubActorClient{shouldFailValidate: true}
	svc := newTestService(&stubLocationClient{}, actorClient, &stubNotificationClient{})

	resp, err := svc.TriggerEmergency(ctx, model.TriggerEmergencyRequest{
		ActorID: "bad-actor", ActorType: "user", OrderID: "order-001",
		EmergencyType: "accident", Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for invalid actor validation")
	}
	if resp != nil && resp.Status != "failed" {
		t.Errorf("expected status failed, got: %s", resp.Status)
	}
	t.Logf("Correctly rejected invalid actor: %v", err)
}

// TestFunctional_NotificationFailureDoesNotBlock tests that notification failure doesn't block emergency.
func TestFunctional_NotificationFailureDoesNotBlock(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	notifClient := &stubNotificationClient{shouldFail: true}
	svc := newTestService(&stubLocationClient{}, &stubActorClient{}, notifClient)

	req := model.TriggerEmergencyRequest{
		ActorID:       "user-emg-notif-fail",
		ActorType:     "user",
		OrderID:       "order-emg-notif",
		Latitude:      -6.2088,
		Longitude:     106.8456,
		EmergencyType: "accident",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("emergency should succeed even when notification fails: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success despite notification failure, got: %s", resp.Status)
	}
	t.Logf("Emergency succeeded despite notification failure: %s", resp.EmergencyID)

	// Verify event still persisted in DB
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM emergency_events WHERE emergency_id = $1",
		resp.EmergencyID).Scan(&count)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 event in DB, got: %d", count)
	}
	t.Logf("Event persisted in DB despite notification failure")
}

// TestFunctional_WithEmergencyContact tests emergency trigger with contact notification.
func TestFunctional_WithEmergencyContact(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	notifClient := &stubNotificationClient{}
	actorClient := &stubActorClient{hasContact: true}
	svc := newTestService(&stubLocationClient{}, actorClient, notifClient)

	req := model.TriggerEmergencyRequest{
		ActorID:       "user-emg-contact",
		ActorType:     "user",
		OrderID:       "order-emg-contact",
		Latitude:      -6.2088,
		Longitude:     106.8456,
		EmergencyType: "accident",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success, got: %s", resp.Status)
	}

	// Verify both notifications sent
	if !notifClient.notifSent {
		t.Error("expected main notification to be sent")
	}
	if !notifClient.contactSent {
		t.Error("expected emergency contact to be notified")
	}
	t.Logf("Both main + contact notifications sent for emergency: %s", resp.EmergencyID)
}

// TestFunctional_EmergencyWithoutOrder tests emergency without order_id (standalone emergency).
func TestFunctional_EmergencyWithoutOrder(t *testing.T) {
	cleanupEvents()
	ctx := context.Background()

	svc := newTestService(&stubLocationClient{}, &stubActorClient{}, &stubNotificationClient{})

	req := model.TriggerEmergencyRequest{
		ActorID:       "user-emg-noorder",
		ActorType:     "user",
		OrderID:       "", // No order
		Latitude:      -6.2088,
		Longitude:     106.8456,
		EmergencyType: "other",
		Timestamp:     time.Now(),
	}

	resp, err := svc.TriggerEmergency(ctx, req)
	if err != nil {
		t.Fatalf("emergency without order should succeed: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success, got: %s", resp.Status)
	}
	t.Logf("Emergency without order succeeded: %s", resp.EmergencyID)

	// Verify in DB
	var orderID sql.NullString
	err = testDB.QueryRow("SELECT order_id FROM emergency_events WHERE emergency_id = $1",
		resp.EmergencyID).Scan(&orderID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	// order_id should be empty string or null
	t.Logf("Order ID in DB: '%s' (valid: %v)", orderID.String, orderID.Valid)
}
