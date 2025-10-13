SERVER_DIR=cmd/server
AGENT_DIR=cmd/agent

SERVER_BIN=$(SERVER_DIR)/server
AGENT_BIN=$(AGENT_DIR)/agent

METRICSTEST_BIN=./metricstest

ITERATION ?= 1

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
		$(if $(SOURCE_PATH),-source-path=$(SOURCE_PATH))


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
