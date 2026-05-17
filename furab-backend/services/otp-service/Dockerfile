# Build stage
FROM golang:alpine AS builder
WORKDIR /app
COPY shared/ ./shared/
COPY services/otp-service/ ./services/otp-service/
WORKDIR /app/services/otp-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /otp-service ./cmd/main.go

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /otp-service .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./otp-service"]
