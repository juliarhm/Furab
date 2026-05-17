# Food Order Service -join ' ')

Food order management service

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
food-order-service/
+-- cmd/main.go
+-- internal/
    +-- handler/foodorder_handler.go
    +-- service/foodorder_service.go
    +-- repository/foodorder_repository.go
    +-- model/foodorder.go
+-- test/
    +-- unit/foodorder_service_test.go
    +-- unit/mock/
    +-- functional/foodorder_functional_test.go
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
export DB_NAME=food-order-service

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
--- PASS: TestNewFoodOrderService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewFoodOrderService_Creation
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
docker build -t furab/food-order-service:latest -f services/food-order-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/food-order-service:latest
```
