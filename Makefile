GO ?= go
ENV_FILE ?= .env
BACKEND_DIR := backend
API_PACKAGE := ./cmd/api
API_BINARY := bin/daftar-api
LOAD_ENV = set -a; . ./$(ENV_FILE); set +a;
COMPOSE = DAFTAR_ENV_FILE=$(ENV_FILE) docker compose --env-file $(ENV_FILE)

.PHONY: help check-env run seed seed-native frontend build frontend-build test test-integration test-race fmt vet tidy check clean docker-build docker-up docker-down docker-logs up down logs

help:
	@echo "Daftar development commands"
	@echo "  make run        Run the API"
	@echo "  make seed       Seed reviewer data through Docker and exit"
	@echo "  make seed-native  Seed reviewer data using native dependencies"
	@echo "  make frontend   Run the Next.js frontend"
	@echo "  make build      Build bin/daftar-api"
	@echo "  make test       Run all tests"
	@echo "  make test-integration  Run MongoDB tests with Testcontainers"
	@echo "  make test-race  Run tests with the race detector"
	@echo "  make fmt        Format Go source"
	@echo "  make vet        Run go vet"
	@echo "  make tidy       Tidy module dependencies"
	@echo "  make check      Format, vet, and test"
	@echo "  make docker-build  Build the API container image"
	@echo "  make docker-up     Start the API and MongoDB"
	@echo "  make docker-down   Stop the Docker stack"
	@echo "  make docker-logs   Follow Docker stack logs"
	@echo "  make up/down/logs  Short aliases for the Docker commands"
	@echo ""
	@echo "All runtime commands load ENV_FILE (default: .env)."

check-env:
	@test -f "$(ENV_FILE)" || { echo "missing $(ENV_FILE); copy .env.example to $(ENV_FILE)"; exit 1; }

run: check-env
	$(LOAD_ENV) \
	export DAFTAR_MONGODB_URI="$${DAFTAR_MONGODB_URI_NATIVE:-$${DAFTAR_MONGODB_URI}}"; \
	cd $(BACKEND_DIR) && $(GO) run $(API_PACKAGE)

seed: check-env
	$(COMPOSE) run --build --rm api -seed

seed-native: check-env
	$(LOAD_ENV) \
	export DAFTAR_MONGODB_URI="$${DAFTAR_MONGODB_URI_NATIVE:-$${DAFTAR_MONGODB_URI}}"; \
	cd $(BACKEND_DIR) && $(GO) run $(API_PACKAGE) -seed

frontend: check-env
	$(LOAD_ENV) \
	export DAFTAR_API_INTERNAL_URL="$${DAFTAR_API_INTERNAL_URL_NATIVE:-$${DAFTAR_API_INTERNAL_URL}}"; \
	cd frontend && npm run dev

build:
	@mkdir -p $(BACKEND_DIR)/bin
	cd $(BACKEND_DIR) && $(GO) build -trimpath -o $(API_BINARY) $(API_PACKAGE)

frontend-build:
	cd frontend && npm run build

test:
	cd $(BACKEND_DIR) && $(GO) test ./...

test-integration:
	cd $(BACKEND_DIR) && $(GO) test -tags=integration ./internal/mercury ./internal/transport/web -count=1

test-race:
	cd $(BACKEND_DIR) && $(GO) test -race ./...

fmt:
	cd $(BACKEND_DIR) && $(GO) fmt ./...

vet:
	cd $(BACKEND_DIR) && $(GO) vet ./...

tidy:
	cd $(BACKEND_DIR) && $(GO) mod tidy

check: fmt vet test

clean:
	@rm -f $(BACKEND_DIR)/$(API_BINARY)

docker-build: check-env
	$(COMPOSE) build

docker-up: check-env
	$(COMPOSE) up --build -d

docker-down: check-env
	$(COMPOSE) down

docker-logs: check-env
	$(COMPOSE) logs -f

up: docker-up

down: docker-down

logs: docker-logs
