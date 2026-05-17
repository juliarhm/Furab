# Pricing Service

Pricing Service adalah microservice yang bertanggung jawab untuk melakukan kalkulasi total biaya pesanan secara akurat. Perhitungan mencakup komponen biaya seperti harga item, biaya layanan (*service fee*), dan biaya pengiriman (*delivery fee*).

## 1. Deskripsi

Pricing Service bersifat stateless dan hanya melakukan kalkulasi real-time berdasarkan data pesanan dan aturan harga yang tersedia. Hasilnya digunakan sebagai dasar bagi Promo Service dan Payment Service untuk menghitung total harga akhir.

## 2. Spesifikasi Input & Output

### Input (Request)

| Fitur | Field |
| :--- | :--- |
| **Hitung Harga** | `order_id` |

### Output (Response)

| Fitur | Field |
| :--- | :--- |
| **Total Harga** | `total_amount` |
| **Rincian Harga** | `item_price`, `delivery_fee`, `service_fee` |

## 3. Struktur Data (Skema Tabel)

### Tabel `pricing_rules`

Menyimpan aturan dan parameter perhitungan biaya.

- `rule_id`: Primary Key.
- `type`: Jenis biaya (`delivery`, `service`, `tax`).
- `value`: Nilai atau besaran biaya (flat atau persentase).
- `description`: Penjelasan mengenai aturan biaya tersebut.

## 4. State & Logika Bisnis

- **Stateless Logic:** Pricing Service tidak menyimpan status transaksi.
- **Fast Response:** Dirancang untuk merespons dengan cepat agar alur transaksi tetap lancar.
- **Perhitungan:** total harga dihitung sebagai `item_price + delivery_fee + service_fee`.

## 5. Interaksi dengan Microservice Lain

- **Order Service:** mengambil data detail item pesanan.
- **Location Service:** mendapatkan informasi jarak tempuh untuk menghitung biaya pengiriman.
- **Promo Service:** menggunakan `total_amount` sebagai dasar perhitungan diskon.
- **Payment Service:** menggunakan total harga akhir untuk proses pembayaran.

## 6. API Endpoint

### GET /api/v1/prices/{orderID}

Request:

```bash
curl http://localhost:8080/api/v1/prices/order-123
```

Response:

```json
{
  "success": true,
  "data": {
    "order_id": "order-123",
    "total_amount": 81650,
    "item_price": 53000,
    "delivery_fee": 26000,
    "service_fee": 2650
  }
}
```

## Struktur Folder

```
pricing-service/
+-- cmd/main.go
+-- internal/
    +-- client/
        +-- location_client.go
        +-- order_client.go
    +-- handler/price_handler.go
    +-- model/price.go
    +-- repository/price_repository.go
    +-- service/price_service.go
+-- test/
    +-- unit/price_service_test.go
    +-- unit/mock/
    +-- functional/price_functional_test.go
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
export DB_NAME=pricing-service

# Jalankan service
cd services/pricing-service
go run cmd/main.go
```

## Menjalankan Tests

### Unit Tests
```bash
go test ./test/unit/... -v
```

### Functional Tests (Dengan Database)
```bash
go test ./test/functional/... -v -tags=functional
```

## Docker

```bash
# Build (dari root project)
docker build -t furab/pricing-service:latest -f services/pricing-service/Dockerfile .

# Run
docker run -p 8080:8080 furab/pricing-service:latest
```
