# Furab Backend - Microservice Architecture

Furab adalah super-app yang menggabungkan layanan **ride-hailing** dan **food delivery** dalam satu platform, mirip Gojek/Grab.

## 🏗️ Architecture

Backend dibangun menggunakan **Go** dengan arsitektur **microservice**, event-driven menggunakan **Kafka** (high-throughput events) dan **RabbitMQ** (transactional events).

## 📁 Project Structure

```
furab-backend/
├── services/           # Semua microservices
│   ├── auth-service/       # Authentication & authorization
│   ├── otp-service/        # OTP generation & verification
│   ├── user-service/       # User profile management
│   ├── driver-service/     # Driver profile & availability
│   ├── ride-order-service/ # ⭐ Ride order management (fully implemented)
│   ├── food-order-service/ # Food order management
│   ├── cart-service/       # Shopping cart
│   ├── matching-service/   # Driver-order matching
│   ├── payment-service/    # Payment processing
│   ├── wallet-service/     # Digital wallet
│   ├── settlement-service/ # Driver settlement
│   ├── pricing-service/    # Dynamic pricing & surge
│   ├── promo-service/      # Promotions & discounts
│   ├── location-service/   # Real-time location tracking
│   ├── chat-service/       # In-app messaging
│   ├── notification-service/ # Push notifications
│   ├── email-service/      # Email delivery
│   ├── emergency-service/  # SOS & emergency
│   ├── merchant-service/   # Merchant management
│   ├── menu-service/       # Restaurant menu
│   ├── rating-service/     # Star ratings
│   ├── review-service/     # Text reviews
│   └── audit-log-service/  # Audit trail
├── shared/             # Shared libraries
├── gateway/            # API Gateway
├── deploy/             # Docker, Kubernetes, Helm configs
├── scripts/            # Build & test scripts
├── tests/              # Cross-service functional tests
├── go.work             # Go workspace
└── Jenkinsfile         # CI/CD pipeline
```

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- PostgreSQL (or use docker-compose)

### Run Locally
```bash
# Start infrastructure (DB, Kafka, RabbitMQ)
docker-compose -f deploy/docker/docker-compose.yml up -d

# Run a specific service
cd services/ride-order-service
go run cmd/main.go
```

### Run Tests
```bash
# Unit tests (no DB required)
go test ./services/ride-order-service/test/unit/... -v

# Functional tests (DB required)
go test ./services/ride-order-service/test/functional/... -v -tags=functional

# All unit tests
./scripts/run-unit-tests.sh

# Lint & vet
./scripts/run-lint.sh
```

## 🔄 CI/CD Pipeline

| Stage | Command | Description |
|-------|---------|-------------|
| 1. Checkout | `git checkout` | Pull latest code |
| 2. Unit Tests | `go test ./test/unit/...` | Run unit tests (no DB) |
| 3. Lint/Vet | `go vet ./...` | Static analysis |
| 4. Build Image | `docker build` | Build Docker images |
| 5. Functional Tests | `go test ./test/functional/...` | Run with real DB |
| 6. Push Image | `docker push` | Push to registry |
| 7. Deploy | `kubectl apply` | Deploy to Kubernetes |
| 8. Verify | Health check | Verify deployment |

## 📡 Events

| Event | Producer | Consumers |
|-------|----------|-----------|
| `ride.created` | ride-order-service | matching-service, notification-service |
| `ride.assigned` | matching-service | ride-order-service, notification-service |
| `ride.started` | driver-service | ride-order-service, location-service |
| `ride.completed` | ride-order-service | payment-service, rating-service |
| `food.created` | food-order-service | merchant-service, notification-service |
| `food.confirmed` | merchant-service | food-order-service, notification-service |
| `food.preparing` | merchant-service | food-order-service |
| `food.ready` | merchant-service | matching-service, notification-service |
| `food.picked_up` | driver-service | food-order-service, notification-service |
| `food.completed` | food-order-service | payment-service, rating-service |
| `payment.authorized` | payment-service | order services |
| `payment.captured` | payment-service | wallet-service, settlement-service |
| `payment.failed` | payment-service | order services, notification-service |
| `settlement.completed` | settlement-service | wallet-service, notification-service |
| `notification.sent` | notification-service | audit-log-service |

## 🛠️ Tech Stack

- **Language**: Go 1.22+
- **HTTP Router**: chi
- **Database**: PostgreSQL (database per service)
- **Message Broker**: Kafka (high-throughput) + RabbitMQ (transactional)
- **Testing**: gomock (unit), testify (assertions)
- **Container**: Docker
- **Orchestration**: Kubernetes + Helm
- **CI/CD**: Jenkins

## 📝 License

Private - Furab Team
