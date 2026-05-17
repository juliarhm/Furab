//go:build functional
// +build functional

// Package functional contains functional tests for chat-service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "chat_service" created automatically by init script or manually
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/chat-service/internal/model"
	"furab-backend/services/chat-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB  *sql.DB
	testSvc service.ChatService
)

// --- In-test PostgreSQL repository (implements service.ChatRepository) ---

type testChatRepo struct {
	db *sql.DB
}

func (r *testChatRepo) SaveMessage(ctx context.Context, msg *model.Message) error {
	query := `INSERT INTO messages (message_id, order_id, sender_id, content, read_status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		msg.MessageID, msg.OrderID, msg.SenderID, msg.Content, msg.ReadStatus, msg.Timestamp)
	return err
}

func (r *testChatRepo) UpdateMessageStatus(ctx context.Context, messageID string, status string) error {
	query := `UPDATE messages SET read_status = $1 WHERE message_id = $2`
	_, err := r.db.ExecContext(ctx, query, status, messageID)
	return err
}

func (r *testChatRepo) GetMessagesByOrderID(ctx context.Context, orderID string) ([]model.Message, error) {
	query := `SELECT message_id, order_id, sender_id, content, read_status, timestamp
		FROM messages WHERE order_id = $1 ORDER BY timestamp ASC`
	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.MessageID, &m.OrderID, &m.SenderID, &m.Content, &m.ReadStatus, &m.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *testChatRepo) CloseSession(ctx context.Context, orderID string) error {
	query := `UPDATE chat_sessions SET closed_at = $1 WHERE order_id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), orderID)
	return err
}

// --- Stubs for external service clients ---

type stubUserClient struct{}

func (s *stubUserClient) ValidateUser(ctx context.Context, userID string) (bool, error) {
	return true, nil
}

type stubDriverClient struct{}

func (s *stubDriverClient) ValidateDriver(ctx context.Context, driverID string) (bool, error) {
	return true, nil
}

type stubNotifClient struct{}

func (s *stubNotifClient) SendNotification(ctx context.Context, receiverID string, message string) error {
	return nil
}

// --- TestMain setup ---

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "chat_service")

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

	// Initialize repository and service
	repo := &testChatRepo{db: testDB}
	testSvc = service.NewChatService(repo, &stubUserClient{}, &stubDriverClient{}, &stubNotifClient{})

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS chat_sessions (
			order_id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			driver_id VARCHAR(36) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			closed_at TIMESTAMP WITH TIME ZONE
		);

		CREATE TABLE IF NOT EXISTS messages (
			message_id VARCHAR(36) PRIMARY KEY,
			order_id VARCHAR(36) NOT NULL,
			sender_id VARCHAR(36) NOT NULL,
			content TEXT NOT NULL,
			read_status VARCHAR(20) NOT NULL DEFAULT 'sent',
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_messages_order_id ON messages(order_id);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS messages")
	testDB.Exec("DROP TABLE IF EXISTS chat_sessions")
}

func cleanupData() {
	testDB.Exec("DELETE FROM messages")
	testDB.Exec("DELETE FROM chat_sessions")
}

func seedChatSession(orderID, userID, driverID string) {
	testDB.Exec(`INSERT INTO chat_sessions (order_id, user_id, driver_id, created_at) VALUES ($1, $2, $3, $4)`,
		orderID, userID, driverID, time.Now().UTC())
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_SendAndGetHistory tests sending a message and retrieving chat history from real DB.
func TestFunctional_SendAndGetHistory(t *testing.T) {
	cleanupData()
	ctx := context.Background()

	orderID := "order-chat-001"
	userID := "user-chat-001"
	driverID := "driver-chat-001"

	// Seed chat session
	seedChatSession(orderID, userID, driverID)

	// Send message → INSERT ke PostgreSQL beneran
	req := model.SendMessageRequest{
		OrderID:     orderID,
		SenderID:    userID,
		SenderType:  "user",
		ReceiverID:  driverID,
		MessageText: "Saya di depan gedung A",
	}

	resp, err := testSvc.SendMessage(ctx, req)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if resp.MessageID == "" {
		t.Fatal("expected non-empty message_id")
	}
	if resp.Status != "sent" {
		t.Errorf("expected status sent, got: %s", resp.Status)
	}
	if resp.MessageText != req.MessageText {
		t.Errorf("expected message '%s', got: '%s'", req.MessageText, resp.MessageText)
	}
	t.Logf("Sent message: %s (status: %s)", resp.MessageID, resp.Status)

	// Get chat history → SELECT dari PostgreSQL beneran
	history, err := testSvc.GetChatHistory(ctx, orderID)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 message in history, got: %d", len(history))
	}
	if history[0].MessageID != resp.MessageID {
		t.Errorf("expected message_id %s, got: %s", resp.MessageID, history[0].MessageID)
	}
	if history[0].Content != req.MessageText {
		t.Errorf("expected content '%s', got: '%s'", req.MessageText, history[0].Content)
	}
	t.Logf("History contains %d message(s)", len(history))
}

