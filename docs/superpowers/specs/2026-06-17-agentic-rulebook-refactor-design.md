# Repo Restructure for Agentic Context

**Date:** 2026-06-17  
**Status:** Pending Approval

## Problem Statement

The repository mixes code, docs, infrastructure, and deployment artifacts at the root. Root `AGENTS.md` is a single 419-line document that combines project overview, architecture principles, domain model, API endpoints, database schema, development guidelines, and MVP notes. When an agent works on a narrow task, it must load unrelated context to find the rules that matter.

## Solution Overview

Reorganize the repo into four top-level buckets:

- `apps/` — application code (`api`, `web`, `infra`)
- `deployments/` — runnable deployment artifacts (`docker-compose.yml`, `Dockerfile`, migrations, `Makefile`, `.env.example`)
- `docs/` — agent-facing rulebooks and project plans
- Root files — `AGENTS.md`, `README.md`, `.gitignore`

Inside `docs/`, split the monolithic `AGENTS.md` into smaller concern-based rulebooks under `docs/rules/`, and add a local `AGENTS.md` to each cell under `apps/api/internal/`. Root `AGENTS.md` becomes a thin landing page.

**Goal:** An agent loads only the context relevant to its task:
- Editing a cell → read `apps/api/internal/<cell>/AGENTS.md`
- Adding a database query → read `docs/rules/03-database.md`
- Working on HTTP behavior → read `docs/rules/04-api-and-webhook.md`
- Running/deploying → read `deployments/AGENTS.md`

## Target Structure

```
AGENTS.md                              # landing page only
README.md                              # human-facing project overview
.gitignore                             # root ignore rules

docs/
├── rules/
│   ├── 01-architecture.md             # DDD, Clean Arch, cells, file naming, dependency direction
│   ├── 02-domain-model.md             # aggregates, value objects, entities, session computation
│   ├── 03-database.md                 # Spanner schema, migrations, transaction patterns, query patterns
│   ├── 04-api-and-webhook.md          # HTTP conventions, endpoints, webhook security, status codes
│   ├── 05-testing.md                  # test conventions by layer, mocks
│   └── 06-development.md              # errors, config, performance, common pitfalls
└── superpowers/
    ├── specs/...
    └── plans/
        ├── STATUS.md                  # moved from root
        └── ...

apps/
├── api/                               # Go module root
│   ├── cmd/
│   │   ├── server/main.go
│   │   ├── migrate/main.go
│   │   └── setup/main.go
│   ├── internal/
│   │   ├── shared/
│   │   │   └── AGENTS.md
│   │   ├── company/
│   │   │   ├── AGENTS.md
│   │   │   └── ...
│   │   ├── staff/
│   │   │   ├── AGENTS.md
│   │   │   └── ...
│   │   ├── activity/
│   │   │   ├── AGENTS.md
│   │   │   └── ...
│   │   └── dashboard/
│   │       ├── AGENTS.md
│   │       └── ...
│   ├── go.mod                         # module path stays github.com/lamkaka/invisible-ms
│   ├── go.sum
│   └── README.md                      # api app quick-start
│
├── web/                               # server-rendered templates and static assets
│   ├── templates/
│   │   ├── layout.html
│   │   ├── dashboard.html
│   │   ├── staff.html
│   │   └── actions.html
│   └── static/
│       ├── css/style.css
│       └── js/app.js
│
└── infra/
    └── nginx/
        ├── Dockerfile
        └── nginx.conf

deployments/
├── docker-compose.yml
├── Dockerfile                         # api app image
├── .dockerignore
├── Makefile
├── .env.example
├── migrations/
│   ├── 001_create_companies.sql
│   ├── 002_create_staff.sql
│   ├── 003_create_activity_logs.sql
│   └── 004_create_company_action_types.sql
└── AGENTS.md                          # deployment context
```

## File Movements

