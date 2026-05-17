# Driver Service -join ' ')

Driver profile and location management service

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
driver-service/
+-- cmd/main.go
+-- internal/
    +-- handler/driver_handler.go
    +-- service/driver_service.go
    +-- repository/driver_repository.go
    +-- model/driver.go
+-- test/
    +-- unit/driver_service_test.go
    +-- unit/mock/
    +-- functional/driver_functional_test.go
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
export DB_NAME=driver-service

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
--- PASS: TestNewDriverService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewDriverService_Creation
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
docker build -t furab/driver-service:latest -f services/driver-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/driver-service:latest
```
