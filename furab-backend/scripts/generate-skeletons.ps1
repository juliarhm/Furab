# Furab Backend - Skeleton Service Generator
# Generates all 22 skeleton microservices with consistent structure

$basePath = "d:\Pekerjaan\furabapps\furab-backend\services"

# Service definitions: name, model, operations
$services = @(
    @{Name="auth-service"; Model="Auth"; Entity="User,Token,Session"; Ops="Login,Register,RefreshToken,Logout"; Desc="Authentication and authorization service"},
    @{Name="otp-service"; Model="OTP"; Entity="OTPRequest,OTPVerification"; Ops="SendOTP,VerifyOTP,ResendOTP"; Desc="OTP generation and verification service"},
    @{Name="user-service"; Model="User"; Entity="UserProfile,UserAddress"; Ops="GetProfile,UpdateProfile,AddAddress,DeleteAddress"; Desc="User profile management service"},
    @{Name="driver-service"; Model="Driver"; Entity="DriverProfile,DriverLocation"; Ops="GetDriver,UpdateLocation,SetAvailability,GetNearbyDrivers"; Desc="Driver profile and location management service"},
    @{Name="food-order-service"; Model="FoodOrder"; Entity="FoodOrder,OrderItem"; Ops="CreateOrder,ConfirmOrder,PrepareOrder,CompleteOrder,CancelOrder"; Desc="Food order management service"},
    @{Name="cart-service"; Model="Cart"; Entity="Cart,CartItem"; Ops="AddItem,RemoveItem,UpdateQuantity,GetCart,ClearCart"; Desc="Shopping cart management service"},
    @{Name="matching-service"; Model="Match"; Entity="MatchRequest,MatchResult"; Ops="FindDriver,AcceptMatch,RejectMatch,GetMatchStatus"; Desc="Driver-order matching service"},
    @{Name="payment-service"; Model="Payment"; Entity="Payment,PaymentMethod"; Ops="Authorize,Capture,Refund,GetPayment"; Desc="Payment processing service"},
    @{Name="wallet-service"; Model="Wallet"; Entity="Wallet,Transaction"; Ops="GetBalance,TopUp,Debit,Transfer,GetHistory"; Desc="Digital wallet management service"},
    @{Name="settlement-service"; Model="Settlement"; Entity="Settlement,SettlementItem"; Ops="CreateSettlement,ProcessSettlement,GetSettlement"; Desc="Driver settlement and payout service"},
    @{Name="pricing-service"; Model="Price"; Entity="PriceEstimate,PriceRule,SurgeZone"; Ops="EstimatePrice,GetSurgeMultiplier,UpdatePriceRule"; Desc="Dynamic pricing and surge pricing service"},
    @{Name="promo-service"; Model="Promo"; Entity="Promo,PromoUsage"; Ops="ValidatePromo,ApplyPromo,CreatePromo,GetPromos"; Desc="Promotions and discount management service"},
    @{Name="location-service"; Model="Location"; Entity="Location,GeoFence"; Ops="UpdateLocation,GetNearby,TrackDriver,GetGeoFence"; Desc="Real-time location tracking service"},
    @{Name="chat-service"; Model="Chat"; Entity="Conversation,Message"; Ops="SendMessage,GetMessages,GetConversation,MarkAsRead"; Desc="In-app chat and messaging service"},
    @{Name="notification-service"; Model="Notification"; Entity="Notification,NotifTemplate"; Ops="Send,GetAll,MarkAsRead,GetUnreadCount"; Desc="Push notification service"},
    @{Name="email-service"; Model="Email"; Entity="EmailRequest,EmailTemplate"; Ops="SendEmail,SendBulk,GetStatus"; Desc="Email delivery service"},
    @{Name="emergency-service"; Model="Emergency"; Entity="EmergencyRequest,SOSAlert"; Ops="TriggerSOS,GetEmergencyContacts,UpdateContacts"; Desc="SOS and emergency alert service"},
    @{Name="merchant-service"; Model="Merchant"; Entity="Merchant,MerchantProfile"; Ops="Register,GetMerchant,UpdateProfile,SetOperatingHours"; Desc="Merchant registration and management service"},
    @{Name="menu-service"; Model="Menu"; Entity="Menu,MenuItem,Category"; Ops="GetMenu,AddItem,UpdateItem,DeleteItem,GetCategories"; Desc="Restaurant menu management service"},
    @{Name="rating-service"; Model="Rating"; Entity="Rating,RatingStats"; Ops="SubmitRating,GetAverage,GetRatings,GetDriverRating"; Desc="Star rating service"},
    @{Name="review-service"; Model="Review"; Entity="Review"; Ops="SubmitReview,GetReviews,FlagReview,GetReviewStats"; Desc="Text review management service"},
    @{Name="audit-log-service"; Model="AuditLog"; Entity="AuditLog"; Ops="LogAction,GetLogs,SearchLogs,GetLogsByUser"; Desc="Audit trail and logging service"}
)

