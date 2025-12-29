SERVER_DIR=cmd/server
AGENT_DIR=cmd/agent

SERVER_BIN=$(SERVER_DIR)/server
AGENT_BIN=$(AGENT_DIR)/agent

METRICSTEST_BIN=./metricstest

ITERATION ?= 1

MIGRATIONS_DIR=./migrations
GOOSE_BIN=goose
PG_DSN=${DATABASE_DSN}

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

vet:
	go vet ./...

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
	$(SERVER_BIN) -a=localhost:8080

run-agent:
	$(AGENT_BIN) -a=http://localhost:8080 -R=10 -p=2

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
