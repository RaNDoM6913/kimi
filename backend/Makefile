SHELL := /bin/bash

.PHONY: fmt lint test run-api run-bot migrate-up migrate-down

fmt:
	go fmt ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

test:
	go test ./...

run-api:
	go run ./cmd/api

run-bot:
	go run ./cmd/bot

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down