// TestFunctional_ChatLifecycle tests the complete chat lifecycle.
// Flow: user sends → driver replies → read receipt → close session
func TestFunctional_ChatLifecycle(t *testing.T) {
	cleanupData()
	ctx := context.Background()

	orderID := "order-chat-002"
	userID := "user-chat-002"
	driverID := "driver-chat-002"

	seedChatSession(orderID, userID, driverID)

	// Step 1: User sends message
	userMsg, err := testSvc.SendMessage(ctx, model.SendMessageRequest{
		OrderID: orderID, SenderID: userID, SenderType: "user",
		ReceiverID: driverID, MessageText: "Halo, saya tunggu di lobby",
	})
	if err != nil {
		t.Fatalf("step 1 - user send: %v", err)
	}
	t.Logf("Step 1 - User sent: %s", userMsg.MessageID)

	// Step 2: Driver replies
	driverMsg, err := testSvc.SendMessage(ctx, model.SendMessageRequest{
		OrderID: orderID, SenderID: driverID, SenderType: "driver",
		ReceiverID: userID, MessageText: "Baik, saya 5 menit lagi sampai",
	})
	if err != nil {
		t.Fatalf("step 2 - driver reply: %v", err)
	}
	t.Logf("Step 2 - Driver replied: %s", driverMsg.MessageID)

	// Step 3: Update read receipt
	err = testSvc.UpdateMessageStatus(ctx, model.ReadReceiptRequest{
		MessageID: userMsg.MessageID, OrderID: orderID, Status: "read",
	})
	if err != nil {
		t.Fatalf("step 3 - read receipt: %v", err)
	}
	t.Log("Step 3 - Read receipt updated")

	// Verify read status in DB
	var readStatus string
	err = testDB.QueryRow("SELECT read_status FROM messages WHERE message_id = $1", userMsg.MessageID).Scan(&readStatus)
	if err != nil {
		t.Fatalf("failed to verify read status: %v", err)
	}
	if readStatus != "read" {
		t.Errorf("expected read status 'read', got: %s", readStatus)
	}

	// Step 4: Verify chat history order
	history, err := testSvc.GetChatHistory(ctx, orderID)
	if err != nil {
		t.Fatalf("step 4 - get history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got: %d", len(history))
	}
	if history[0].SenderID != userID {
		t.Errorf("expected first message from user, got: %s", history[0].SenderID)
	}
	if history[1].SenderID != driverID {
		t.Errorf("expected second message from driver, got: %s", history[1].SenderID)
	}
	t.Logf("Step 4 - History: %d messages, ordered correctly", len(history))

	// Step 5: Close chat session
	err = testSvc.CloseChatSession(ctx, orderID)
	if err != nil {
		t.Fatalf("step 5 - close session: %v", err)
	}

	var closedAt sql.NullTime
	err = testDB.QueryRow("SELECT closed_at FROM chat_sessions WHERE order_id = $1", orderID).Scan(&closedAt)
	if err != nil {
		t.Fatalf("failed to verify session close: %v", err)
	}
	if !closedAt.Valid {
		t.Error("expected closed_at to be set")
	}
	t.Logf("Step 5 - Session closed at: %v", closedAt.Time)
}

