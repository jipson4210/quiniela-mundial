.PHONY: run test migrate-up migrate-down migrate-create sqlc dev build clean

APP_NAME := quiniela-api
DB_URL  ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Include .env if present
ifneq (,$(wildcard .env))
    include .env
    export
endif

## build: compile the API binary
build:
	go build -o bin/$(APP_NAME) ./cmd/api/

## run: start the API server
run: build
	./bin/$(APP_NAME)

## dev: run with hot-reload (requires air)
dev:
	air

## test: run all tests
test:
	go test ./... -count=1 -short

## test-integration: run integration tests with Docker DB
test-integration:
	go test ./... -count=1 -tags=integration

## migrate-up: apply all pending migrations
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

## migrate-down: revert last migration
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

## migrate-create: create new migration pair (usage: make migrate-create NAME=add_table)
migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

## sqlc: generate type-safe Go from SQL queries
sqlc:
	sqlc generate

## docker-up: start all services
docker-up:
	docker compose up -d

## docker-down: stop all services
docker-down:
	docker compose down

## clean: remove build artifacts
clean:
	rm -rf bin/
	go clean -cache -testcache

## lint: run golangci-lint
lint:
	golangci-lint run ./...
