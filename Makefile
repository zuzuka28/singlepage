SHELL := /bin/sh

APP_NAME ?= singlepage
LISTEN ?= 127.0.0.1:8080
DB ?= ./data.db

.DEFAULT_GOAL := help

.PHONY: help install dev dev-web dev-api check lint test test-unit test-e2e test-go vet build build-web build-go run clean

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
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" go run .

check: ## Run frontend checks and Go static analysis
	npm run check
	go vet ./...
	golangci-lint run ./...

lint: ## Run the complete Go lint configuration
	golangci-lint run ./...

test: test-unit test-go test-e2e ## Run all tests

test-unit: ## Run frontend unit tests
	npm test

test-e2e: ## Run Playwright end-to-end tests
	npm run test:e2e

test-go: ## Run Go tests with race detection
	go test -race ./...

vet: ## Run Go vet
	go vet ./...

build: build-web build-go ## Build the production frontend and binary

build-web: ## Build the Svelte frontend
	npm run build

build-go: ## Build the Go binary
	go build -o $(APP_NAME) .

run: build ## Build and run the production application
	SINGLEPAGE_HTTP_LISTEN="$(LISTEN)" SINGLEPAGE_SQLITE_DSN="$(DB)" ./$(APP_NAME)

clean: ## Remove generated build and test artifacts
	rm -rf web/dist internal/handler/frontend/dist/assets playwright-report test-results
	rm -f internal/handler/frontend/dist/index.html
	rm -f $(APP_NAME)
