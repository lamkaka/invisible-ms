# Repo Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the repository into `apps/`, `deployments/`, and `docs/` buckets; split the monolithic `AGENTS.md` into concern-based rulebooks; rename the interface layer from `handler` to `controller`; and verify builds, tests, and Docker Compose.

**Architecture:** Go backend moves to `apps/api/` as the module root; server-rendered web assets move to `apps/web/`; nginx moves to `apps/infra/nginx/`; deployment artifacts move to `deployments/`; agent-facing docs split under `docs/rules/` and per-cell `AGENTS.md` files.

**Tech Stack:** Go, Google Cloud Spanner, Docker Compose, nginx, Alpine.js.

---

## Task 1: Move Go module to `apps/api/`

**Files:**
- Move: `cmd/` → `apps/api/cmd/`
- Move: `internal/` → `apps/api/internal/`
- Move: `go.mod` → `apps/api/go.mod`
- Move: `go.sum` → `apps/api/go.sum`
- Create: `apps/api/README.md`

- [ ] **Step 1: Create destination directories**

```bash
mkdir -p apps/api/cmd apps/api/internal
```

- [ ] **Step 2: Move Go source and module files**

```bash
git mv cmd apps/api/
git mv internal apps/api/
git mv go.mod apps/api/
git mv go.sum apps/api/
```

- [ ] **Step 3: Add `apps/api/README.md`**

```markdown
# IMS API

Go backend for the IMS hourly-staff management system.

## Run

```bash
cd apps/api
go run ./cmd/server
```

## Test

```bash
cd apps/api
go test ./...
```
```

- [ ] **Step 4: Verify the module still builds**

```bash
cd apps/api && go build ./...
```

Expected: build succeeds.

---

## Task 2: Move web assets to `apps/web/`

**Files:**
- Move: `templates/` → `apps/web/templates/`
- Move: `web/static/` → `apps/web/static/`

- [ ] **Step 1: Create destination directories and move files**

```bash
mkdir -p apps/web
git mv templates apps/web/
git mv web/static apps/web/
rm -rf web
```

- [ ] **Step 2: Verify only `apps/web/` remains for frontend assets**

```bash
ls apps/web/
```

Expected: `static/`, `templates/`.

---

## Task 3: Move infrastructure config to `apps/infra/`

**Files:**
- Move: `nginx/` → `apps/infra/nginx/`

- [ ] **Step 1: Move nginx config**

```bash
mkdir -p apps/infra
git mv nginx apps/infra/
```

---

## Task 4: Move deployment artifacts to `deployments/`

**Files:**
- Move: `docker-compose.yml` → `deployments/docker-compose.yml`
- Move: `Dockerfile` → `apps/api/Dockerfile`
- Move: `.dockerignore` → `deployments/.dockerignore`
- Move: `Makefile` → `deployments/Makefile`
- Move: `.env.example` → `deployments/.env.example`
- Move: `migrations/` → `apps/api/migrations/`
- Create: `deployments/AGENTS.md`

- [ ] **Step 1: Create destination directory and move files**

```bash
mkdir -p deployments
mkdir -p apps/api/migrations && git mv deployments/migrations/*.sql apps/api/migrations/ && git mv Dockerfile apps/api/Dockerfile && git mv docker-compose.yml .dockerignore Makefile .env.example deployments/
```

- [ ] **Step 2: Add `deployments/AGENTS.md`**

```markdown
# Deployments

This folder contains everything needed to run IMS locally with Docker Compose.

## Files

- `docker-compose.yml` — local stack: Spanner emulator, API, nginx
- `../apps/api/Dockerfile` — API application image (in apps/api)
- `../apps/api/migrations/` — Spanner DDL migrations (in apps/api)
- `Makefile` — common commands
- `.env.example` — required environment variables

## Quick start

```bash
cd deployments
cp .env.example .env
make up
```

## Run migrations only

```bash
make migrate
```
```

---

## Task 5: Move `STATUS.md` to `docs/superpowers/plans/`

**Files:**
- Move: `STATUS.md` → `docs/superpowers/plans/STATUS.md`

- [ ] **Step 1: Move STATUS.md**

```bash
git mv STATUS.md docs/superpowers/plans/STATUS.md
```

---

