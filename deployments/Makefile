COMPANY_CODE ?= ACME
COMPANY_NAME ?= ACME Corp
COMPANIES ?= 1
ROLES_PER_COMPANY ?= 3
STAFF_PER_COMPANY ?= 5
CHECKOUT_FRACTION ?= 1.0
MAX_SESSION_SECONDS ?= 2.0

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
		--company-code "$(COMPANY_CODE)" \
		--company-name "$(COMPANY_NAME)" \
		--companies $(COMPANIES) --roles-per-company $(ROLES_PER_COMPANY) --staff-per-company $(STAFF_PER_COMPANY) \
		--checkout-fraction $(CHECKOUT_FRACTION) --max-session-seconds $(MAX_SESSION_SECONDS)

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart: docker-down docker-up
