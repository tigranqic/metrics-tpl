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

GODOC_PORT ?= 8089
GODOC_TMP ?= /tmp/godoc
MODULE_NAME := metrics-tpl

LINTER_BIN := ./linter

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
	@echo "  make lint-run        - Build and run custom linter"
	@echo "  make staticcheck-run - Run staticcheck on all packages"
	@echo "  make migrate-new name=NAME - Create new migration with given NAME"
	@echo "  make migrate-up       - Apply all up migrations"
	@echo "  make migrate-down     - Apply one down migration"
	@echo "  make migrate-reset    - Reset all migrations"
	@echo "  make migrate-fix      - Fix migration numbering"
	@echo "  make migrate-status   - Show migration status"
	@echo "  make godoc            - Start godoc server for module documentation"
	@echo "  make cover            - Run coverage tests with covertest"
	
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

GODOC_PORT ?= 8089
MODULE_NAME := metrics-tpl

godoc:
	@TMP_DIR=$$(mktemp -d /tmp/godoc-XXXXXX); \
	echo "Using temp dir: $$TMP_DIR"; \
	mkdir -p $$TMP_DIR/src/$(MODULE_NAME); \
	rsync -a \
		--exclude .git \
		--exclude vendor \
		./ $$TMP_DIR/src/$(MODULE_NAME); \
	echo "Starting godoc at http://localhost:$(GODOC_PORT)"; \
	GO111MODULE=off \
	GOMOD=/dev/null \
	GOPATH=$$TMP_DIR \
	GOCACHE=$$TMP_DIR/cache \
	godoc -http=:$(GODOC_PORT)

build-linter:
	go build -o $(LINTER_BIN) ./cmd/linter
	chmod +x $(LINTER_BIN)

lint-run: build-linter
	$(LINTER_BIN) ./...

staticcheck-run:
	staticcheck ./...