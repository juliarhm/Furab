//go:build functional
// +build functional

// Package functional contains functional tests for notification-service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "notification_service" created automatically by init script or manually
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/notification-service/internal/model"
	"furab-backend/services/notification-service/internal/repository"
	"furab-backend/services/notification-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.NotificationRepository
	testSvc  service.NotificationService
)

// --- In-test PostgreSQL repository (implements repository.NotificationRepository) ---

type testNotifRepo struct {
	db *sql.DB
}

func (r *testNotifRepo) SaveNotificationLog(ctx context.Context, notifLog model.NotificationLog) error {
	query := `INSERT INTO notification_logs
		(notification_id, receiver_id, title, message, channel, status, reference_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (notification_id) DO UPDATE SET status = $6`
	_, err := r.db.ExecContext(ctx, query,
		notifLog.NotificationID, notifLog.ReceiverID, notifLog.Title, notifLog.Message,
		notifLog.Channel, notifLog.Status, notifLog.ReferenceID, notifLog.Timestamp)
	return err
}

func (r *testNotifRepo) GetTemplateByEventType(ctx context.Context, eventType string) (*model.NotifTemplate, error) {
	query := `SELECT event_type, title_template, message_template
		FROM notification_templates WHERE event_type = $1`
	var t model.NotifTemplate
	err := r.db.QueryRowContext(ctx, query, eventType).Scan(&t.EventType, &t.TitleTemplate, &t.MessageTemplate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- Stub for EmailClient ---

type stubEmailClient struct{}

func (s *stubEmailClient) SendEmail(ctx context.Context, receiverID string, title string, message string) error {
	return nil
}

// --- TestMain setup ---

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "notification_service")

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

	// Seed templates
	if err := seedTemplates(); err != nil {
		log.Fatalf("Failed to seed templates: %v", err)
	}

	// Initialize repository and service
	repo := &testNotifRepo{db: testDB}
	testRepo = repo
	testSvc = service.NewNotificationService(testRepo, &stubEmailClient{})

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS notification_templates (
			event_type VARCHAR(50) PRIMARY KEY,
			title_template TEXT NOT NULL,
			message_template TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS notification_logs (
			notification_id VARCHAR(36) PRIMARY KEY,
			receiver_id VARCHAR(36) NOT NULL,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			channel VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'sent',
			reference_id VARCHAR(36),
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_notification_logs_receiver ON notification_logs(receiver_id);
	`
	_, err := testDB.Exec(query)
	return err
}

func seedTemplates() error {
	templates := []struct {
		eventType string
		title     string
		message   string
	}{
		{"ride.created", "Pesanan Baru", "Pesanan ride Anda telah dibuat"},
		{"ride.assigned", "Driver Ditemukan", "Driver sedang menuju lokasi Anda"},
		{"ride.completed", "Perjalanan Selesai", "Perjalanan Anda telah selesai"},
		{"ride.cancelled", "Pesanan Dibatalkan", "Pesanan ride Anda telah dibatalkan"},
		{"payment.success", "Pembayaran Berhasil", "Pembayaran Anda telah berhasil diproses"},
	}

	for _, t := range templates {
		_, err := testDB.Exec(`INSERT INTO notification_templates (event_type, title_template, message_template)
			VALUES ($1, $2, $3) ON CONFLICT (event_type) DO NOTHING`, t.eventType, t.title, t.message)
		if err != nil {
			return err
		}
	}
	return nil
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS notification_logs")
	testDB.Exec("DROP TABLE IF EXISTS notification_templates")
}

func cleanupLogs() {
	testDB.Exec("DELETE FROM notification_logs")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_SendPushNotification tests sending a push notification and verifying the log in DB.
func TestFunctional_SendPushNotification(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.EventNotificationRequest{
		EventType:   "ride.created",
		ReceiverID:  "user-notif-001",
		ReferenceID: "order-001",
		Channel:     "push",
	}

	resp, err := testSvc.SendNotification(ctx, req)
	if err != nil {
		t.Fatalf("failed to send notification: %v", err)
	}

	if resp.NotificationID == "" {
		t.Fatal("expected non-empty notification_id")
	}
	if resp.Status != "success" {
		t.Errorf("expected status success, got: %s", resp.Status)
	}
	if resp.Channel != "push" {
		t.Errorf("expected channel push, got: %s", resp.Channel)
	}
	t.Logf("Sent push notification: %s (status: %s)", resp.NotificationID, resp.Status)

	// Verify log in DB
	var title, channel, status string
	err = testDB.QueryRow("SELECT title, channel, status FROM notification_logs WHERE notification_id = $1",
		resp.NotificationID).Scan(&title, &channel, &status)
	if err != nil {
		t.Fatalf("failed to verify log in DB: %v", err)
	}
	if title != "Pesanan Baru" {
		t.Errorf("expected title 'Pesanan Baru', got: %s", title)
	}
	if channel != "push" {
		t.Errorf("expected channel 'push', got: %s", channel)
	}
	if status != "sent" {
		t.Errorf("expected status 'sent', got: %s", status)
	}
	t.Logf("DB log verified: title='%s', channel='%s', status='%s'", title, channel, status)
}

// TestFunctional_SendEmailNotification tests sending a notification via email channel.
func TestFunctional_SendEmailNotification(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.EventNotificationRequest{
		EventType:   "payment.success",
		ReceiverID:  "user-notif-002",
		ReferenceID: "payment-001",
		Channel:     "email",
	}

	resp, err := testSvc.SendNotification(ctx, req)
	if err != nil {
		t.Fatalf("failed to send email notification: %v", err)
	}

	if resp.NotificationID == "" {
		t.Fatal("expected non-empty notification_id")
	}
	if resp.Channel != "email" {
		t.Errorf("expected channel email, got: %s", resp.Channel)
	}
	t.Logf("Sent email notification: %s (channel: %s)", resp.NotificationID, resp.Channel)

	// Verify log in DB
	var message, status string
	err = testDB.QueryRow("SELECT message, status FROM notification_logs WHERE notification_id = $1",
		resp.NotificationID).Scan(&message, &status)
	if err != nil {
		t.Fatalf("failed to verify log: %v", err)
	}
	if message != "Pembayaran Anda telah berhasil diproses" {
		t.Errorf("expected template message, got: %s", message)
	}
	t.Logf("DB log verified: message='%s', status='%s'", message, status)
}

// TestFunctional_InvalidEventType tests that unknown event types are rejected.
func TestFunctional_InvalidEventType(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.EventNotificationRequest{
		EventType:   "unknown.event",
		ReceiverID:  "user-notif-003",
		ReferenceID: "ref-001",
		Channel:     "push",
	}

	_, err := testSvc.SendNotification(ctx, req)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
	t.Logf("Correctly rejected unknown event type: %v", err)
}

// TestFunctional_ValidationErrors tests that missing required fields are rejected.
func TestFunctional_ValidationErrors(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	// Missing event_type
	_, err := testSvc.SendNotification(ctx, model.EventNotificationRequest{
		EventType: "", ReceiverID: "user-001", ReferenceID: "ref-001", Channel: "push",
	})
	if err == nil {
		t.Fatal("expected error for empty event_type")
	}
	t.Logf("Correctly rejected empty event_type: %v", err)

	// Missing receiver_id
	_, err = testSvc.SendNotification(ctx, model.EventNotificationRequest{
		EventType: "ride.created", ReceiverID: "", ReferenceID: "ref-001", Channel: "push",
	})
	if err == nil {
		t.Fatal("expected error for empty receiver_id")
	}
	t.Logf("Correctly rejected empty receiver_id: %v", err)

	// Missing reference_id
	_, err = testSvc.SendNotification(ctx, model.EventNotificationRequest{
		EventType: "ride.created", ReceiverID: "user-001", ReferenceID: "", Channel: "push",
	})
	if err == nil {
		t.Fatal("expected error for empty reference_id")
	}
	t.Logf("Correctly rejected empty reference_id: %v", err)

	// Invalid channel
	_, err = testSvc.SendNotification(ctx, model.EventNotificationRequest{
		EventType: "ride.created", ReceiverID: "user-001", ReferenceID: "ref-001", Channel: "sms",
	})
	if err == nil {
		t.Fatal("expected error for invalid channel")
	}
	t.Logf("Correctly rejected invalid channel: %v", err)

	// Missing channel
	_, err = testSvc.SendNotification(ctx, model.EventNotificationRequest{
		EventType: "ride.created", ReceiverID: "user-001", ReferenceID: "ref-001", Channel: "",
	})
	if err == nil {
		t.Fatal("expected error for empty channel")
	}
	t.Logf("Correctly rejected empty channel: %v", err)
}

// TestFunctional_GenerateTemplate tests template generation for known event types.
func TestFunctional_GenerateTemplate(t *testing.T) {
	ctx := context.Background()

	template, err := testSvc.GenerateNotificationTemplate(ctx, "ride.assigned")
	if err != nil {
		t.Fatalf("failed to generate template: %v", err)
	}
	if template == nil {
		t.Fatal("expected non-nil template")
	}
	if template.EventType != "ride.assigned" {
		t.Errorf("expected event_type ride.assigned, got: %s", template.EventType)
	}
	if template.TitleTemplate != "Driver Ditemukan" {
		t.Errorf("expected title 'Driver Ditemukan', got: %s", template.TitleTemplate)
	}
	t.Logf("Template: event=%s, title='%s', message='%s'", template.EventType, template.TitleTemplate, template.MessageTemplate)

	// Non-existent template
	_, err = testSvc.GenerateNotificationTemplate(ctx, "nonexistent.event")
	if err == nil {
		t.Fatal("expected error for non-existent template")
	}
	t.Logf("Correctly rejected non-existent template: %v", err)
}

// TestFunctional_MultipleNotifications tests sending multiple notifications and verifying all logs.
func TestFunctional_MultipleNotifications(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	events := []string{"ride.created", "ride.assigned", "ride.completed"}

	for i, event := range events {
		resp, err := testSvc.SendNotification(ctx, model.EventNotificationRequest{
			EventType:   event,
			ReceiverID:  fmt.Sprintf("user-multi-%d", i+1),
			ReferenceID: fmt.Sprintf("order-multi-%d", i+1),
			Channel:     "push",
		})
		if err != nil {
			t.Fatalf("send notification %d (%s): %v", i+1, event, err)
		}
		t.Logf("Sent %s → %s", event, resp.NotificationID)
	}

	// Count logs in DB
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM notification_logs").Scan(&count)
	if err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 logs in DB, got: %d", count)
	}
	t.Logf("Total logs in DB: %d", count)
}
