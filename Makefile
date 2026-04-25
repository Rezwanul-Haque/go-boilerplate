.PHONY: dev dev-down build migrate-up migrate-down test test-unit test-integration lint tidy feature swagger

DB_URL=postgres://postgres:postgres@localhost:5432/go_boilerplate?sslmode=disable

dev:
	@[ -f .env ] || (cp .env.example .env && echo "Created .env from .env.example")
	docker-compose up --build

dev-down:
	docker-compose down

build:
	go build -o bin/server ./cmd/main.go

migrate-up:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path app/infra/database/migrations -database "$(DB_URL)" down

test: test-unit

test-unit:
	go test ./... -v -count=1

test-integration:
	docker-compose up -d postgres
	sleep 3
	go test ./... -v -count=1 -tags integration

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

feature:
	@if [ -z "$(name)" ]; then echo "Usage: make feature name=<feature-name>"; exit 1; fi
	go run ./cmd/scaffold/main.go $(name)

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main.go -o docs/swagger --parseDependency --parseInternal
