//go:build functional
// +build functional

// Package functional contains functional tests for email-service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "email_service" created automatically by init script or manually
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/email-service/internal/model"
	"furab-backend/services/email-service/internal/repository"
	"furab-backend/services/email-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.EmailRepository
	testSvc  service.EmailService
)

// --- In-test PostgreSQL repository (implements repository.EmailRepository) ---

type testEmailRepo struct {
	db *sql.DB
}

func (r *testEmailRepo) SaveEmailLog(ctx context.Context, emailLog model.EmailLog) error {
	query := `INSERT INTO email_logs
		(email_id, receiver_email, subject, status, timestamp, receiver_id, reference_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (email_id) DO UPDATE SET status = $4`
	_, err := r.db.ExecContext(ctx, query,
		emailLog.EmailID, emailLog.ReceiverEmail, emailLog.Subject,
		emailLog.Status, emailLog.Timestamp, emailLog.ReceiverID, emailLog.ReferenceID)
	return err
}

func (r *testEmailRepo) GetTemplateByID(ctx context.Context, templateID string) (*model.EmailTemplate, error) {
	query := `SELECT template_id, subject, body FROM email_templates WHERE template_id = $1`
	var t model.EmailTemplate
	err := r.db.QueryRowContext(ctx, query, templateID).Scan(&t.TemplateID, &t.Subject, &t.Body)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- Stub for EmailSender ---

type stubEmailSender struct {
	lastEmail string
	lastSubj  string
}

func (s *stubEmailSender) Send(ctx context.Context, receiverEmail, subject, body string) error {
	s.lastEmail = receiverEmail
	s.lastSubj = subject
	return nil
}

// --- TestMain setup ---

var testSender *stubEmailSender

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "email_service")

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
	repo := &testEmailRepo{db: testDB}
	testRepo = repo
	testSender = &stubEmailSender{}
	testSvc = service.NewEmailService(testRepo, testSender)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS email_templates (
			template_id VARCHAR(36) PRIMARY KEY,
			subject VARCHAR(255) NOT NULL,
			body TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS email_logs (
			email_id VARCHAR(36) PRIMARY KEY,
			receiver_email VARCHAR(255) NOT NULL,
			subject VARCHAR(255) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'sent',
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			receiver_id VARCHAR(36),
			reference_id VARCHAR(36)
		);

		CREATE INDEX IF NOT EXISTS idx_email_logs_receiver ON email_logs(receiver_id);
	`
	_, err := testDB.Exec(query)
	return err
}

func seedTemplates() error {
	templates := []struct {
		id      string
		subject string
		body    string
	}{
		{"tpl-welcome", "Selamat Datang di Furab", "Halo {name}, selamat bergabung dengan Furab!"},
		{"tpl-receipt", "Invoice Perjalanan #{order_id}", "Detail perjalanan Anda: {detail}"},
		{"tpl-reset", "Reset Password", "Klik link berikut untuk reset password: {link}"},
	}

	for _, t := range templates {
		_, err := testDB.Exec(`INSERT INTO email_templates (template_id, subject, body)
			VALUES ($1, $2, $3) ON CONFLICT (template_id) DO NOTHING`, t.id, t.subject, t.body)
		if err != nil {
			return err
		}
	}
	return nil
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS email_logs")
	testDB.Exec("DROP TABLE IF EXISTS email_templates")
}

func cleanupLogs() {
	testDB.Exec("DELETE FROM email_logs")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_SendDirectEmail tests sending a direct email and verifying the log in DB.
func TestFunctional_SendDirectEmail(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.SendEmailRequest{
		ReceiverEmail: "user@example.com",
		Subject:       "Test Subject",
		Body:          "Test Body Content",
		ReceiverID:    "user-email-001",
		ReferenceID:   "order-001",
	}

	resp, err := testSvc.SendEmail(ctx, req)
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	if resp.EmailID == "" {
		t.Fatal("expected non-empty email_id")
	}
	if resp.Status != "sent" {
		t.Errorf("expected status sent, got: %s", resp.Status)
	}
	if resp.ReceiverEmail != req.ReceiverEmail {
		t.Errorf("expected receiver %s, got: %s", req.ReceiverEmail, resp.ReceiverEmail)
	}
	if resp.Subject != req.Subject {
		t.Errorf("expected subject '%s', got: '%s'", req.Subject, resp.Subject)
	}
	t.Logf("Sent email: %s → %s (status: %s)", resp.EmailID, resp.ReceiverEmail, resp.Status)

	// Verify log in DB
	var subject, status, receiverID string
	err = testDB.QueryRow("SELECT subject, status, receiver_id FROM email_logs WHERE email_id = $1",
		resp.EmailID).Scan(&subject, &status, &receiverID)
	if err != nil {
		t.Fatalf("failed to verify log in DB: %v", err)
	}
	if subject != req.Subject {
		t.Errorf("DB subject: expected '%s', got: '%s'", req.Subject, subject)
	}
	if status != "sent" {
		t.Errorf("DB status: expected 'sent', got: '%s'", status)
	}
	if receiverID != req.ReceiverID {
		t.Errorf("DB receiver_id: expected '%s', got: '%s'", req.ReceiverID, receiverID)
	}
	t.Logf("DB log verified: subject='%s', status='%s'", subject, status)
}

// TestFunctional_SendEmailWithTemplate tests sending email using a template from DB.
func TestFunctional_SendEmailWithTemplate(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.SendEmailRequest{
		ReceiverEmail: "user@example.com",
		TemplateID:    "tpl-welcome",
		ReceiverID:    "user-email-002",
		ReferenceID:   "reg-001",
	}

	resp, err := testSvc.SendEmail(ctx, req)
	if err != nil {
		t.Fatalf("failed to send template email: %v", err)
	}

	if resp.EmailID == "" {
		t.Fatal("expected non-empty email_id")
	}
	// Subject should come from template
	if resp.Subject != "Selamat Datang di Furab" {
		t.Errorf("expected template subject, got: %s", resp.Subject)
	}
	t.Logf("Sent template email: %s, subject='%s'", resp.EmailID, resp.Subject)

	// Verify sender stub received correct params
	if testSender.lastEmail != req.ReceiverEmail {
		t.Errorf("sender received email: expected %s, got: %s", req.ReceiverEmail, testSender.lastEmail)
	}
}

// TestFunctional_SendEmailWithTemplateNotFound tests that non-existent template returns error.
func TestFunctional_SendEmailWithTemplateNotFound(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.SendEmailRequest{
		ReceiverEmail: "user@example.com",
		TemplateID:    "tpl-nonexistent",
		ReceiverID:    "user-email-003",
		ReferenceID:   "ref-001",
	}

	_, err := testSvc.SendEmail(ctx, req)
	if err == nil {
		t.Fatal("expected error for non-existent template")
	}
	t.Logf("Correctly rejected non-existent template: %v", err)
}

// TestFunctional_TriggerEventEmail tests event-driven email sending.
func TestFunctional_TriggerEventEmail(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.EventEmailRequest{
		EventType:     "payment.success",
		ReceiverEmail: "payment@example.com",
		ReceiverID:    "user-event-001",
		ReferenceID:   "payment-001",
	}

	resp, err := testSvc.TriggerEventEmail(ctx, req)
	if err != nil {
		t.Fatalf("failed to trigger event email: %v", err)
	}

	if resp.EmailID == "" {
		t.Fatal("expected non-empty email_id")
	}
	if resp.Subject != "Invoice" {
		t.Errorf("expected subject 'Invoice', got: %s", resp.Subject)
	}
	t.Logf("Triggered event email: %s, subject='%s'", resp.EmailID, resp.Subject)

	// Verify in DB
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM email_logs WHERE reference_id = $1", req.ReferenceID).Scan(&count)
	if err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 log for reference, got: %d", count)
	}
}

// TestFunctional_TriggerInvalidEvent tests that unknown event types are rejected.
func TestFunctional_TriggerInvalidEvent(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.EventEmailRequest{
		EventType:     "unknown.event",
		ReceiverEmail: "user@example.com",
		ReceiverID:    "user-invalid-001",
		ReferenceID:   "ref-invalid",
	}

	_, err := testSvc.TriggerEventEmail(ctx, req)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
	t.Logf("Correctly rejected unknown event: %v", err)
}

// TestFunctional_SendEmailWithResult tests the SendEmailWithResult wrapper.
func TestFunctional_SendEmailWithResult(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	req := model.SendEmailRequest{
		ReceiverEmail: "result@example.com",
		Subject:       "Result Test",
		Body:          "Testing result wrapper",
		ReceiverID:    "user-result-001",
		ReferenceID:   "ref-result-001",
	}

	result, err := testSvc.SendEmailWithResult(ctx, req)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected result status success, got: %s", result.Status)
	}
	if result.Message != "email berhasil dikirim" {
		t.Errorf("expected success message, got: %s", result.Message)
	}
	if result.Data == nil {
		t.Fatal("expected non-nil data in result")
	}
	if result.Data.EmailID == "" {
		t.Error("expected non-empty email_id in result data")
	}
	t.Logf("Result: status=%s, message=%s, email_id=%s", result.Status, result.Message, result.Data.EmailID)
}

// TestFunctional_InvalidRequest tests that missing required fields are rejected.
func TestFunctional_InvalidRequest(t *testing.T) {
	cleanupLogs()
	ctx := context.Background()

	// Missing receiver_email
	_, err := testSvc.SendEmail(ctx, model.SendEmailRequest{
		ReceiverEmail: "", Subject: "Test", Body: "Test",
	})
	if err == nil {
		t.Fatal("expected error for empty receiver_email")
	}
	t.Logf("Correctly rejected empty receiver_email: %v", err)

	// Missing subject
	_, err = testSvc.SendEmail(ctx, model.SendEmailRequest{
		ReceiverEmail: "test@example.com", Subject: "", Body: "Test",
	})
	if err == nil {
		t.Fatal("expected error for empty subject")
	}
	t.Logf("Correctly rejected empty subject: %v", err)

	// Missing body
	_, err = testSvc.SendEmail(ctx, model.SendEmailRequest{
		ReceiverEmail: "test@example.com", Subject: "Test", Body: "",
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	t.Logf("Correctly rejected empty body: %v", err)
}
