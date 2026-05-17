# API Gateway

Reverse proxy yang meneruskan request ke microservice yang sesuai.

## Cara Menjalankan

```bash
go run cmd/main.go
```

## Routing

| Path | Target Service |
|------|---------------|
| `/api/v1/rides/*` | ride-order-service |
| `/api/v1/foods/*` | food-order-service |
| `/api/v1/payments/*` | payment-service |
| `/api/v1/users/*` | user-service |
| ... | ... |
