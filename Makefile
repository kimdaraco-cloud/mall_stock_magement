# @ai-modified 2026-07-02 add project Makefile with dev/build/test/migrate targets
SHELL := /bin/bash

.PHONY: dev run build test lint fmt migrate-up migrate-down migrate-create seed db-up db-down

dev: ## run with live reload if air is installed, plain go run otherwise
	@if command -v air >/dev/null 2>&1; then air; else \
		echo "air not installed — falling back to go run"; go run ./cmd/server; fi

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else \
		echo "golangci-lint not installed — running go vet"; go vet ./...; fi

fmt:
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then goimports -w .; fi

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

# usage: make migrate-create name=add_products
migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=xxx" && exit 1)
	@ts=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${ts}_$(name).up.sql migrations/$${ts}_$(name).down.sql; \
	echo "created migrations/$${ts}_$(name).{up,down}.sql"

seed:
	go run ./scripts/seed

db-up: ## start the dev postgres container
	docker start mall-stock-pg 2>/dev/null || docker run -d --name mall-stock-pg \
		-e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=mall_stock \
		-p 5432:5432 postgres:16-alpine

db-down:
	docker stop mall-stock-pg
