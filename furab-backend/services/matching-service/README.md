# Matching Service -join ' ')

Driver-order matching service

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
matching-service/
+-- cmd/main.go
+-- internal/
    +-- handler/match_handler.go
    +-- service/match_service.go
    +-- repository/match_repository.go
    +-- model/match.go
+-- test/
    +-- unit/match_service_test.go
    +-- unit/mock/
    +-- functional/match_functional_test.go
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
export DB_NAME=matching-service

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
--- PASS: TestNewMatchService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewMatchService_Creation
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
docker build -t furab/matching-service:latest -f services/matching-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/matching-service:latest
```
