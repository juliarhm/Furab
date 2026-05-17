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

	"furab-backend/services/audit-log-service/internal/model"
	"furab-backend/services/audit-log-service/internal/repository"
	"furab-backend/services/audit-log-service/internal/service"

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
	_, err = adminDB.Exec("CREATE DATABASE audit_log_service")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("Failed to create database: %v", err)
	}
	adminDB.Close()

	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5432 user=furab password=furab_secret dbname=audit_log_service sslmode=disable"
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
		log.Printf("Waiting for audit_log_service database... attempt %d/30: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to audit_log_service database after 30 attempts: %v", err)
	}
	log.Println("Database connection to audit_log_service established successfully.")

	setupSchema()
	code := m.Run()
	teardownSchema()
	testDB.Close()
	os.Exit(code)
}

func setupSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		action TEXT NOT NULL,
		entity TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		old_data TEXT NOT NULL,
		new_data TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL
	);`
	if _, err := testDB.Exec(query); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}
}

func teardownSchema() {
	if _, err := testDB.Exec("DROP TABLE IF EXISTS audit_logs"); err != nil {
		log.Printf("Failed to teardown schema: %v", err)
	}
}

func cleanupAuditLogs() {
	if _, err := testDB.Exec("DELETE FROM audit_logs"); err != nil {
		log.Fatalf("Failed to cleanup audit_logs: %v", err)
	}
}

func auditLogRequest() *model.AuditLog {
	return &model.AuditLog{
		UserID:    "user-001",
		Action:    "CREATE",
		Entity:    "merchant",
		EntityID:  "merchant-123",
		OldData:   "{}",
		NewData:   `{"name":"Warung Makan"}`,
		IPAddress: "192.168.1.100",
		UserAgent: "FurabApp/1.0 Android",
	}
}

func TestFunctional_AuditLog_CreateAndSearch(t *testing.T) {
	cleanupAuditLogs()
	repo := repository.NewAuditLogRepository(testDB)
	svc := service.NewAuditLogService(repo)
	ctx := context.Background()

	req := auditLogRequest()
	err := svc.CreateAuditLog(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.ID)
	assert.False(t, req.CreatedAt.IsZero())

	t.Run("SearchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{UserID: "user-001", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, req.ID, results[0].ID)
		assert.Equal(t, req.UserID, results[0].UserID)
		assert.Equal(t, req.Action, results[0].Action)
		assert.Equal(t, req.Entity, results[0].Entity)
		assert.Equal(t, req.EntityID, results[0].EntityID)
		assert.Equal(t, req.OldData, results[0].OldData)
		assert.Equal(t, req.NewData, results[0].NewData)
		assert.Equal(t, req.IPAddress, results[0].IPAddress)
		assert.Equal(t, req.UserAgent, results[0].UserAgent)
	})

	t.Run("SearchByAction", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{Action: "CREATE", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})

	t.Run("SearchByEntity", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{Entity: "merchant", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})

	t.Run("SearchByDateRange", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{
			StartDate: time.Now().Add(-1 * time.Hour),
			EndDate:   time.Now().Add(1 * time.Hour),
			Limit:     10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
	})
}

func TestFunctional_AuditLog_SearchNoMatch(t *testing.T) {
	cleanupAuditLogs()
	repo := repository.NewAuditLogRepository(testDB)
	svc := service.NewAuditLogService(repo)
	ctx := context.Background()

	err := svc.CreateAuditLog(ctx, auditLogRequest())
	assert.NoError(t, err)

	t.Run("NoMatchByUserID", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{UserID: "non-existent", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByAction", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{Action: "ARCHIVE", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByEntity", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{Entity: "spaceship", Limit: 10})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})

	t.Run("NoMatchByDateRange", func(t *testing.T) {
		results, total, err := svc.SearchAuditLogs(ctx, model.SearchAuditLogRequest{
			StartDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2000, 12, 31, 23, 59, 59, 0, time.UTC),
			Limit:     10,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, results)
	})
}

func TestFunctional_AuditLog_EdgeCases(t *testing.T) {
	cleanupAuditLogs()
	repo := repository.NewAuditLogRepository(testDB)
	svc := service.NewAuditLogService(repo)
	ctx := context.Background()

	t.Run("CreateWithEmptyUserID", func(t *testing.T) {
		req := auditLogRequest()
		req.UserID = ""
		err := svc.CreateAuditLog(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user id is required")
	})

	t.Run("CreateWithEmptyAction", func(t *testing.T) {
		req := auditLogRequest()
		req.Action = ""
		err := svc.CreateAuditLog(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action is required")
	})

	t.Run("CreateWithEmptyEntity", func(t *testing.T) {
		req := auditLogRequest()
		req.Entity = ""
		err := svc.CreateAuditLog(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entity is required")
	})

	t.Run("GetWithEmptyID", func(t *testing.T) {
		_, err := svc.GetAuditLog(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audit log id is required")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := svc.GetAuditLog(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audit log not found")
	})

	t.Run("DeleteWithEmptyID", func(t *testing.T) {
		err := svc.DeleteAuditLog(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audit log id is required")
	})
}

func TestFunctional_AuditLog_FullLifecycle(t *testing.T) {
	cleanupAuditLogs()
	repo := repository.NewAuditLogRepository(testDB)
	svc := service.NewAuditLogService(repo)
	ctx := context.Background()

	req := auditLogRequest()
	err := svc.CreateAuditLog(ctx, req)
	assert.NoError(t, err)

	created, err := svc.GetAuditLog(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "CREATE", created.Action)

	err = svc.UpdateAuditLog(ctx, &model.AuditLog{
		ID: req.ID, Action: "UPDATE", Entity: "merchant", EntityID: "merchant-456",
		OldData: `{"name":"Warung Makan"}`, NewData: `{"name":"Warung Premium"}`,
	})
	assert.NoError(t, err)

	updated, err := svc.GetAuditLog(ctx, req.ID)
	assert.NoError(t, err)
	assert.Equal(t, "UPDATE", updated.Action)
	assert.Equal(t, "merchant-456", updated.EntityID)

	err = svc.DeleteAuditLog(ctx, req.ID)
	assert.NoError(t, err)

	_, err = svc.GetAuditLog(ctx, req.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "audit log not found")
}
