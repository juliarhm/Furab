# Payment Service

Orchestrator pembayaran stateless untuk platform Furab dengan implementasi Two-Phase Payment (Authorize & Capture).

## 📋 Deskripsi

Payment Service menangani **lifecycle lengkap** transaksi pembayaran:

```
PENDING → AUTHORIZED → CAPTURED → REFUNDED
    ↓          ↓           ↓
 FAILED    FAILED      FAILED
    ↓          ↓
 CANCELLED  CANCELLED
```

### Fitur Utama
- ✅ Two-Phase Payment Flow (Authorize & Capture)
- ✅ Koordinasi dengan Wallet Service (Lock/Unlock/Deduct/Credit)
- ✅ Koordinasi dengan Pricing & Promo Service untuk kalkulasi harga final
- ✅ Settlement Service integration untuk distribusi dana
- ✅ Idempotency Key untuk duplicate prevention
- ✅ Audit Trail dengan Payment Logs
- ✅ Payment Methods & Providers management

## 🛠️ Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Router | [chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL |
| Testing | gomock, go test |
| UUID | google/uuid |

## 📁 Struktur Folder

```
payment-service/
├── cmd/
│   └── main.go                      # Entry point (bootstrap & wiring)
├── internal/
│   ├── handler/
│   │   └── payment_handler.go       # HTTP handlers & routing
│   ├── service/
│   │   └── payment_service.go       # Business logic & orchestration
│   ├── repository/
│   │   └── payment_repository.go    # Data access layer (PostgreSQL)
│   └── model/
│       └── payment.go               # Domain models & DTOs
├── test/
│   ├── unit/
│   │   ├── payment_service_test.go  # Unit tests (mocked dependencies)
│   │   └── mock/
│   │       └── mock_payment_repository.go
│   └── functional/
│       └── payment_functional_test.go  # Functional tests (real DB)
├── migrations/
│   └── 001_create_payments.sql      # Database schema
├── go.mod
├── Dockerfile
└── README.md
```

## 🚀 Cara Menjalankan

### Prerequisites
- Go 1.22+
- PostgreSQL
- Koordinasi dengan Wallet, Pricing, Promo, & Settlement Services

### 1. Setup Database

```bash
# Buat database
createdb -U postgres payment_service

# Jalankan migration
psql -U postgres -d payment_service -f migrations/001_create_payments.sql
```

### 2. Set Environment Variables

```bash
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=furab
export DB_PASSWORD=furab_secret
export DB_NAME=payment_service
export ENVIRONMENT=development
```

### 3. Jalankan Service

```bash
# Dari folder payment-service
go run cmd/main.go

# Output yang diharapkan:
# time=2024-01-01T00:00:00.000Z level=INFO msg="starting payment-service" service=payment-service port=8080
# time=2024-01-01T00:00:00.000Z level=INFO msg="connected to database"
# time=2024-01-01T00:00:00.000Z level=INFO msg="server listening" address=0.0.0.0:8080
```

## 🧪 Menjalankan Tests

### Unit Tests (Tanpa Database)

Unit test **TIDAK memerlukan** database atau service external apapun. Semua dependency di-mock.

```bash
# Dari folder payment-service
go test ./test/unit/... -v

# Atau dari root project
go test ./services/payment-service/test/unit/... -v
```

**Cara mengetahui unit test BERHASIL:**
```
=== RUN   TestInitiatePayment_Success
--- PASS: TestInitiatePayment_Success (0.00s)
=== RUN   TestCapturePayment_Success
--- PASS: TestCapturePayment_Success (0.00s)
...
PASS
ok      furab-backend/services/payment-service/test/unit    0.025s
```

**Test coverage:**
```bash
go test ./test/unit/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 📡 API Endpoints

### Health Check
```
GET /api/v1/payments/health
```

### Initiate Payment (Phase 1: Authorize)
```
POST /api/v1/payments/
Content-Type: application/json

{
  "order_id": "order-123",
  "user_id": "user-456",
  "payment_method": "credit-card-001",
  "payment_detail": "{...encrypted details...}",
  "promo_code": "PROMO2024",
  "amount": 100000,
  "idempotency_key": "unique-key-from-client"
}