foreach ($svc in $services) {
    $svcName = $svc.Name
    $svcPath = "$basePath\$svcName"
    $modelName = $svc.Model
    $modelLower = $modelName.ToLower()
    $desc = $svc.Desc
    $entities = $svc.Entity -split ","
    $ops = $svc.Ops -split ","
    $moduleName = "furab-backend/services/$svcName"

    Write-Host "Creating $svcName..."

    # --- go.mod ---
    $goMod = @"
module $moduleName

go 1.22

require (
	furab-backend/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	go.uber.org/mock v0.4.0
)

replace furab-backend/shared => ../../shared
"@
    New-Item -Path "$svcPath" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\go.mod" -Value $goMod -Encoding UTF8

    # --- model ---
    $entityFields = ""
    foreach ($e in $entities) {
        $e = $e.Trim()
        $entityFields += @"

// $e represents the $e model in $svcName.
type $e struct {
	ID        string    ``json:"id"``
	CreatedAt time.Time ``json:"created_at"``
	UpdatedAt time.Time ``json:"updated_at"``
	// TODO: Add $e-specific fields
}

"@
    }

    $modelContent = @"
// Package model defines the domain models for $svcName.
package model

import "time"
$entityFields
"@
    New-Item -Path "$svcPath\internal\model" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\internal\model\${modelLower}.go" -Value $modelContent -Encoding UTF8

    # --- repository ---
    $repoInterface = ""
    foreach ($op in $ops) {
        $op = $op.Trim()
        $repoInterface += "`n`t// $op performs the $op operation.`n`t$op(ctx context.Context) error`n"
    }

    $repoContent = @"
// Package repository provides data access layer for $svcName.
package repository

import "context"

// ${modelName}Repository defines the interface for $svcName data access.
type ${modelName}Repository interface {
$repoInterface}

// postgres${modelName}Repository implements ${modelName}Repository using PostgreSQL.
type postgres${modelName}Repository struct {
	// TODO: add *sql.DB field
}

// NewPostgres${modelName}Repository creates a new PostgreSQL-based repository.
func NewPostgres${modelName}Repository() ${modelName}Repository {
	return &postgres${modelName}Repository{}
}
"@
    New-Item -Path "$svcPath\internal\repository" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\internal\repository\${modelLower}_repository.go" -Value $repoContent -Encoding UTF8

    # --- service ---
    $svcInterface = ""
    foreach ($op in $ops) {
        $op = $op.Trim()
        $svcInterface += "`n`t// $op implements the business logic for $op.`n`t$op(ctx context.Context) error`n"
    }

    $svcContent = @"
// Package service implements the business logic for $svcName.
package service

import "context"

// ${modelName}Service defines the interface for $svcName business logic.
type ${modelName}Service interface {
$svcInterface}

// ${modelLower}ServiceImpl is the concrete implementation of ${modelName}Service.
type ${modelLower}ServiceImpl struct {
	// TODO: add repository and event publisher dependencies
}

// New${modelName}Service creates a new ${modelName}Service.
func New${modelName}Service() ${modelName}Service {
	return &${modelLower}ServiceImpl{}
}
"@
    New-Item -Path "$svcPath\internal\service" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\internal\service\${modelLower}_service.go" -Value $svcContent -Encoding UTF8

    # --- handler ---
    $handlerContent = @"
// Package handler provides HTTP handlers for $svcName.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// ${modelName}Handler handles HTTP requests for $svcName.
type ${modelName}Handler struct {
	// TODO: add service dependency
}

// New${modelName}Handler creates a new ${modelName}Handler.
func New${modelName}Handler() *${modelName}Handler {
	return &${modelName}Handler{}
}

// RegisterRoutes registers all $svcName routes.
func (h *${modelName}Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/${modelLower}s", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "$svcName",
			})
		})
		// TODO: Register endpoint routes
	})
}
"@
    New-Item -Path "$svcPath\internal\handler" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\internal\handler\${modelLower}_handler.go" -Value $handlerContent -Encoding UTF8

    # --- cmd/main.go ---
    $mainContent = @"
// Package main is the entry point for $svcName.
package main

