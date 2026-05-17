# Walkthrough: Functional Tests untuk 5 Services

## Ringkasan

Berhasil mengimplementasikan **functional tests** untuk 5 services, mengikuti pola `ride-order-service` sebagai referensi. Semua test menggunakan **real database** (bukan mock), dengan build tag `//go:build functional`.

## Files yang Diubah

### 1. Location Service (Redis) — 6 Test Cases
**File:** [location_functional_test.go](file:///c:/Users/Lenovo/OneDrive/dokumenku/GitHub/furabapps/furab-backend/services/location-service/test/functional/location_functional_test.go)

| Test | Deskripsi |
|------|-----------|
| `TestFunctional_UpdateAndTrackDriver` | Update lokasi → track → verifikasi lat/lng di Redis |
| `TestFunctional_SearchNearbyDrivers` | Register 3 driver → GEOSEARCH 5km & 10km radius |
| `TestFunctional_UpdateDriverStatus` | Available vs Busy filtering di search |
| `TestFunctional_InvalidDriverID` | Validasi driver_id kosong |
| `TestFunctional_UpdateLocationOverwrite` | Overwrite lokasi lama |
| `TestFunctional_InvalidRadius` | Radius 0 dan negatif ditolak |

**Infra:** Real **Redis** (DB 1), stub `DriverServiceClient`

---

### 2. Chat Service (PostgreSQL) — 5 Test Cases
**File:** [chat_functional_test.go](file:///c:/Users/Lenovo/OneDrive/dokumenku/GitHub/furabapps/furab-backend/services/chat-service/test/functional/chat_functional_test.go)

| Test | Deskripsi |
|------|-----------|
| `TestFunctional_SendAndGetHistory` | Kirim pesan → ambil history dari DB |
| `TestFunctional_ChatLifecycle` | User send → Driver reply → Read receipt → Close session |
| `TestFunctional_InvalidRequest` | Field kosong ditolak |
| `TestFunctional_MultipleMessages` | 5 pesan → verifikasi urutan di history |
| `TestFunctional_ReadReceiptValidation` | Status invalid ditolak |

**Infra:** Real **PostgreSQL** (`chat_service` DB), in-test repo, stubs untuk User/Driver/Notification clients
**Schema:** `chat_sessions` + `messages` tables

---

### 3. Notification Service (PostgreSQL) — 6 Test Cases
**File:** [notification_functional_test.go](file:///c:/Users/Lenovo/OneDrive/dokumenku/GitHub/furabapps/furab-backend/services/notification-service/test/functional/notification_functional_test.go)

| Test | Deskripsi |
|------|-----------|
| `TestFunctional_SendPushNotification` | Push notification + verifikasi log di DB |
| `TestFunctional_SendEmailNotification` | Email channel notification |
| `TestFunctional_InvalidEventType` | Unknown event type ditolak |
| `TestFunctional_ValidationErrors` | Missing fields: event_type, receiver_id, etc. |
| `TestFunctional_GenerateTemplate` | Template lookup dari DB |
| `TestFunctional_MultipleNotifications` | 3 notifikasi → count di DB |

**Infra:** Real **PostgreSQL** (`notification_service` DB), seeded templates, stub `EmailClient`
**Schema:** `notification_templates` + `notification_logs` tables

---

### 4. Email Service (PostgreSQL) — 7 Test Cases
**File:** [email_functional_test.go](file:///c:/Users/Lenovo/OneDrive/dokumenku/GitHub/furabapps/furab-backend/services/email-service/test/functional/email_functional_test.go)

| Test | Deskripsi |
|------|-----------|
| `TestFunctional_SendDirectEmail` | Kirim email langsung + verifikasi log di DB |
| `TestFunctional_SendEmailWithTemplate` | Kirim pakai template_id dari DB |
| `TestFunctional_SendEmailWithTemplateNotFound` | Template tidak ada ditolak |
| `TestFunctional_TriggerEventEmail` | Event-driven email (payment.success) |
| `TestFunctional_TriggerInvalidEvent` | Unknown event ditolak |
| `TestFunctional_SendEmailWithResult` | Result wrapper verification |
| `TestFunctional_InvalidRequest` | Missing receiver_email/subject/body |

**Infra:** Real **PostgreSQL** (`email_service` DB), seeded templates, stub `EmailSender`
**Schema:** `email_templates` + `email_logs` tables

---

### 5. Emergency Service (PostgreSQL) — 8 Test Cases
**File:** [emergency_functional_test.go](file:///c:/Users/Lenovo/OneDrive/dokumenku/GitHub/furabapps/furab-backend/services/emergency-service/test/functional/emergency_functional_test.go)

| Test | Deskripsi |
|------|-----------|
| `TestFunctional_TriggerEmergencyFull` | Full flow + verifikasi event di DB + notifikasi |
| `TestFunctional_DriverEmergency` | Emergency dari driver |
| `TestFunctional_LocationFallback` | Location service fail → fallback ke request payload |
| `TestFunctional_InvalidActor` | actor_id kosong / actor_type bukan user/driver |
| `TestFunctional_InvalidActorValidation` | Actor validation gagal |
| `TestFunctional_NotificationFailureDoesNotBlock` | Notif gagal tapi emergency tetap sukses |
| `TestFunctional_WithEmergencyContact` | Dengan emergency contact |
| `TestFunctional_EmergencyWithoutOrder` | Emergency tanpa order_id |

**Infra:** Real **PostgreSQL** (`emergency_service` DB), configurable stubs per-test
**Schema:** `emergency_events` table

---

### go.mod Changes
| Service | Dependency Ditambahkan |
|---------|----------------------|
| chat-service | `github.com/jackc/pgx/v5 v5.6.0` |
| notification-service | `github.com/jackc/pgx/v5 v5.6.0`, `github.com/google/uuid v1.6.0` |
| email-service | `github.com/jackc/pgx/v5 v5.6.0` |
| emergency-service | `github.com/jackc/pgx/v5 v5.6.0` |

## Verification

Semua 5 functional tests **berhasil compile** ✅:
```
go build -tags=functional ./test/functional/...  → OK (semua 5 service)
```

## Cara Menjalankan Functional Tests

### Prerequisites
1. **Docker Desktop** harus running
2. Start infrastructure:
```bash
docker compose -f deploy/docker/docker-compose.yml up -d postgres redis
```
3. Buat database untuk masing-masing service (jika belum ada):
```sql
CREATE DATABASE chat_service;
CREATE DATABASE notification_service;
CREATE DATABASE email_service;
CREATE DATABASE emergency_service;
```

### Run Tests

> [!IMPORTANT]
> Gunakan `GOWORK=off` jika ada go.work file yang bermasalah (merchant-service/rating-service/review-service memiliki merge conflict di go.mod).

```powershell
# Set environment variable untuk bypass go.work
$env:GOWORK="off"

# 1. Location Service (butuh Redis)
cd services/location-service
go test ./test/functional/... -v -tags=functional

# 2. Chat Service (butuh PostgreSQL)
cd services/chat-service
go test ./test/functional/... -v -tags=functional

# 3. Notification Service (butuh PostgreSQL)
cd services/notification-service
go test ./test/functional/... -v -tags=functional

# 4. Email Service (butuh PostgreSQL)
cd services/email-service
go test ./test/functional/... -v -tags=functional

# 5. Emergency Service (butuh PostgreSQL)
cd services/emergency-service
go test ./test/functional/... -v -tags=functional
```

### Environment Variables (Optional)

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `furab` | PostgreSQL user |
| `DB_PASSWORD` | `furab_secret` | PostgreSQL password |
| `DB_NAME` | `{service}_service` | Database name per service |
| `REDIS_HOST` | `localhost` | Redis host (location-service only) |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | _(empty)_ | Redis password |

> [!NOTE]
> Functional tests **tidak mengubah** service code yang existing. Hanya file `*_functional_test.go` dan `go.mod`/`go.sum` yang diubah.

## Pola Arsitektur

Semua functional tests mengikuti pola ride-order-service:

```
TestMain()
├── Connect to real DB (PostgreSQL/Redis)
├── Wait for DB ready (max 30s)
├── setupSchema() → CREATE TABLE IF NOT EXISTS
├── Init in-test repo (implements interface) + stubs
├── Init service layer
├── m.Run() → execute tests
└── teardownSchema() → DROP TABLE
```

Setiap test case:
```
TestFunctional_XxxYyy()
├── cleanupData() → DELETE FROM tables
├── Setup test data (seed if needed)
├── Call service methods (real DB operations)
├── Assert response
└── Verify data persisted in DB via direct SQL query
```