Response (201 Created):
{
  "data": {
    "payment_id": "payment-789",
    "order_id": "order-123",
    "user_id": "user-456",
    "amount": 100000,
    "final_amount": 85000,
    "payment_method": "credit-card-001",
    "payment_status": "authorized",
    "transaction_reference": "TXN-order-123",
    "transaction_time": "2024-01-01T10:00:00Z",
    "created_at": "2024-01-01T10:00:00Z"
  },
  "message": "payment initiated successfully"
}
```

### Get Payment
```
GET /api/v1/payments/{paymentID}

Response (200 OK):
{
  "data": {
    "payment_id": "payment-789",
    ...
  }
}
```

### Capture Payment (Phase 2: Capture)
```
PUT /api/v1/payments/{paymentID}/capture

Response (200 OK):
{
  "data": {
    "payment_id": "payment-789",
    "payment_status": "captured",
    ...
  },
  "message": "payment captured successfully"
}
```

### Cancel Payment
```
PUT /api/v1/payments/{paymentID}/cancel

Response (200 OK):
{
  "data": {
    "payment_id": "payment-789",
    "payment_status": "cancelled",
    ...
  },
  "message": "payment cancelled successfully"
}
```

### Refund Payment
```
PUT /api/v1/payments/{paymentID}/refund

Response (200 OK):
{
  "data": {
    "payment_id": "payment-789",
    "payment_status": "refunded",
    ...
  },
  "message": "payment refunded successfully"
}
```

## 🔄 Interaksi Microservice

### Wallet Service
- **LockBalance**: Pre-authorize saldo during payment initiation
- **UnlockBalance**: Release lock jika payment cancelled
- **DeductBalance**: Deduct saldo ketika payment captured
- **CreditBalance**: Return saldo kepada user untuk refund

### Pricing & Promo Service
- **GetTotalAmount**: Ambil base amount dari order
- **ApplyPromo**: Kalkulasi final amount setelah diskon

### Settlement Service
- **TriggerSettlement**: Distribute dana ke driver/merchant setelah payment captured

## 📊 Database Schema

### payments table
| Column | Type | Notes |
|--------|------|-------|
| id | VARCHAR(36) | Primary Key (UUID) |
| order_id | VARCHAR(36) | Reference to order |
| user_id | VARCHAR(36) | User initiating payment |
| amount | DOUBLE PRECISION | Original amount (IDR) |
| final_amount | DOUBLE PRECISION | Amount after discounts |
| method_id | VARCHAR(36) | FK to payment_methods |
| payment_detail | TEXT | Serialized method details |
| payment_status | VARCHAR(20) | pending, authorized, captured, refunded, failed, cancelled |
| transaction_reference | VARCHAR(100) | TXN-{order_id} |
| idempotency_key | VARCHAR(100) | UNIQUE for idempotency |
| transaction_time | TIMESTAMP | Payment timestamp |
| created_at | TIMESTAMP | Record creation time |
| updated_at | TIMESTAMP | Last update time |

### payment_logs table (Audit Trail)
| Column | Type | Notes |
|--------|------|-------|
| id | SERIAL | Primary Key |
| payment_id | VARCHAR(36) | FK to payments |
| status | VARCHAR(20) | Status snapshot |
| timestamp | TIMESTAMP | When change occurred |

### payment_methods table
| Column | Type | Notes |
|--------|------|-------|
| id | VARCHAR(36) | Primary Key |
| method_name | VARCHAR(100) | e.g., Visa, GCash |
| provider | VARCHAR(50) | e.g., Stripe, Xendit |
| created_at | TIMESTAMP | Record creation |
| updated_at | TIMESTAMP | Last update |

## ⚡ Karakteristik Teknis

- **High Availability**: Stateless design, horizontal scalable
- **High Concurrency**: Optimized PostgreSQL queries with proper indexes
- **Real-time Processing**: Synchronous two-phase payment flow
- **Source of Truth**: Wallet Service untuk saldo balance
- **Idempotency**: Client-provided idempotency key untuk duplicate prevention
- **Audit Trail**: Semua state changes tercatat di payment_logs

## Menjalankan Tests

### Unit Tests (Tanpa Database)
```bash
go test ./test/unit/... -v
```

**Test BERHASIL jika output:**
```
--- PASS: TestNewPaymentService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewPaymentService_Creation
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
docker build -t furab/payment-service:latest -f services/payment-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/payment-service:latest
```
