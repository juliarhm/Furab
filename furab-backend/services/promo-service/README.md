# Promo Service

Promo Service adalah microservice yang bertanggung jawab untuk mengelola seluruh siklus hidup promosi, mulai dari penyimpanan data promo, validasi kode, hingga perhitungan nilai diskon.

## Deskripsi

Promo Service menerima input `total_amount` dari Pricing Service, memvalidasi promo yang masuk, dan menghitung `discount_amount` serta `final_amount` untuk Payment Service.

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Router | chi |
| Database | PostgreSQL |
| Testing | go test |

## Struktur Folder

```
promo-service/
+-- cmd/main.go
+-- internal/
    +-- client/
        +-- order_client.go
        +-- user_client.go
    +-- handler/promo_handler.go
    +-- model/promo.go
    +-- repository/promo_repository.go
    +-- service/promo_service.go
+-- test/
    +-- unit/promo_service_test.go
    +-- unit/mock/
    +-- functional/promo_functional_test.go
+-- go.mod
+-- Dockerfile
+-- README.md
```

## API Endpoint

### POST /api/v1/promos/validate

Request body:

```json
{
  "promo_code": "DISKONHEMAT",
  "user_id": "user-1",
  "order_id": "order-1",
  "total_amount": 100000
}
```

Response:

```json
{
  "success": true,
  "data": {
    "status": "Valid",
    "discount_amount": 10000,
    "final_amount": 90000
  }
}
```

## Cara Menjalankan

```bash
# Set environment variables
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=furab
export DB_PASSWORD=furab_secret
export DB_NAME=promo-service

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
--- PASS: TestNewPromoService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewPromoService_Creation
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
docker build -t furab/promo-service:latest -f services/promo-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/promo-service:latest
```
