SERVER_DIR=cmd/server
AGENT_DIR=cmd/agent

SERVER_BIN=$(SERVER_DIR)/server
AGENT_BIN=$(AGENT_DIR)/agent

METRICSTEST_BIN=./metricstest

ITERATION ?= 1

MIGRATIONS_DIR=./migrations
GOOSE_BIN=goose
PG_DSN=${DATABASE_DSN}

STATICTEST_BIN=./statictest-darwin-arm64

.PHONY: all build test clean fmt vet lint help

all: build test

build-server:
	go build -o $(SERVER_BIN) $(SERVER_DIR)/*.go

build-agent:
	go build -o $(AGENT_BIN) $(AGENT_DIR)/*.go

build: build-server build-agent

test:
	$(METRICSTEST_BIN) -test.v -test.run=^TestIteration$(ITERATION)$$ \
		-agent-binary-path=$(AGENT_BIN) \
		-binary-path=$(SERVER_BIN) \
		$(if $(SOURCE_PATH),-source-path=$(SOURCE_PATH)) \
		-server-port=${SERVER_PORT} \
		-file-storage-path=${FILE_STORAGE_PATH} \
		-database-dsn=${DATABASE_DSN} \
		-key=${KEY}


clean:
	rm -f $(SERVER_BIN) $(AGENT_BIN)

fmt:
	go fmt ./...

download-statictest:
	@if [ ! -f $(STATICTEST_BIN) ]; then \
		echo "statictest binary not found. Downloading..."; \
		@mkdir -p .tools; \
		curl -sSL https://github.com/Yandex-Practicum/go-autotests/releases/latest/download/statictest-darwin-arm64 -o $(STATICTEST_BIN); \
		chmod +x $(STATICTEST_BIN); \
	else \
		echo "Using local statictest binary..."; \
	fi

vet: download-statictest
	@echo "Running go vet with statictest..."
	go vet -vettool=$(STATICTEST_BIN) ./...

lint:
	golangci-lint run

help:
	@echo "Makefile commands:"
	@echo "  make build           - Build server and agent binaries"
	@echo "  make test            - Run autotests (use ITERATION=N to specify)"
	@echo "  make clean           - Remove binaries"
	@echo "  make fmt             - Format code"
	@echo "  make vet             - Run 'go vet'"
	@echo "  make lint            - Run golangci-lint (optional)"

run-server:
	$(SERVER_BIN) -a=localhost:8080 --audit-file=audit

run-agent:
	$(AGENT_BIN) -a=http://localhost:8080 -r=10 -p=2

migrate-new:
	@echo "Creating new migration: $(name)"
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) create $(name) sql
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) fix

migrate-up:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" up

migrate-down:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" down

migrate-reset:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" reset

migrate-fix:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) fix

migrate-status:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" status

cover:
	./covertest-darwin-arm64 ./...

check-fmt:
	@echo "Checking code formatting with gofmt..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not properly formatted:"; \
		gofmt -l .; \
		exit 1; \
	else \
		echo "All files are properly formatted."; \
	fi

check-imports:
	@echo "Checking imports with goimports..."
	@if [ -n "$$(goimports -l .)" ]; then \
		echo "The following files have import issues:"; \
		goimports -l .; \
		exit 1; \
	else \
		echo "All imports are correct."; \
	fi

check: check-fmt check-imports
	@echo "Code formatting and imports are OK ✅"

fmt-all:
	gofmt -w .
	goimports -w .
	@echo "Code and imports formatted ✅"
