.PHONY: help build run test tidy docker-up docker-down docker-logs migrate-up

help:
	@echo "Available commands:"
	@echo "  make tidy        - Download and clean Go module dependencies"
	@echo "  make build       - Compile server binary into bin/server"
	@echo "  make run         - Run server locally with environment variables"
	@echo "  make test        - Run automated tests"
	@echo "  make docker-up   - Start PostgreSQL with pgvector and application stack"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make docker-logs - Tail Docker container logs"

tidy:
	go mod tidy

build:
	go build -v -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v -race ./...

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