## Task 6: Rename interface layer from `handler` to `controller`

**Files:**
- Rename: `apps/api/internal/company/company_handler.go` → `company_controller.go`
- Rename: `apps/api/internal/company/company_handler_test.go` → `company_controller_test.go`
- Rename: `apps/api/internal/staff/staff_handler.go` → `staff_controller.go`
- Rename: `apps/api/internal/staff/staff_handler_test.go` → `staff_controller_test.go`
- Rename: `apps/api/internal/activity/activity_handler.go` → `activity_controller.go`
- Rename: `apps/api/internal/activity/activity_handler_test.go` → `activity_controller_test.go`
- Rename: `apps/api/internal/dashboard/dashboard_api_handler.go` → `dashboard_api_controller.go`
- Rename: `apps/api/internal/dashboard/dashboard_api_handler_test.go` → `dashboard_api_controller_test.go`
- Rename: `apps/api/internal/dashboard/dashboard_web_handler.go` → `dashboard_web_controller.go`
- Rename: `apps/api/internal/dashboard/dashboard_web_handler_test.go` → `dashboard_web_controller_test.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Rename all handler files and tests**

```bash
cd apps/api/internal
for cell in company staff activity dashboard; do
  cd "$cell"
  for f in *_handler.go; do git mv "$f" "${f%_handler.go}_controller.go"; done
  for f in *_handler_test.go; do git mv "$f" "${f%_handler_test.go}_controller_test.go"; done
  cd ..
done
cd ../../..
```

- [ ] **Step 2: Replace `Handler` with `Controller` in Go source**

Run from repo root:

```bash
find apps/api -type f -name '*.go' -exec sed -i '' 's/Handler/Controller/g' {} +
```

Then review the diff for any unintended renames (e.g., variables named `handler` in comments).

```bash
git diff --stat
```

- [ ] **Step 3: Verify builds and tests still pass**

```bash
cd apps/api && go test ./...
```

Expected: all tests pass.

---

## Task 7: Update Go code for new asset paths

**Files:**
- Modify: `apps/api/cmd/server/main.go`
- Modify: `apps/api/internal/shared/config.go`

- [ ] **Step 1: Add asset path config fields**

Edit `apps/api/internal/shared/config.go`:

```go
func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(GetEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	return &Config{
		SpannerProjectID:   GetEnv("SPANNER_PROJECT_ID", ""),
		SpannerInstanceID:  GetEnv("SPANNER_INSTANCE_ID", ""),
		SpannerDatabaseID:  GetEnv("SPANNER_DATABASE_ID", ""),
		Port:               port,
		WebhookSecret:      GetEnv("WEBHOOK_SECRET", ""),
		CORSAllowedOrigins: GetEnv("CORS_ALLOWED_ORIGINS", "*"),
		TemplatesPath:      GetEnv("TEMPLATES_PATH", "../web/templates"),
		StaticPath:         GetEnv("STATIC_PATH", "../web/static"),
	}, nil
}
```

Add to the `Config` struct:

```go
TemplatesPath string
StaticPath    string
```

- [ ] **Step 2: Use config paths in main.go**

Edit `apps/api/cmd/server/main.go`:

```go
dashboardWebHandler, err := dashboard.NewDashboardWebHandler(dashboardService, cfg.TemplatesPath)
// ...
router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticPath))))
```

- [ ] **Step 3: Verify server builds**

```bash
cd apps/api && go build ./cmd/server
```

Expected: build succeeds.

---

## Task 8: Create API Dockerfile

**Files:**
- Create: `apps/api/Dockerfile`

- [ ] **Step 1: Create `apps/api/Dockerfile`**

```dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /build

ENV GOTOOLCHAIN=auto

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /app/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk --no-cache add ca-certificates wget
WORKDIR /app/api

COPY --from=builder /app/server /app/api/server
COPY --from=builder /app/migrate /app/api/migrate
COPY apps/web/ /app/web/
COPY apps/api/migrations/ /app/api/migrations/

ENV TEMPLATES_PATH=/app/web/templates
ENV STATIC_PATH=/app/web/static
ENV PORT=8080

EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 2: Build the image from repo root to verify**

```bash
docker build -f apps/api/Dockerfile -t ims-api .
```

Expected: image builds successfully.

---

