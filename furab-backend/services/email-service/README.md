# Email Service

## Deskripsi

Email Service merupakan microservice yang bertanggung jawab untuk mengirimkan email kepada user, driver, atau merchant sebagai bentuk komunikasi formal dalam sistem.

Email digunakan untuk:
- Bukti transaksi (receipt/invoice)
- Informasi penting non-realtime (refund, payment gagal)
- Keamanan akun (verifikasi, reset password, login mencurigakan)

Service ini bersifat event-driven, dan umumnya dipicu melalui Notification Service (sebagai channel email), meskipun beberapa service seperti Payment Service atau Auth Service dapat memicu langsung pada kondisi tertentu.

Email Service juga dapat menggunakan template untuk membentuk isi email berdasarkan `event_type`.

## Input

### 1. Kirim Email

| Field | Tipe | Deskripsi |
|------|------|----------|
| receiver_email | string | Email penerima |
| subject | string | Judul email |
| body | text | Isi email |

### 2. Trigger Event

| Field | Tipe | Deskripsi |
|------|------|----------|
| event_type | string | Jenis event pemicu email |
| reference_id | string | ID referensi (order_id / payment_id / dan lain-lain) |
| receiver_email | string | Email penerima |
| timestamp | datetime | Waktu event terjadi |
| metadata | JSON | Data tambahan untuk template |

## Output

### 1. Email Terkirim

| Field | Tipe | Deskripsi |
|------|------|----------|
| email_id | string | ID email unik |
| receiver_email | string | Email penerima |
| subject | string | Judul email |
| status | string | sent / failed |
| timestamp | datetime | Waktu pengiriman |

### 2. Response Sistem

| Field | Tipe | Deskripsi |
|------|------|----------|
| status | string | success / failed |
| message | string | Informasi hasil pengiriman |

## Data

### Struktur Data: `email_log`

| Field | Tipe |
|------|------|
| email_id | string |
| receiver_email | string |
| subject | string |
| status | string |
| timestamp | datetime |
| receiver_id | string |
| reference_id | string |

## State

Email Service bersifat stateless, karena tidak menyimpan session pengguna.

Namun, riwayat email disimpan dalam database sebagai `email_log` untuk:
- Monitoring
- Audit
- Debugging
- Tracking pengiriman

## Interaksi dengan Microservice Lain

Email Service berinteraksi dengan beberapa service berikut:
- Notification Service: trigger email sebagai channel notifikasi
- Payment Service: invoice dan bukti pembayaran
- Auth Service: email verifikasi / reset password
- Ride Order Service: ringkasan perjalanan
- Food Order Service: ringkasan pesanan makanan

## Asumsi Load Sistem

- Pengiriman email: sekitar 100-300 request/detik
- Trigger event dari service lain: sekitar 100-200 request/detik

Email Service didesain untuk menangani beban tinggi pada jam sibuk secara asynchronous untuk menghindari blocking pada service utama.

## Alur Singkat

1. Service lain memicu event (misalnya Payment Success)
2. Notification Service menerima event
3. Jika channel = email, event diteruskan ke Email Service
4. Email Service membentuk email (template atau direct input)
5. Email dikirim ke user
6. Log disimpan ke `email_log`

## Tech Stack

| Komponen | Teknologi |
|----------|-----------|
| Language | Go 1.22+ |
| HTTP Router | chi |
| Database | PostgreSQL |
| Testing | gomock, go test |

## Menjalankan Tests

### Unit Tests (Tanpa Database)
```bash
go test ./test/unit/... -v
```

### Functional Tests (Dengan Database)
```bash
go test ./test/functional/... -v -tags=functional
```
