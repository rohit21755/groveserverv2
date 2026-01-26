.PHONY: help build run test clean docker-build docker-up docker-down docker-logs migrate-up migrate-down migrate-create

# Variables
BINARY_NAME=main
DOCKER_COMPOSE=docker-compose
MIGRATE_CMD=migrate -path ./migrations -database

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building application..."
	@go build -o $(BINARY_NAME) ./cmd/api

run: ## Run the application
	@echo "Running application..."
	@go run ./cmd/api

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@$(DOCKER_COMPOSE) build

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@$(DOCKER_COMPOSE) up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@$(DOCKER_COMPOSE) down

docker-logs: ## Show Docker logs
	@$(DOCKER_COMPOSE) logs -f

docker-restart: ## Restart Docker containers
	@echo "Restarting Docker containers..."
	@$(DOCKER_COMPOSE) restart

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@migrate $(MIGRATE_CMD) "$$(grep DATABASE_URL .env | cut -d '=' -f2)" up

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	@migrate $(MIGRATE_CMD) "$$(grep DATABASE_URL .env | cut -d '=' -f2)" down 1

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir migrations -seq $(NAME)

migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@migrate $(MIGRATE_CMD) "$$(grep DATABASE_URL .env | cut -d '=' -f2)" force $(VERSION)

dev: ## Start development environment (docker up + run app)
	@$(DOCKER_COMPOSE) up -d postgres redis
	@echo "Waiting for services to be ready..."
	@sleep 5
	@go run ./cmd/api

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Run linter

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/api/main.go -o docs
	@echo "Swagger docs generated in ./docs"
