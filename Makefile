.PHONY: build run test migrate

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

migrate:
	@echo "TODO: run migrations against Spanner (use spanner-cli or custom tool)"
	@echo "Example: for f in migrations/*.sql; do echo \"Running $$f...\"; done"
