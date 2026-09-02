SHELL := /bin/sh

APP_NAME ?= singlepage
LISTEN ?= 127.0.0.1:8080
DB ?= ./data.db

.DEFAULT_GOAL := help

.PHONY: help install dev dev-web dev-api dev-daemon dev-app generate generate-openapi generate-wire bindings-app check check-app lint test test-unit test-e2e test-go vet build build-web build-go build-daemon build-app run run-daemon run-app clean

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install frontend dependencies
	npm install

dev: ## Run frontend and backend development servers
	@trap 'kill 0' INT TERM EXIT; \
		$(MAKE) dev-api & \
		$(MAKE) dev-web & \
		wait

dev-web: ## Run Vite development server on :5173
	npm run dev

dev-api: ## Run Go API server
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" go run ./cmd/web

dev-daemon: ## Run the headless API daemon
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" go run ./cmd/daemon

dev-app: ## Build and run the Wails v3 app without the Wails CLI watcher
	$(MAKE) build-app
	$(MAKE) run-app

check: check-openapi check-app lint ## Run frontend checks and Go static analysis, including Wails-tagged code
	npm run check
	go vet ./...

generate: generate-openapi generate-wire bindings-app ## Regenerate OpenAPI, Wire, and Wails generated code

generate-openapi: ## Regenerate OpenAPI server and public REST and TypeScript clients
	go generate ./internal/handler/httpapi/gen ./pkg/rest
	npm run api:generate

.PHONY: check-openapi
check-openapi: ## Verify committed TypeScript OpenAPI types have not drifted
	npm run api:check

generate-wire: ## Regenerate the application dependency graph
	go tool wire ./internal/provider

bindings-app: ## Regenerate Wails TypeScript bindings
	go tool wails3 generate bindings -ts -b -names -noevents -f='-tags=wails' -d cmd/app/internal/app/frontend/bindings ./cmd/app/...

check-app: ## Compile, test, and vet Wails-tagged app code
	go test -tags wails ./cmd/app ./cmd/app/internal/app ./cmd/app/internal/page ./cmd/app/internal/session ./internal/provider
	go vet -tags wails ./cmd/app ./cmd/app/internal/app ./cmd/app/internal/page ./cmd/app/internal/session ./internal/provider
	mkdir -p bin
	go build -tags wails -o bin/$(APP_NAME)-app-check ./cmd/app

lint: ## Run the complete Go lint configuration
	golangci-lint run ./...
	golangci-lint run --build-tags wails ./cmd/app ./cmd/app/internal/app ./cmd/app/internal/page ./cmd/app/internal/session ./internal/provider

test: test-unit test-go test-e2e ## Run all tests

test-unit: ## Run frontend unit tests
	npm test

test-e2e: ## Run Playwright end-to-end tests
	npm run test:e2e

test-go: ## Run Go tests with race detection
	go test -race ./...
	go test -race -tags wails ./cmd/app ./cmd/app/internal/app ./cmd/app/internal/page ./cmd/app/internal/session ./internal/provider

vet: ## Run Go vet
	go vet ./...

build: build-web build-go build-daemon ## Build browser and daemon binaries

build-web: ## Build the Svelte frontend
	rm -rf internal/handler/frontend/dist/assets internal/handler/frontend/dist/index.html
	npm run build

build-go: ## Build the Go binary
	go build -o $(APP_NAME) ./cmd/web

build-daemon: ## Build the headless API daemon
	mkdir -p bin
	go build -o bin/$(APP_NAME)-daemon ./cmd/daemon

build-app: bindings-app ## Build the Wails v3 desktop application
	rm -rf cmd/app/internal/app/frontend/dist/assets cmd/app/internal/app/frontend/dist/index.html
	VITE_SINGLEPAGE_RUNTIME=wails SINGLEPAGE_TARGET=app npm run build
	mkdir -p bin
	go build -tags 'wails production' -o bin/$(APP_NAME)-app ./cmd/app

run: build ## Build and run the production application
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" ./$(APP_NAME)

run-daemon: build-daemon ## Build and run the headless API daemon
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" ./bin/$(APP_NAME)-daemon

run-app: ## Run the previously built Wails v3 desktop app
	./bin/$(APP_NAME)-app

clean: ## Remove generated build and test artifacts
	rm -rf web/dist internal/handler/frontend/dist/assets cmd/app/internal/app/frontend/dist/assets playwright-report test-results
	rm -f internal/handler/frontend/dist/index.html
	rm -f cmd/app/internal/app/frontend/dist/index.html
	rm -f $(APP_NAME)
	rm -f bin/$(APP_NAME)-daemon
	rm -f bin/$(APP_NAME)-app
	rm -f bin/$(APP_NAME)-app-check
