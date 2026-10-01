# Run `make help` to list the targets.

SERVICES := api-gateway auth-service product-service inventory-service order-service payment-service cart-service review-service
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:15432/ecommerce_test?sslmode=disable
GOBIN := $(shell go env GOPATH)/bin

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.env:
	cp .env.example .env

## ---- Development -----------------------------------------------------------

.PHONY: up
up: .env ## Build and start the whole stack (waits until healthy)
	docker compose up -d --build --wait

.PHONY: down
down: ## Stop the stack and delete its volumes
	docker compose down -v

.PHONY: logs
logs: ## Follow the logs of all services
	docker compose logs -f --tail=100

.PHONY: smoke
smoke: ## Run the end-to-end smoke test against the running stack
	./scripts/smoke-test.sh

## ---- Code quality -----------------------------------------------------------

.PHONY: build
build: ## Compile every service into ./bin
	@for s in $(SERVICES); do CGO_ENABLED=0 go build -o bin/$$s ./$$s || exit 1; done

.PHONY: test
test: ## Run unit tests (integration tests are skipped)
	go test -race ./...

.PHONY: test-integration
test-integration: ## Run all tests, including integration tests against Postgres (make up first)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -count=1 -coverprofile=coverage.out ./...

.PHONY: lint
lint: ## Run golangci-lint and buf lint
	golangci-lint run ./...
	buf lint

.PHONY: proto
proto: ## Regenerate gRPC code from api/proto
	buf generate

.PHONY: tools
tools: ## Install the code generators and linters
	go install github.com/bufbuild/buf/cmd/buf@v1.73.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
