SHELL := /bin/bash

.PHONY: run test tidy fmt up down

run:
	go run ./cmd/bot

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

up:
	docker compose up -d

down:
	docker compose down -v