// TestFunctional_InvalidRequest tests that invalid requests are rejected.
func TestFunctional_InvalidRequest(t *testing.T) {
	cleanupData()
	ctx := context.Background()

	// Empty order_id
	_, err := testSvc.SendMessage(ctx, model.SendMessageRequest{
		OrderID: "", SenderID: "user-001", SenderType: "user",
		ReceiverID: "driver-001", MessageText: "test",
	})
	if err == nil {
		t.Fatal("expected error for empty order_id")
	}
	t.Logf("Correctly rejected empty order_id: %v", err)

	// Empty message text
	_, err = testSvc.SendMessage(ctx, model.SendMessageRequest{
		OrderID: "order-001", SenderID: "user-001", SenderType: "user",
		ReceiverID: "driver-001", MessageText: "",
	})
	if err == nil {
		t.Fatal("expected error for empty message_text")
	}
	t.Logf("Correctly rejected empty message_text: %v", err)

	// Invalid sender type
	_, err = testSvc.SendMessage(ctx, model.SendMessageRequest{
		OrderID: "order-001", SenderID: "user-001", SenderType: "admin",
		ReceiverID: "driver-001", MessageText: "test",
	})
	if err == nil {
		t.Fatal("expected error for invalid sender_type")
	}
	t.Logf("Correctly rejected invalid sender_type: %v", err)

	// Empty order_id for GetChatHistory
	_, err = testSvc.GetChatHistory(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty order_id in GetChatHistory")
	}
	t.Logf("Correctly rejected empty order_id in GetChatHistory: %v", err)
}

// TestFunctional_MultipleMessages tests sending multiple messages and verifying order.
func TestFunctional_MultipleMessages(t *testing.T) {
	cleanupData()
	ctx := context.Background()

	orderID := "order-chat-003"
	userID := "user-chat-003"
	driverID := "driver-chat-003"

	seedChatSession(orderID, userID, driverID)

	// Send 5 messages alternating between user and driver
	messages := []struct {
		senderID   string
		senderType string
		receiverID string
		text       string
	}{
		{userID, "user", driverID, "Halo driver"},
		{driverID, "driver", userID, "Halo, saya menuju ke lokasi"},
		{userID, "user", driverID, "Saya di gedung biru"},
		{driverID, "driver", userID, "Baik, 3 menit lagi"},
		{userID, "user", driverID, "Terima kasih"},
	}

	for i, msg := range messages {
		_, err := testSvc.SendMessage(ctx, model.SendMessageRequest{
			OrderID: orderID, SenderID: msg.senderID, SenderType: msg.senderType,
			ReceiverID: msg.receiverID, MessageText: msg.text,
		})
		if err != nil {
			t.Fatalf("send message %d: %v", i+1, err)
		}
		// Small delay to ensure timestamp ordering
		time.Sleep(10 * time.Millisecond)
	}
	t.Log("Sent 5 messages")

	// Get history
	history, err := testSvc.GetChatHistory(ctx, orderID)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}

	if len(history) != 5 {
		t.Fatalf("expected 5 messages, got: %d", len(history))
	}

	// Verify order matches send order
	for i, msg := range history {
		if msg.Content != messages[i].text {
			t.Errorf("message %d: expected '%s', got: '%s'", i+1, messages[i].text, msg.Content)
		}
		t.Logf("  [%d] %s: %s", i+1, msg.SenderID, msg.Content)
	}
}

// TestFunctional_ReadReceiptValidation tests that invalid read receipt statuses are rejected.
func TestFunctional_ReadReceiptValidation(t *testing.T) {
	cleanupData()
	ctx := context.Background()

	// Empty message_id
	err := testSvc.UpdateMessageStatus(ctx, model.ReadReceiptRequest{
		MessageID: "", OrderID: "order-001", Status: "read",
	})
	if err == nil {
		t.Fatal("expected error for empty message_id")
	}
	t.Logf("Correctly rejected empty message_id: %v", err)

	// Invalid status value
	err = testSvc.UpdateMessageStatus(ctx, model.ReadReceiptRequest{
		MessageID: "msg-001", OrderID: "order-001", Status: "invalid_status",
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	t.Logf("Correctly rejected invalid status: %v", err)
}
