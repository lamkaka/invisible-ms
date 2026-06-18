.PHONY: build run test migrate seed docker-build docker-up docker-down docker-logs docker-restart

build:
	cd ../apps/api && go build -o bin/server ./cmd/server

run:
	cd ../apps/api && go run ./cmd/server

test:
	cd ../apps/api && go test ./...

migrate:
	cd ../apps/api && go run ./cmd/migrate

seed:
	test -d .venv || python3 -m venv .venv; \
	.venv/bin/pip install -q requests && \
	export WEBHOOK_SECRET=$$(grep -s '^WEBHOOK_SECRET=' .env | cut -d= -f2- || echo "test-secret"); \
	.venv/bin/python seed.py \
		--webhook-secret "$$WEBHOOK_SECRET" \
		$(if $(COMPANY_CODE),--company-code $(COMPANY_CODE)) \
		$(if $(COMPANY_NAME),--company-name $(COMPANY_NAME)) \
		--companies 1 --roles-per-company 3 --staff-per-company 5 \
		--checkout-fraction 1.0 --max-session-seconds 2.0

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart: docker-down docker-up
