# Audit Log Service -join ' ')

Audit trail and logging service

## Deskripsi

TODO: Tambahkan deskripsi lengkap service ini.

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Router | chi |
| Database | PostgreSQL |
| Testing | gomock, go test |

## Struktur Folder

```
audit-log-service/
+-- cmd/main.go
+-- internal/
    +-- handler/auditlog_handler.go
    +-- service/auditlog_service.go
    +-- repository/auditlog_repository.go
    +-- model/auditlog.go
+-- test/
    +-- unit/auditlog_service_test.go
    +-- unit/mock/
    +-- functional/auditlog_functional_test.go
+-- go.mod
+-- Dockerfile
+-- README.md
```

## Cara Menjalankan

```bash
# Set environment variables
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=furab
export DB_PASSWORD=furab_secret
export DB_NAME=audit-log-service

# Jalankan service
go run cmd/main.go
```

## Menjalankan Tests

### Unit Tests (Tanpa Database)
```bash
go test ./test/unit/... -v
```

**Test BERHASIL jika output:**
```
--- PASS: TestNewAuditLogService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewAuditLogService_Creation
FAIL
```

### Functional Tests (Dengan Database)
```bash
# Pastikan PostgreSQL berjalan
go test ./test/functional/... -v -tags=functional
```

## Docker

```bash
# Build (dari root project)
docker build -t furab/audit-log-service:latest -f services/audit-log-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/audit-log-service:latest
```
