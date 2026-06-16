.PHONY: build run test migrate docker-build docker-up docker-down docker-logs docker-restart

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

migrate:
	go run ./cmd/migrate

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart: docker-down docker-up
