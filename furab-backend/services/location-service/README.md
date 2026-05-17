# 🛰️ Location Service

## 📌 Deskripsi
Location Service merupakan microservice yang bertanggung jawab dalam mengelola lokasi driver secara real-time serta status ketersediaan driver untuk mendukung proses matching dan tracking dalam layanan ride.

Service ini digunakan pada dua proses utama:
- **Matching** → menyediakan kandidat driver terdekat
- **Tracking** → menampilkan posisi driver saat perjalanan (PICKING_UP dan ON_THE_WAY)

Selain itu, service ini juga mengelola status driver:
- **available** → dapat menerima order
- **busy** → tidak ikut matching, tetapi tetap bisa di-track

---

# 📥 Input

## 1. Update Lokasi Driver
| Field        | Tipe     |
|--------------|----------|
| driver_id    | string   |
| latitude     | float    |
| longitude    | float    |
| timestamp    | datetime |

---

## 2. Update Status Driver
| Field          | Tipe   |
|----------------|--------|
| driver_id      | string |
| driver_status  | string |

Keterangan:
- `available` → ikut matching
- `busy` → tidak ikut matching

---

## 3. Pencarian Driver
| Field             | Tipe  |
|------------------|-------|
| latitude_origin  | float |
| longitude_origin | float |
| radius           | float |

---

## 4. Request Tracking Lokasi
| Field     | Tipe   |
|-----------|--------|
| driver_id | string |
| order_id  | string |

---

📌 **Catatan:**  
Pada food delivery, pencarian driver menggunakan lokasi merchant sebagai titik origin.

---

# 📤 Output

## 1. Hasil Pencarian Driver
| Field         | Tipe   |
|---------------|--------|
| driver_id     | string |
| latitude      | float  |
| longitude     | float  |
| distance      | float  |
| driver_status | string |

---

## 2. Data Tracking Lokasi
| Field     | Tipe     |
|-----------|----------|
| driver_id | string   |
| latitude  | float    |
| longitude | float    |
| timestamp | datetime |

---

## 3. Response Sistem
| Field   | Tipe   |
|---------|--------|
| status  | string |
| message | string |

---

# 🗄️ Data

## 1. driver_location
Menyimpan lokasi driver secara real-time.

| Field       | Tipe     |
|-------------|----------|
| driver_id   | string   |
| latitude    | float    |
| longitude   | float    |
| timestamp   | datetime |
| expiry_time | TTL      |

---

## 2. driver_status
Menyimpan status driver.

| Field         | Tipe   |
|---------------|--------|
| driver_id     | string |
| driver_status | string |

---

# ⚙️ Mekanisme Penyimpanan

- Menggunakan **Redis (in-memory)**
- Menggunakan **Geo-index** untuk pencarian radius
- Data diperbarui secara berkala
- Data lama akan expired (TTL)
- Driver dengan status **busy** tidak masuk hasil pencarian

---

# 🧠 State Management

Location Service bersifat **stateless**, artinya:
- Tidak menyimpan session user
- Setiap request diproses secara independen
- Data disimpan di Redis

---

# 🔗 Interaksi dengan Microservices

- **Driver Service**  
  Validasi driver yang mengirim lokasi

- **Matching Service**  
  Mengambil kandidat driver dan mengubah status driver menjadi busy saat accept order

- **Merchant Service**  
  Menyediakan lokasi merchant sebagai titik origin (food delivery)

- **Ride Order Service**  
  - Menggunakan data lokasi untuk tracking  
  - Mengubah status driver menjadi available saat order selesai  

---

# 📊 Asumsi Load

| Proses             | Request Rate        |
|--------------------|-------------------|
| Update lokasi      | ±1000 req/detik   |
| Pencarian driver   | ±500 req/detik    |
| Tracking lokasi    | ±500–800 req/detik|

---

# 🧪 Testing Strategy

## 🔹 Unit Test
- Tidak menggunakan Redis
- Menggunakan mock repository
- Fokus pada:
  - Validasi input
  - Logic service

---

## 🔹 Functional Test
- Menggunakan Redis / in-memory DB
- Menguji flow:
  - Update lokasi
  - Matching driver
  - Tracking

---

## 🔹 Integration Test
- Tanpa mock
- Menggunakan Redis asli / TestContainer
- Menguji:
  - Penyimpanan data
  - Geo query (radius search)

---

# ⚙️ Pipeline Testing

1. Checkout Repository  
2. Unit Test → `go test ./...`  
3. Lint/Vet → `go vet ./...`  
4. Build Image → `docker build`  
5. Functional Test  
6. Push Image  
7. Deploy ke Kubernetes  
8. Verify Service  

---

# 📌 Catatan

- Unit test akan tetap berjalan meskipun implementasi belum lengkap
- Functional test dapat gagal karena sistem belum fully implemented
- Fokus utama adalah struktur dan pendekatan testing
