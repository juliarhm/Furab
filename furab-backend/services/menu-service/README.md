# Menu Service -join ' ')

Restaurant menu management service

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
menu-service/
+-- cmd/main.go
+-- internal/
    +-- handler/menu_handler.go
    +-- service/menu_service.go
    +-- repository/menu_repository.go
    +-- model/menu.go
+-- test/
    +-- unit/menu_service_test.go
    +-- unit/mock/
    +-- functional/menu_functional_test.go
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
export DB_NAME=menu-service

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
--- PASS: TestNewMenuService_Creation
PASS
```

**Test GAGAL jika output:**
```
--- FAIL: TestNewMenuService_Creation
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
docker build -t furab/menu-service:latest -f services/menu-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/menu-service:latest
```
