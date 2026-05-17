# Ride Order Service

Service untuk mengelola pesanan ride (ojek online) di platform Furab.

## 📋 Deskripsi

Ride Order Service menangani **lifecycle lengkap** dari pemesanan ride:

```
PENDING → ASSIGNED → STARTED → COMPLETED
    ↓         ↓
 CANCELLED  CANCELLED
```

### Fitur Utama
- ✅ Create ride order dengan estimasi tarif otomatis
- ✅ Assign driver ke order
- ✅ Status transition dengan validasi state machine
- ✅ Ride lifecycle management (start, complete, cancel)
- ✅ Pagination untuk riwayat order
- ✅ Event publishing untuk inter-service communication

## 🛠️ Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Router | [chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL |
| Message Broker | Apache Kafka |
| Testing | gomock, go test |
| UUID | google/uuid |

## 📁 Struktur Folder

```
ride-order-service/
├── cmd/
│   └── main.go              # Entry point (bootstrap & wiring)
├── internal/
│   ├── handler/
│   │   └── order_handler.go  # HTTP handlers
│   ├── service/
│   │   └── order_service.go  # Business logic
│   ├── repository/
│   │   └── order_repository.go # Data access layer (PostgreSQL)
│   └── model/
│       └── order.go          # Domain models & DTOs
├── test/
│   ├── unit/
│   │   ├── order_service_test.go  # Unit tests (mocked)
│   │   └── mock/
│   │       ├── mock_order_repository.go
│   │       └── mock_event_publisher.go
│   └── functional/
│       └── order_functional_test.go  # Functional tests (real DB)
├── migrations/
│   └── 001_create_ride_orders.sql
├── api/
│   └── swagger.yaml          # OpenAPI 3.0 specification
├── go.mod
├── Dockerfile
└── README.md
```

## 🚀 Cara Menjalankan

### Prerequisites
- Go 1.22+
- PostgreSQL
- Apache Kafka (optional, untuk event publishing)

### 1. Setup Database

```bash
# Buat database
createdb -U postgres ride_order_service

# Jalankan migration
psql -U postgres -d ride_order_service -f migrations/001_create_ride_orders.sql
```

### 2. Set Environment Variables

```bash
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=furab
export DB_PASSWORD=furab_secret
export DB_NAME=ride_order_service
export KAFKA_BROKERS=localhost:9092
export ENVIRONMENT=development
```

### 3. Jalankan Service

```bash
# Dari folder ride-order-service
go run cmd/main.go

# Output yang diharapkan:
# time=2024-01-01T00:00:00.000Z level=INFO msg="starting ride-order-service" service=ride-order-service port=8080
# time=2024-01-01T00:00:00.000Z level=INFO msg="connected to database"
# time=2024-01-01T00:00:00.000Z level=INFO msg="server listening" address=0.0.0.0:8080
```

## 🧪 Menjalankan Tests

### Unit Tests (Tanpa Database)

Unit test **TIDAK memerlukan** database atau service external apapun. Semua dependency di-mock menggunakan `gomock`.

```bash
# Dari folder ride-order-service
go test ./test/unit/... -v

# Atau dari root project
go test ./services/ride-order-service/test/unit/... -v
```

**Cara mengetahui unit test BERHASIL:**
```
=== RUN   TestCreateOrder_Success
--- PASS: TestCreateOrder_Success (0.00s)
=== RUN   TestCreateOrder_NilRequest
--- PASS: TestCreateOrder_NilRequest (0.00s)
...
PASS
ok      furab-backend/services/ride-order-service/test/unit    0.015s
```

**Jika GAGAL:**
```
=== RUN   TestCreateOrder_Success
--- FAIL: TestCreateOrder_Success (0.00s)
    order_service_test.go:95: expected status PENDING, got: ASSIGNED
FAIL
exit status 1
```

**Test coverage:**
```bash
go test ./test/unit/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
# Buka coverage.html di browser untuk melihat detail coverage
```

### Functional Tests (Dengan Database)

Functional test **MEMERLUKAN** PostgreSQL yang berjalan. Test akan membuat tabel sendiri dan membersihkannya setelah selesai.

```bash
# 1. Pastikan PostgreSQL berjalan
docker run -d --name pg-test \
  -e POSTGRES_USER=furab \
  -e POSTGRES_PASSWORD=furab_secret \
  -e POSTGRES_DB=ride_order_service_test \
  -p 5432:5432 postgres:16-alpine

# 2. Jalankan functional test
go test ./test/functional/... -v -tags=functional

# Atau dengan custom DB config
DB_HOST=localhost DB_PORT=5432 DB_USER=furab DB_PASSWORD=furab_secret DB_NAME=ride_order_service_test \
  go test ./test/functional/... -v -tags=functional
```

