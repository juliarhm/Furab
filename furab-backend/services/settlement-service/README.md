# Settlement Service -join ' ')

Driver settlement and payout service

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
settlement-service/
+-- cmd/main.go
+-- internal/
    +-- handler/settlement_handler.go
    +-- service/settlement_service.go
    +-- repository/settlement_repository.go
    +-- model/settlement.go
+-- test/
    +-- unit/settlement_service_test.go
    +-- unit/mock/
    +-- functional/settlement_functional_test.go
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
export DB_NAME=settlement-service

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
--- PASS: TestNewSettlementService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewSettlementService_Creation
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
docker build -t furab/settlement-service:latest -f services/settlement-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/settlement-service:latest
```