## Task 9: Update docker-compose.yml

**Files:**
- Modify: `deployments/docker-compose.yml`

- [ ] **Step 1: Update build contexts and dockerfile paths**

Edit `deployments/docker-compose.yml`:

```yaml
services:
  migrate:
    build:
      context: ..
      dockerfile: apps/api/Dockerfile
    # ...

  app:
    build:
      context: ..
      dockerfile: apps/api/Dockerfile
    # ...

  nginx:
    build:
      context: ..
      dockerfile: apps/infra/nginx/Dockerfile
    # ...
```

- [ ] **Step 2: Update migrate command path**

Ensure the migrate service still runs `/app/api/migrate` (or wherever the binary lands after the Dockerfile change).

---

## Task 10: Update nginx Dockerfile

**Files:**
- Modify: `apps/infra/nginx/Dockerfile`

- [ ] **Step 1: Update COPY path**

Edit `apps/infra/nginx/Dockerfile`:

```dockerfile
FROM nginx:alpine
COPY apps/infra/nginx/nginx.conf /etc/nginx/nginx.conf
```

---

## Task 11: Update Makefile

**Files:**
- Modify: `deployments/Makefile`

- [ ] **Step 1: Update paths in Makefile commands**

Common changes:

```makefile
.PHONY: up down migrate test build

up:
	docker compose up --build

down:
	docker compose down

migrate:
	docker compose run --rm migrate

test:
	cd ../apps/api && go test ./...

build:
	docker build -f Dockerfile -t ims-api ..
```

Review the existing `deployments/Makefile` and adjust any remaining paths.

---

## Task 12: Split root `AGENTS.md` into `docs/rules/`

**Files:**
- Create: `docs/rules/01-architecture.md`
- Create: `docs/rules/02-domain-model.md`
- Create: `docs/rules/03-database.md`
- Create: `docs/rules/04-api-and-webhook.md`
- Create: `docs/rules/05-testing.md`
- Create: `docs/rules/06-development.md`
- Rewrite: `AGENTS.md` (root)

- [ ] **Step 1: Create `docs/rules/` directory**

```bash
mkdir -p docs/rules
```

- [ ] **Step 2: Extract architecture content**

Create `docs/rules/01-architecture.md` from the original `AGENTS.md` sections "Architecture Principles", "File Naming Convention", "Project Structure", "Code Organization", and "Dependency Injection". Update the file naming table to say `*_controller.go` for the interface layer.

- [ ] **Step 3: Extract domain model content**

Create `docs/rules/02-domain-model.md` from "Domain Model" and "Business Rules".

- [ ] **Step 4: Extract database content**

Create `docs/rules/03-database.md` from "Database Schema", "Spanner Transaction Patterns", and "Dashboard Query Patterns".

- [ ] **Step 5: Extract API content**

Create `docs/rules/04-api-and-webhook.md` from "API Endpoints" and "Webhook Security". Replace "Handler responsibilities" with "Controller responsibilities".

- [ ] **Step 6: Extract testing content**

Create `docs/rules/05-testing.md` from "Testing".

- [ ] **Step 7: Extract development guidelines**

Create `docs/rules/06-development.md` from "Development Guidelines", "Error Handling", "Configuration", "Performance Guidelines", and "Common Pitfalls".

- [ ] **Step 8: Rewrite root `AGENTS.md`**

```markdown
# IMS - Hourly Staff Management System

IMS is a multi-tenant HR application for managing hourly staff. Workers check in/out via WhatsApp keyword commands; the system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

## Tech Stack

- **Backend:** Go
- **Database:** Google Cloud Spanner
- **Frontend:** Server-rendered HTML + Alpine.js
- **Messaging:** External WhatsApp/Waha webhook
- **Architecture:** DDD + Clean Architecture + Cell-Based Architecture

## Agent Quick Start

- Working on a cell → read `apps/api/internal/<cell>/AGENTS.md`
- Database questions → read `docs/rules/03-database.md`
- API or webhook questions → read `docs/rules/04-api-and-webhook.md`
- Architecture rules → read `docs/rules/01-architecture.md`
- Running/deploying → read `deployments/AGENTS.md`

## Project Status

See `docs/superpowers/plans/STATUS.md`.

## MVP Scope

- No authentication
- WhatsApp integration via external webhook
- Basic dashboard with today's stats
- Company and staff management via REST API
- Check-in/check-out via WhatsApp keywords

## Future Enhancements

- Authentication
- Additional action types
- Export reports
- Email notifications
- Mobile app for managers
```

