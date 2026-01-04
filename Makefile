SHELL := /bin/bash

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: install
install: ## Install dependencies
	go mod download

.PHONY: generate
generate: ## Generate Prisma client
	go run github.com/steebchen/prisma-client-go generate

.PHONY: db-push
db-push: ## Push database schema
	go run github.com/steebchen/prisma-client-go db push

.PHONY: test
test: ## Run tests
	go test ./... -v -cover

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test ./... -v -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

.PHONY: lint
lint: ## Run linter
	golangci-lint run --timeout=5m

.PHONY: build
build: generate ## Build the application
	go build -o server ./cmd/server

.PHONY: run
run: ## Run the application
	go run ./cmd/server/main.go

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t dws-ticket-service:latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env dws-ticket-service:latest

.PHONY: clean
clean: ## Clean build artifacts
	rm -f server coverage.out coverage.html
	rm -rf prisma/db

.PHONY: setup
setup: install generate db-push ## Setup project (install + generate + db-push)
