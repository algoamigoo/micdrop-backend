include .env
export

.PHONY: docker-up docker-down docker-logs db-up db-down db-status db-reset run test test-cover test-integration build

# --- Docker Infrastructure ---
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# --- Database & Migrations ---
db-up:
	goose -dir migrations up

db-down:
	goose -dir migrations down

db-status:
	goose -dir migrations status

# Full reset — drops and recreates the schema, then migrates fresh
db-reset:
	goose -dir migrations reset
	goose -dir migrations up

db-create:
	goose -dir migrations create $(name) sql

# --- Application ---
run:
	go run cmd/server/main.go

build:
	go build -o bin/micdrop cmd/server/main.go

# --- Testing ---
test:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...

test-integration:
	go test -tags=integration -v ./internal/repository/
