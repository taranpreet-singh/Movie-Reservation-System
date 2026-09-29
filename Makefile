include .env
export 

MIGRATIONS_PATH = internal/migrations

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

run:
	go run ./cmd/api

.PHONY: migrate-up migrate-down run