| Current path | New path |
|--------------|----------|
| `cmd/server/main.go` | `apps/api/cmd/server/main.go` |
| `cmd/migrate/main.go` | `apps/api/cmd/migrate/main.go` |
| `cmd/setup/main.go` | `apps/api/cmd/setup/main.go` |
| `internal/...` | `apps/api/internal/...` |
| `go.mod` | `apps/api/go.mod` |
| `go.sum` | `apps/api/go.sum` |
| `templates/` | `apps/web/templates/` |
| `web/static/` | `apps/web/static/` |
| `nginx/` | `apps/infra/nginx/` |
| `docker-compose.yml` | `deployments/docker-compose.yml` |
| `Dockerfile` | `deployments/Dockerfile` |
| `.dockerignore` | `deployments/.dockerignore` |
| `Makefile` | `deployments/Makefile` |
| `.env.example` | `deployments/.env.example` |
| `migrations/` | `deployments/migrations/` |
| `STATUS.md` | `docs/superpowers/plans/STATUS.md` |
| `AGENTS.md` (root) | split across `docs/rules/` and cell `AGENTS.md` files |

## Code Changes Required

### Go module path
Keep `github.com/lamkaka/invisible-ms` as the module path inside `apps/api/go.mod`. Internal imports stay unchanged.

### Template and static asset paths
`cmd/server/main.go` currently loads `./templates` and `./web/static` from the working directory. After the move, the server must find `apps/web/templates` and `apps/web/static`.

Because `apps/api/` is the Go module root, the server usually runs from `apps/api/`. Relative paths from there point upward one level.

**Decision:** Make asset paths configurable via environment variables with defaults relative to `apps/api/`:
- `TEMPLATES_PATH` defaults to `../web/templates`
- `STATIC_PATH` defaults to `../web/static`

For local development from the repo root, set `TEMPLATES_PATH=apps/web/templates` and `STATIC_PATH=apps/web/static`.

Inside the Docker image, set `WORKDIR /app/api` and copy `apps/api/` to `/app/api/` and `apps/web/` to `/app/web/`. The defaults then resolve to `/app/web/templates` and `/app/web/static`.

### Dockerfile
Move `Dockerfile` to `deployments/Dockerfile`. Update the build context in `deployments/docker-compose.yml` to `..` so the API image can access both `apps/api/` and `apps/web/`.

### docker-compose.yml
Move to `deployments/docker-compose.yml`. Update:
- API service: `context: ..`, `dockerfile: deployments/Dockerfile`
- Nginx service: `context: ..`, `dockerfile: apps/infra/nginx/Dockerfile`
- Spanner emulator stays as-is (it is a deployment concern)

### Makefile
Move to `deployments/Makefile`. Update any paths that assume a root location (e.g., `go test ./...` becomes `cd ../apps/api && go test ./...`).

### nginx Dockerfile
Move to `apps/infra/nginx/Dockerfile`. Update its `COPY nginx/nginx.conf` path to `apps/infra/nginx/nginx.conf` relative to the repo-root build context.

### Handler to Controller rename
Rename the interface layer from "handler" to "controller" across the Go codebase:
- File suffix: `*_handler.go` → `*_controller.go`
- Test files: `*_handler_test.go` → `*_controller_test.go`
- Type names: `*Handler` → `*Controller`
- Constructors: `New*Handler` → `New*Controller`
- Receiver variables: `h` → `c` where they refer to a controller
- Mock and helper names in tests: `handlerMock*` → `controllerMock*`, `handlerTestMocks` → `controllerTestMocks`
- Update references in `cmd/server/main.go`
- Update the layer description in `docs/rules/01-architecture.md` from "Handler layer" to "Controller layer"

Keep method names like `RegisterRoutes` and HTTP handler function signatures unchanged.

## Content Mapping for `AGENTS.md`

