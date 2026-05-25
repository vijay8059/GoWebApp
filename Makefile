## GoWebApp — developer Makefile.
## All targets are POSIX-shell friendly. Run `make help` for the catalog.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

APP        := gowebapp
PKG        := github.com/GoWebApp
BIN_DIR    := bin
CMD_DIR    := ./cmd/server
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
GOFLAGS    := -trimpath
DOCKER_IMG ?= $(APP):$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_.-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## Sync go.mod / go.sum.
	go mod tidy

.PHONY: fmt
fmt: ## Format Go sources.
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install/).
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

.PHONY: test
test: ## Run unit tests with race detector and coverage.
	go test -race -count=1 -coverprofile=coverage.out ./...

.PHONY: cover
cover: test ## Open HTML coverage report.
	go tool cover -html=coverage.out

.PHONY: build
build: ## Build a static binary into bin/.
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags='$(LDFLAGS)' -o $(BIN_DIR)/$(APP) $(CMD_DIR)

.PHONY: run
run: ## Run the server locally.
	go run $(CMD_DIR)

.PHONY: token
token: ## Mint a JWT for local testing. Requires JWT_SECRET in env. Usage: make token SUB=alice
	@JWT_SECRET=$${JWT_SECRET:?set JWT_SECRET} \
		go run ./cmd/token -sub $${SUB:-test-user}

.PHONY: docker
docker: ## Build the Docker image.
	docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMG) .

.PHONY: up
up: ## Start the full stack via docker-compose.
	docker compose up --build -d

.PHONY: down
down: ## Stop the docker-compose stack.
	docker compose down -v

.PHONY: logs
logs: ## Tail app logs from the docker-compose stack.
	docker compose logs -f app

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR) coverage.out

.PHONY: ci
ci: tidy vet test build ## Run the full CI gate locally.
