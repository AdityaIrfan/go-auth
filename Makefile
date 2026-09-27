# Load environment variables dari .env
-include .env
export

.PHONY: run build up down swagger test test-cover migrate-create migrate-up migrate-down

DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"

run:
	go run cmd/api/main.go

build:
	go build -buildvcs=false -o bin/api cmd/api/main.go

dev:
	docker-compose -f deploy/docker-compose.local.yml up

dev-build:
	docker-compose -f deploy/docker-compose.local.yml up --build

up:
	docker compose -f deploy/docker-compose.local.yml up -d

down:
	docker compose -f deploy/docker-compose.local.yml down

swagger:
	swag init -g cmd/api/main.go --parseInternal -o docs

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up:
	migrate -path migrations -database $(DB_URL) up

migrate-down:
	migrate -path migrations -database $(DB_URL) down 1
