# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy shared library first (needed for the replace directive)
COPY shared/ ./shared/

# Copy service files
COPY services/ride-order-service/ ./services/ride-order-service/

# Download dependencies
WORKDIR /app/services/ride-order-service
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ride-order-service ./cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /ride-order-service .

# Copy migration files
COPY services/ride-order-service/migrations/ ./migrations/

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run binary
ENTRYPOINT ["./ride-order-service"]