---

## Task 13: Add per-cell `AGENTS.md`

**Files:**
- Create: `apps/api/internal/shared/AGENTS.md`
- Create: `apps/api/internal/company/AGENTS.md`
- Create: `apps/api/internal/staff/AGENTS.md`
- Create: `apps/api/internal/activity/AGENTS.md`
- Create: `apps/api/internal/dashboard/AGENTS.md`

- [ ] **Step 1: Create cell-local context files**

For each cell, create an `AGENTS.md` that includes:
- Cell purpose
- Owned aggregate(s)
- File list and responsibilities
- Inbound/outbound dependencies
- API endpoints provided
- Cell-specific business rules
- Link to `docs/rules/01-architecture.md` for naming and layer rules

Example for `apps/api/internal/company/AGENTS.md`:

```markdown
# Company Cell

## Purpose

Manage companies and their role catalogs. This is the standalone root cell; other cells depend on it.

## Aggregate

- `Company` — tenant identifier, name, roles

## Files

- `company_domain.go` — aggregate, `Role` value object, validation
- `company_service.go` — CRUD orchestration
- `company_repository.go` — Spanner adapter
- `company_controller.go` — REST endpoints
- `company_action_type_repository.go` — configurable action type storage

## Dependencies

- None (standalone cell)
- `staff` cell depends on this cell

## API Endpoints

- `GET /api/companies`
- `POST /api/companies`
- `GET /api/companies/:code`
- `PUT /api/companies/:code`
- `POST /api/companies/:code/roles`
- `DELETE /api/companies/:code/roles/:role`
- `GET /api/companies/:code/action-types`
- `POST /api/companies/:code/action-types`
- `PATCH /api/companies/:code/action-types/:action_type/keyword`
- `DELETE /api/companies/:code/action-types/:action_type`

## Rules

- Company code is immutable.
- Role hourly rates must be non-negative.
- Removing a role assigned to staff should be prevented.

## Shared Conventions

See `docs/rules/01-architecture.md` for layer rules and naming.
```

Create similar files for `staff`, `activity`, `dashboard`, and `shared`.

---

## Task 14: Update cross-references

**Files:**
- Modify: `README.md`
- Modify: `docs/rules/*.md`
- Modify: cell `AGENTS.md` files

- [ ] **Step 1: Update `README.md` paths**

Replace references to root-level files with new paths:
- `STATUS.md` → `docs/superpowers/plans/STATUS.md`
- `AGENTS.md` → `AGENTS.md` (still root) or specific `docs/rules/` files

- [ ] **Step 2: Verify internal links**

Run:

```bash
grep -R "\.\./\|docs/rules\|internal/" docs/rules AGENTS.md README.md
```

Fix any broken relative links.

---

## Task 15: Final verification

- [ ] **Step 1: Run Go tests**

```bash
cd apps/api && go test ./...
```

Expected: all tests pass.

- [ ] **Step 2: Build server binary**

```bash
cd apps/api && go build ./cmd/server
```

Expected: build succeeds.

- [ ] **Step 3: Build Docker images**

```bash
cd deployments
docker compose build
```

Expected: both `app` and `nginx` images build successfully.

- [ ] **Step 4: Run full stack briefly**

```bash
cd deployments
docker compose up -d
sleep 15
curl -f http://localhost:8888/api/dashboard/stats
docker compose down
```

Expected: HTTP 200 from the dashboard stats endpoint.

- [ ] **Step 5: Check git status**

```bash
git status
```

Expected: no leftover untracked files at old paths; all moves are reflected.

---

## Self-Review Checklist

- [ ] Every file in the original repo has a destination in the new structure
- [ ] No `*_handler.go` or `*Handler` type remains
- [ ] `apps/api/` is a valid Go module (`go test ./...` passes)
- [ ] Docker Compose builds from `deployments/`
- [ ] Root `AGENTS.md` is under 80 lines
- [ ] `docs/rules/` contains exactly six files
- [ ] Each cell has an `AGENTS.md`