| Current `AGENTS.md` section | New home |
|----------------------------|----------|
| Overview + Tech Stack | Root `AGENTS.md` landing page |
| Architecture Principles | `docs/rules/01-architecture.md` |
| File Naming Convention | `docs/rules/01-architecture.md` |
| Project Structure | Root `AGENTS.md` + cell `AGENTS.md` files |
| Domain Model | `docs/rules/02-domain-model.md` + cell `AGENTS.md` files |
| Business Rules | `docs/rules/02-domain-model.md` + cell `AGENTS.md` files |
| Dashboard Requirements | `apps/api/internal/dashboard/AGENTS.md` |
| API Endpoints | `docs/rules/04-api-and-webhook.md` + cell `AGENTS.md` files |
| Database Schema | `docs/rules/03-database.md` |
| Code Organization | `docs/rules/01-architecture.md` |
| Dependency Injection | `docs/rules/01-architecture.md` |
| Error Handling | `docs/rules/06-development.md` |
| Spanner Transaction Patterns | `docs/rules/03-database.md` |
| Webhook Security | `docs/rules/04-api-and-webhook.md` |
| Role Validation | `apps/api/internal/staff/AGENTS.md` + `apps/api/internal/company/AGENTS.md` |
| Testing | `docs/rules/05-testing.md` |
| Configuration | `docs/rules/06-development.md` |
| Performance Guidelines | `docs/rules/06-development.md` |
| Dashboard Query Patterns | `docs/rules/03-database.md` + `apps/api/internal/dashboard/AGENTS.md` |
| Common Pitfalls | `docs/rules/06-development.md` |
| MVP Scope | Root `AGENTS.md` |
| Future Enhancements | Root `AGENTS.md` or `README.md` |

## Per-File Responsibilities

### `AGENTS.md` (root)
- One-paragraph project overview
- Tech stack bullet list
- Links to `docs/rules/`, cell `AGENTS.md` files, and `deployments/AGENTS.md`
- Quick-start mapping: "If you are editing X, read Y"
- MVP scope and future enhancements

### `docs/rules/01-architecture.md`
- Domain-Driven Design principles
- Clean Architecture layers and dependency direction
- Cell-Based Architecture rules
- Cell dependency graph
- File naming convention
- Code organization rules

### `docs/rules/02-domain-model.md`
- Aggregates and value objects
- Session computation rules
- Cross-cell business rules

### `docs/rules/03-database.md`
- Cloud Spanner schema
- Migration conventions
- ReadWriteTransaction vs single Apply guidance
- Dashboard query patterns
- Index usage guidance

### `docs/rules/04-api-and-webhook.md`
- HTTP status code mapping
- Webhook security
- API endpoint inventory
- Controller responsibilities

### `docs/rules/05-testing.md`
- Testing strategy by layer
- Mock repository conventions
- Test file naming

### `docs/rules/06-development.md`
- Error handling
- Configuration
- Performance guidelines
- Common pitfalls

### `apps/api/internal/<cell>/AGENTS.md`
- Cell purpose
- Aggregate(s) owned by the cell
- File inventory and responsibilities
- Inbound/outbound dependencies
- API endpoints provided by the cell
- Cell-specific business rules
- Link to `docs/rules/` for shared conventions

### `apps/api/internal/shared/AGENTS.md`
- Shared utilities purpose
- Config, errors, middleware, SQL helpers
- When to add code here vs in a cell

### `deployments/AGENTS.md`
- How to build and run locally
- Docker Compose services overview
- Migration commands
- Environment variables from `.env.example`

## Out of Scope

- No new features
- No new tests (existing tests move and are renamed with their packages)
- No changes to business logic
- No changes to `README.md` content except path updates

## Acceptance Criteria

- [ ] Root `AGENTS.md` is under 80 lines and contains only overview, links, and quick-start guidance
- [ ] Each section from the original `AGENTS.md` has a clear new home
- [ ] `docs/rules/` contains exactly the six concern files listed above
- [ ] Each cell under `apps/api/internal/` has an `AGENTS.md`
- [ ] `apps/api/internal/shared/` has an `AGENTS.md`
- [ ] `deployments/AGENTS.md` exists
- [ ] `apps/api/` contains the Go module (`go.mod`, `go.sum`) and builds successfully
- [ ] `apps/web/` contains `templates/` and `static/`
- [ ] `apps/infra/nginx/` contains nginx config and Dockerfile
- [ ] `deployments/` contains docker-compose, api Dockerfile, migrations, Makefile, `.env.example`
- [ ] `STATUS.md` lives at `docs/superpowers/plans/STATUS.md`
- [ ] No duplicated content across rulebooks; cross-cutting rules live in `docs/rules/` and cells link to them
- [ ] All internal links between documents use relative paths
- [ ] `docker compose up` from `deployments/` works end-to-end
- [ ] `go test ./...` from `apps/api/` passes
- [ ] No file or type uses the `Handler` suffix; all interface-layer files are `*_controller.go`
- [ ] `cmd/server/main.go` instantiates `*Controller` types and calls `New*Controller` constructors