import (
	"log"
	"net/http"
	"time"

	"furab-backend/services/$svcName/internal/handler"
	"furab-backend/shared/config"
	sharedlogger "furab-backend/shared/logger"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load("$svcName")
	logger := sharedlogger.New(cfg.ServiceName, cfg.Environment)

	logger.Info("starting $svcName", "port", cfg.ServerPort)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Register routes
	h := handler.New${modelName}Handler()
	h.RegisterRoutes(r)

	// Start server
	logger.Info("server listening", "address", cfg.ServerAddr())
	if err := http.ListenAndServe(cfg.ServerAddr(), r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
"@
    New-Item -Path "$svcPath\cmd" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\cmd\main.go" -Value $mainContent -Encoding UTF8

    # --- mock ---
    $mockContent = @"
// Package mock provides mock implementations for $svcName testing.
package mock

// TODO: Generate mocks using mockgen:
//   mockgen -source=../../internal/repository/${modelLower}_repository.go -destination=mock_${modelLower}_repository.go -package=mock

// Mock${modelName}Repository is a mock implementation of repository.${modelName}Repository.
type Mock${modelName}Repository struct {
	// TODO: implement with gomock
}
"@
    New-Item -Path "$svcPath\test\unit\mock" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\test\unit\mock\mock_${modelLower}_repository.go" -Value $mockContent -Encoding UTF8

    # --- unit test ---
    $unitTestContent = @"
// Package unit contains unit tests for $svcName.
// Unit tests do NOT access any database or external service.
package unit

import (
	"testing"
)

// TestNew${modelName}Service_Creation tests that the service can be created.
func TestNew${modelName}Service_Creation(t *testing.T) {
	// TODO: Initialize service with mock dependencies
	// svc := service.New${modelName}Service()
	// if svc == nil {
	//     t.Fatal("expected non-nil service")
	// }
	t.Skip("TODO: Implement with mocked dependencies")
}

// Test${modelName}_BasicOperation tests a basic operation.
func Test${modelName}_BasicOperation(t *testing.T) {
	// TODO: Test basic CRUD operation with mocked repository
	t.Skip("TODO: Implement test")
}

// Test${modelName}_ValidationError tests input validation.
func Test${modelName}_ValidationError(t *testing.T) {
	// TODO: Test validation with invalid input
	t.Skip("TODO: Implement test")
}
"@
    Set-Content -Path "$svcPath\test\unit\${modelLower}_service_test.go" -Value $unitTestContent -Encoding UTF8

    # --- functional test ---
    $functionalTestContent = @"
//go:build functional
// +build functional

// Package functional contains functional tests for $svcName.
// These tests access a real database.
// Run with: go test ./test/functional/... -v -tags=functional
package functional

import (
	"testing"
)

// TestFunctional_${modelName}_CreateAndGet tests basic CRUD with real database.
func TestFunctional_${modelName}_CreateAndGet(t *testing.T) {
	// TODO: Setup test database connection
	// TODO: Create entity and verify retrieval
	t.Skip("TODO: Implement with real database")
}

// TestFunctional_${modelName}_FullFlow tests the complete lifecycle.
func TestFunctional_${modelName}_FullFlow(t *testing.T) {
	// TODO: Test full business flow with real database
	t.Skip("TODO: Implement with real database")
}
"@
    New-Item -Path "$svcPath\test\functional" -ItemType Directory -Force | Out-Null
    Set-Content -Path "$svcPath\test\functional\${modelLower}_functional_test.go" -Value $functionalTestContent -Encoding UTF8

    # --- Dockerfile ---
    $dockerContent = @"
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY shared/ ./shared/
COPY services/$svcName/ ./services/$svcName/
WORKDIR /app/services/$svcName
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /$svcName ./cmd/main.go

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /$svcName .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./$svcName"]
"@
    Set-Content -Path "$svcPath\Dockerfile" -Value $dockerContent -Encoding UTF8

    # --- README.md ---
    $readmeContent = @"
# $($svcName.Split('-') | ForEach-Object { (Get-Culture).TextInfo.ToTitleCase($_) }) -join ' ')

$desc

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

``````
$svcName/
+-- cmd/main.go
+-- internal/
    +-- handler/${modelLower}_handler.go
    +-- service/${modelLower}_service.go
    +-- repository/${modelLower}_repository.go
    +-- model/${modelLower}.go
+-- test/
    +-- unit/${modelLower}_service_test.go
    +-- unit/mock/
    +-- functional/${modelLower}_functional_test.go
+-- go.mod
+-- Dockerfile
+-- README.md
``````

## Cara Menjalankan

``````bash
# Set environment variables
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=furab
export DB_PASSWORD=furab_secret
export DB_NAME=$svcName

# Jalankan service
go run cmd/main.go
``````

## Menjalankan Tests

### Unit Tests (Tanpa Database)
``````bash
go test ./test/unit/... -v
``````

**Test BERHASIL jika output:**
``````
--- PASS: TestNew${modelName}Service_Creation
PASS
``````

**Test GAGAL jika output:**
``````
--- FAIL: TestNew${modelName}Service_Creation
FAIL
``````

### Functional Tests (Dengan Database)
``````bash
# Pastikan PostgreSQL berjalan
go test ./test/functional/... -v -tags=functional
``````

## Docker

``````bash
# Build (dari root project)
docker build -t furab/$svcName`:latest -f services/$svcName/Dockerfile .

# Run
docker run -p 8080:8080 furab/$svcName`:latest
``````
"@
    Set-Content -Path "$svcPath\README.md" -Value $readmeContent -Encoding UTF8
}

Write-Host ""
Write-Host "============================================="
Write-Host "All 22 skeleton services created successfully!"
Write-Host "============================================="