**Cara mengetahui functional test BERHASIL:**
```
=== RUN   TestFunctional_CreateAndGetOrder
    order_functional_test.go:XX: Created order: 550e...(status: PENDING)
--- PASS: TestFunctional_CreateAndGetOrder (0.05s)
=== RUN   TestFunctional_FullRideFlow
    order_functional_test.go:XX: Ride completed (status: COMPLETED, fare: Rp 18500)
--- PASS: TestFunctional_FullRideFlow (0.08s)
...
PASS
ok      furab-backend/services/ride-order-service/test/functional    0.25s
```

**Jika GAGAL** (contoh: database tidak tersedia):
```
--- FAIL: TestFunctional_CreateAndGetOrder (0.00s)
    order_functional_test.go:XX: failed to create order: connection refused
FAIL
```

### Daftar Test Cases

#### Unit Tests (15 test cases)

| # | Test | Deskripsi | Status |
|---|------|-----------|--------|
| 1 | `TestCreateOrder_Success` | Buat order valid → PENDING | ✅ |
| 2 | `TestCreateOrder_NilRequest` | Request nil → error | ✅ |
| 3 | `TestCreateOrder_InvalidPickup` | Pickup kosong → error | ✅ |
| 4 | `TestCreateOrder_InvalidDropoff` | Dropoff kosong → error | ✅ |
| 5 | `TestCreateOrder_EmptyUserID` | UserID kosong → error | ✅ |
| 6 | `TestGetOrder_Success` | Get order → data benar | ✅ |
| 7 | `TestGetOrder_NotFound` | Order tidak ada → error | ✅ |
| 8 | `TestGetOrder_EmptyID` | ID kosong → error | ✅ |
| 9 | `TestAssignDriver_Success` | Assign driver → ASSIGNED | ✅ |
| 10 | `TestAssignDriver_InvalidStatus` | Assign ke COMPLETED → error | ✅ |
| 11 | `TestAssignDriver_AlreadyAssigned` | Double assign → error | ✅ |
| 12 | `TestStartRide_Success` | Start ASSIGNED → STARTED | ✅ |
| 13 | `TestStartRide_InvalidStatus` | Start PENDING → error | ✅ |
| 14 | `TestCompleteRide_Success` | Complete STARTED → COMPLETED | ✅ |
| 15 | `TestCompleteRide_InvalidStatus` | Complete PENDING → error | ✅ |
| 16 | `TestCancelRide_Success` | Cancel PENDING → CANCELLED | ✅ |
| 17 | `TestCancelRide_AlreadyCompleted` | Cancel COMPLETED → error | ✅ |
| 18 | `TestGetUserOrders_Success` | Get multiple orders | ✅ |
| 19 | `TestGetUserOrders_Empty` | No orders → empty list | ✅ |

#### Functional Tests (5 test cases)

| # | Test | Deskripsi | Butuh DB |
|---|------|-----------|----------|
| 1 | `TestFunctional_CreateAndGetOrder` | Create → Get → verify | ✅ |
| 2 | `TestFunctional_FullRideFlow` | Full lifecycle test | ✅ |
| 3 | `TestFunctional_CancelRide` | Cancel flow | ✅ |
| 4 | `TestFunctional_InvalidTransition` | Invalid status change | ✅ |
| 5 | `TestFunctional_GetUserOrders` | Pagination test | ✅ |

## 📡 API Endpoints

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/rides` | Buat ride order baru |
| `GET` | `/api/v1/rides/{orderID}` | Ambil detail ride |
| `PUT` | `/api/v1/rides/{orderID}/assign` | Assign driver |
| `PUT` | `/api/v1/rides/{orderID}/start` | Mulai ride |
| `PUT` | `/api/v1/rides/{orderID}/complete` | Selesaikan ride |
| `PUT` | `/api/v1/rides/{orderID}/cancel` | Batalkan ride |
| `GET` | `/api/v1/rides/user/{userID}` | Ambil semua ride user |

### Contoh Request

```bash
# Create ride order
curl -X POST http://localhost:8080/api/v1/rides \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "pickup_location": {
      "latitude": -6.2088,
      "longitude": 106.8456,
      "address": "Monas, Jakarta Pusat"
    },
    "dropoff_location": {
      "latitude": -6.1751,
      "longitude": 106.8650,
      "address": "Ancol, Jakarta Utara"
    }
  }'
```

## 📡 Events (Kafka)

| Event | Kapan dipublish |
|-------|----------------|
| `ride.created` | Setelah order dibuat |
| `ride.assigned` | Setelah driver di-assign |
| `ride.started` | Setelah ride dimulai |
| `ride.completed` | Setelah ride selesai |

## 🔧 Environment Variables

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `SERVER_PORT` | `8080` | Port HTTP server |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `furab` | PostgreSQL user |
| `DB_PASSWORD` | `furab_secret` | PostgreSQL password |
| `DB_NAME` | `ride-order-service` | Database name |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `ENVIRONMENT` | `development` | Runtime environment |

## 📦 Docker

```bash
# Build image (dari root project)
docker build -t furab/ride-order-service:latest \
  -f services/ride-order-service/Dockerfile .

# Run container
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  furab/ride-order-service:latest
```
