.PHONY: dev dev-down build migrations up down test test-unit test-integration lint tidy feature rm swagger

DB_URL=postgres://postgres:postgres@localhost:5432/go_boilerplate?sslmode=disable

dev:
	@[ -f .env ] || (cp .env.example .env && echo "Created .env from .env.example")
	docker-compose up --build

dev-down:
	docker-compose down

build:
	go build -o bin/server ./cmd/main.go

migrations:
	@if echo "$(MAKECMDGOALS)" | grep -qw "up"; then \
		migrate -path app/infra/database/migrations -database "$(DB_URL)" up; \
	elif echo "$(MAKECMDGOALS)" | grep -qw "down"; then \
		migrate -path app/infra/database/migrations -database "$(DB_URL)" down; \
	elif [ -n "$(name)" ]; then \
		go run ./cmd/scaffold/migrate/main.go $(name); \
	else \
		echo "Usage:"; \
		echo "  make migrations name=<feature>   generate migration files"; \
		echo "  make migrations up               apply all pending migrations"; \
		echo "  make migrations down             roll back last migration"; \
		exit 1; \
	fi

up: ;
down: ;

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
	@if [ -z "$(name)" ]; then echo "Usage: make feature [rm] name=<feature-name>"; exit 1; fi
	@if echo "$(MAKECMDGOALS)" | grep -qw "rm"; then \
		go run ./cmd/scaffold/main.go rm $(name); \
	else \
		go run ./cmd/scaffold/main.go $(name); \
	fi

rm: ;

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main.go -o docs/swagger --parseDependency --parseInternal
