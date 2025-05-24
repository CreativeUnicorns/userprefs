# Makefile for Supabase example

.PHONY: help setup run run-local run-api clean test docker-up docker-down docker-logs

# Default target
help:
	@echo "Available commands:"
	@echo "  setup       - Install dependencies"
	@echo "  setup-database - Initialize database schema (requires .env)"
	@echo "  run         - Run demo with Supabase cloud (requires .env)"
	@echo "  run-local   - Run demo with local Docker environment"
	@echo "  run-api     - Start HTTP API server"
	@echo "  run-api-local - Start HTTP API server with local Docker"
	@echo "  run-client  - Run API client example (requires API server)"
	@echo "  test        - Run tests"
	@echo "  integration-test - Run comprehensive integration tests"
	@echo "  docker-up   - Start local PostgreSQL and Redis"
	@echo "  docker-down - Stop local services"
	@echo "  docker-logs - View Docker logs"
	@echo "  clean       - Clean up generated files"

# Install dependencies
setup:
	go mod tidy
	@echo "Dependencies installed. Next steps:"
	@echo "1. Copy .env.example to .env and add your Supabase credentials"
	@echo "2. Run 'make setup-database' to initialize the schema"
	@echo "3. Run 'make run' to start the demo"

# Initialize database schema (run this after configuring .env)
setup-database:
	@echo "Setting up database schema..."
	go run setup_database.go

# Run with Supabase cloud
run:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	go run main.go

# Run with local Docker environment
run-local: docker-up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Running with local environment..."
	@cp .env.local .env
	go run main.go

# Run API server with Supabase cloud
run-api:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	go run cmd/api/api_server.go

# Run API server with local Docker environment
run-api-local: docker-up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Starting API server with local environment..."
	@cp .env.local .env
	go run cmd/api/api_server.go

# Start Docker services
docker-up:
	docker-compose up -d postgres redis
	@echo "PostgreSQL and Redis started. Use 'make docker-logs' to view logs."

# Start Docker services with management tools
docker-up-with-tools:
	docker-compose --profile tools up -d
	@echo "All services started including pgAdmin (http://localhost:8080) and Redis Commander (http://localhost:8081)"

# Stop Docker services
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# Clean up
clean:
	docker-compose down -v
	rm -f .env
	go clean

# Run API client example
run-client:
	@echo "Running API client example..."
	@echo "Make sure the API server is running (make run-api or make run-api-local)"
	go run cmd/client/client_example.go

# Test the example
test:
	go test -v ./...

# Run comprehensive integration tests
integration-test:
	./integration_test.sh

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Show Docker service status
status:
	docker-compose ps

# Reset demo data in local database
reset-demo-data: docker-up
	@echo "Resetting demo data..."
	@sleep 2
	docker-compose exec postgres psql -U postgres -d postgres -c "SELECT userprefs.reset_demo_data();"

# Connect to local PostgreSQL
psql:
	docker-compose exec postgres psql -U postgres -d postgres

# Connect to local Redis
redis-cli:
	docker-compose exec redis redis-cli

# Show database schema
show-schema:
	docker-compose exec postgres psql -U postgres -d postgres -c "\d+ userprefs.*"

# Backup local database
backup:
	@mkdir -p backups
	docker-compose exec postgres pg_dump -U postgres postgres > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Database backed up to backups/"

# Load sample data
load-sample-data:
	@echo "Loading sample data..."
	@cp .env.local .env
	go run main.go > /dev/null 2>&1 &
	@sleep 3
	@pkill -f "go run main.go" || true
	@echo "Sample data loaded"
