.PHONY: all build run stop clean dev test swagger swagger-init

# Default target
all: build

# Install swagger tool
swagger-init:
	go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger docs
swagger:
	cd api && swag init -g cmd/api/main.go -o docs

# Build all services
build:
	docker-compose build

# Start all services
run:
	docker-compose up -d

# Stop all services
stop:
	docker-compose down

# Clean up
clean:
	docker-compose down -v
	docker system prune -f

# Development mode - start infrastructure only
dev-infra:
	docker-compose up -d postgres redis nats

# Run gateway locally (requires dev-infra)
dev-gateway:
	cd gateway && go run cmd/gateway/main.go

# Run API locally (requires dev-infra)
dev-api:
	cd api && go run cmd/api/main.go

# Run web locally (requires dev-api)
dev-web:
	cd web && npm install && npm run dev

# Run tests
test:
	cd gateway && go test ./...
	cd api && go test ./...

# View logs
logs:
	docker-compose logs -f

# Database migration
migrate:
	docker-compose exec postgres psql -U openfms -d openfms -f /docker-entrypoint-initdb.d/01_init.sql

# Create admin user
create-admin:
	docker-compose exec postgres psql -U openfms -d openfms -c "INSERT INTO users (username, password, email, role, status) VALUES ('admin', '\$2a\$10\$N9qo8uLOickgx2ZMRZoMy.MqrqQzBZN0UfGNEsKYGsGvJz1eKx3.K', 'admin@openfms.local', 'admin', 1) ON CONFLICT DO NOTHING;"

# Status
status:
	docker-compose ps
