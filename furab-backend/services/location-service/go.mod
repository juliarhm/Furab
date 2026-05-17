module furab-backend/services/location-service

go 1.24.0

require (
	furab-backend/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/redis/go-redis/v9 v9.19.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/mock v0.6.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace furab-backend/shared => ../../shared
