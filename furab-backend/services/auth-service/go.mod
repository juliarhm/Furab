module furab-backend/services/auth-service

go 1.24.0

require (
	furab-backend/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
	go.uber.org/mock v0.6.0
)

replace furab-backend/shared => ../../shared
