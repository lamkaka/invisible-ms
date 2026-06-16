# IMS - Hourly Staff Management System

Multi-tenant HR application for managing hourly staff (freelancers, contractors, part-time, shift staff). Workers check in and out via WhatsApp using keyword commands. The system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

## Architecture

The application follows Domain-Driven Design (DDD) with Clean Architecture and Cell-Based Architecture. Each bounded context is a self-contained cell with strict dependency rules.

```
                     +---------------+
                     |   Dashboard   |
                     |  (read-only)  |
                     +-------+-------+
                             |
            +----------------+----------------+
            |                |                |
     +------v------+  +-----v------+  +------v------+
     |   Company   |  |   Staff   |  |  Activity   |
     | (standalone)|  | -> company |  | -> staff   |
     +-------------+  +------------+  +-------------+
```

Each cell follows this layer structure:

```
Handler -> Service -> Domain <- Repository
```

- **Domain layer** (`_domain.go`): All business logic, validation, rules, and computations.
- **Service layer** (`_service.go`): Thin orchestration only -- loads from repository, calls domain methods, persists, returns.
- **Repository layer** (`_repository.go`): Port interface + Spanner adapter implementation.
- **Handler layer** (`_handler.go`): HTTP request/response handling, no business logic.

### Cell Dependencies

| Cell | Dependencies |
|------|-------------|
| `company` | Standalone -- no internal dependencies |
| `staff` | Depends on `company` (validates roles exist in company catalog) |
| `activity` | Depends on `staff` (validates staff exists and has role), `company` (keyword resolution) |
| `dashboard` | Depends on all cells (read-only aggregation) |

### File Naming Convention

| Suffix | Purpose |
|--------|---------|
| `_domain.go` | Entities, value objects, business rules, validation |
| `_service.go` | Use case orchestration |
| `_repository.go` | Repository interface + Spanner adapter |
| `_handler.go` | HTTP handlers and route registration |

## Tech Stack

- **Backend:** Go 1.26 with gorilla/mux router
- **Database:** Google Cloud Spanner (emulator for local development)
- **Frontend:** Server-rendered HTML + Alpine.js 3.x (CDN, no build step)
- **Messaging:** External webhook integration (WhatsApp via Waha gateway)
- **Containerization:** Docker Compose with Nginx reverse proxy
- **Testing:** Standard library testing with table-driven tests

## Project Structure

```
ims/
├── cmd/
│   ├── server/main.go             # Entry point, wires all cells
│   ├── migrate/main.go            # Database migration tool (Spanner)
│   └── setup/main.go              # One-time database setup (instance + database + migrations)
├── internal/
│   ├── shared/                    # Shared infrastructure
│   │   ├── config.go              # Environment variable loading, Spanner client
│   │   ├── errors.go              # Domain error types (NotFound, AlreadyExists, InvalidInput)
│   │   └── middleware.go          # HTTP middleware (logging, CORS)
│   ├── company/                   # Company management cell
│   │   ├── company_domain.go      # Company aggregate, Role value object, ActionType configuration
│   │   ├── company_service.go     # CRUD orchestration + action type management
│   │   ├── company_repository.go  # Spanner repository for companies + roles
│   │   ├── company_action_type_repository.go  # Spanner repository for action types
│   │   └── company_handler.go     # REST API handlers for companies, roles, action types
│   ├── staff/                    # Staff management cell
│   │   ├── staff_domain.go       # Staff aggregate, role assignment rules
│   │   ├── staff_service.go      # Orchestration with company role validation
│   │   ├── staff_repository.go   # Spanner repository with atomic transactions
│   │   └── staff_handler.go      # REST API handlers for staff
│   ├── activity/                  # Activity/check-in tracking cell
│   │   ├── activity_domain.go     # ActivityLog aggregate, message parsing, session calculations
│   │   ├── activity_webhook_service.go   # Webhook processing orchestration
│   │   ├── activity_session_service.go   # Session computation and activity queries
│   │   ├── activity_repository.go        # Spanner repository with atomic check-out validation
│   │   └── activity_handler.go    # Webhook endpoint + REST activity query handlers
│   └── dashboard/                 # Dashboard aggregation cell (CQRS read model)
│       ├── dashboard_domain.go    # Stats value objects (DashboardStats, TodayOverview, etc.)
│       ├── dashboard_service.go   # Orchestration: query repos, call domain aggregation
│       ├── dashboard_repository.go       # Read-side Spanner queries with SQL aggregations
│       ├── dashboard_api_handler.go      # JSON API endpoint for dashboard stats
│       └── dashboard_web_handler.go      # Server-rendered HTML pages
├── web/static/
│   ├── css/style.css              # Dashboard styling (1023 lines)
│   └── js/app.js                  # Alpine.js components (dashboard, staff, action types)
├── templates/
│   ├── layout.html                # Base HTML template with navigation
│   ├── dashboard.html             # Dashboard page with real-time stats
│   ├── staff.html               # Staff management page
│   └── actions.html               # Action type configuration page
├── migrations/
│   ├── 001_create_companies.sql           # Companies + company_roles tables
│   ├── 002_create_staff.sql             # Workers + staff_roles tables
│   ├── 003_create_activity_logs.sql       # Activity logs table + indexes
│   └── 004_create_company_action_types.sql # Action types + seed data
├── nginx/
│   ├── nginx.conf                 # Reverse proxy config (static files + API proxy)
│   └── Dockerfile                 # Nginx 1.27-alpine with static assets
├── Dockerfile                     # Multi-stage Go build (server + migrate binaries)
├── docker-compose.yml             # Full local stack (4 services)
├── Makefile                       # Build/run/test/docker targets
├── go.mod                         # Module: github.com/lamkaka/invisible-ms
└── .env.example                   # Required environment variables
```

## Prerequisites

- **Go 1.24+** (for local development without Docker)
- **Docker Desktop 4.0+** (for containerized setup with Spanner emulator)
- **Make** (optional, for Makefile targets)

## Quick Start (Docker)

The fastest way to get running is with Docker Compose, which starts a complete stack with Spanner emulator, database migrations, the Go API server, and Nginx reverse proxy.

```bash
# Clone the repository
git clone <repo-url> ims
cd ims

# Copy environment file
cp .env.example .env

# Start all services
make docker-up

# Follow logs (optional)
make docker-logs
```

The stack starts 4 services in order:

1. **spanner-emulator** -- Google Cloud Spanner emulator (gRPC on :9010, REST on :9020)
2. **migrate** -- Creates Spanner instance and database, applies all migration files, then exits
3. **app** -- Go API server (internal port 8080)
4. **nginx** -- Reverse proxy serving the application on port **8888**

Open **http://localhost:8888** in your browser.

To stop:

```bash
make docker-down
```

### Build Docker Images

```bash
make docker-build
```

This builds both the Go binary image and the Nginx image.

## Local Development (Without Docker)

### 1. Start Spanner Emulator

```bash
docker run -d --name spanner-emulator \
  -p 9010:9010 -p 9020:9020 \
  gcr.io/cloud-spanner-emulator/emulator
```

### 2. Set Environment Variables

```bash
export SPANNER_PROJECT_ID=invisible-ms-local
export SPANNER_INSTANCE_ID=invisible-ms-instance
export SPANNER_DATABASE_ID=invisible-ms-db
export SPANNER_EMULATOR_HOST=localhost:9010
export PORT=8080
export WEBHOOK_SECRET=test-secret
```

### 3. Run Database Migrations

```bash
make migrate
```

This creates the Spanner instance and database (if they don't exist), applies all DDL from `migrations/*.sql`, then executes DML seed statements.

### 4. Start the Server

```bash
make run
```

The server starts on **http://localhost:8080**.

### 5. Seed Test Data (Optional)

```bash
go run ./cmd/setup
```

Creates sample companies, staff, and activity data for development and testing.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `SPANNER_PROJECT_ID` | `invisible-ms-local` | Yes | GCP project or emulator project ID |
| `SPANNER_INSTANCE_ID` | `invisible-ms-instance` | Yes | Spanner instance name |
| `SPANNER_DATABASE_ID` | `invisible-ms-db` | Yes | Spanner database name |
| `SPANNER_EMULATOR_HOST` | (empty) | For emulator | Spanner emulator host:port (e.g., `localhost:9010`) |
| `PORT` | `8080` | No | HTTP server port |
| `WEBHOOK_SECRET` | (empty) | For webhooks | Secret value for webhook authentication header |

## API Endpoints

### Webhook

```
POST /webhook/message
```

Receives WhatsApp check-in/check-out events. Requires `X-Webhook-Secret` header matching the configured `WEBHOOK_SECRET`.

**Request body:**
```json
{
  "phone": "+1234567890",
  "message": "IN CLEANING",
  "company_code": "ACME"
}
```

### Company Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/companies` | List all companies |
| POST | `/api/companies` | Create a company (body: `company_code`, `company_name`) |
| GET | `/api/companies/{code}` | Get company details with roles |
| POST | `/api/companies/{code}/roles` | Add role (body: `role_name`, `hourly_rate`) |
| DELETE | `/api/companies/{code}/roles/{role}` | Remove role |
| GET | `/api/companies/{code}/action-types` | List configured action types |
| POST | `/api/companies/{code}/action-types` | Create custom action type (body: `action_type`, `keyword`) |
| PUT | `/api/companies/{code}/action-types/{action}` | Update action type keyword (body: `keyword`) |
| DELETE | `/api/companies/{code}/action-types/{action}` | Delete custom action type (system types protected) |

### Staff Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/staff` | List staff (query: `company_code`) |
| POST | `/api/staff` | Create staff (body: `phone_number`, `name`, `company_code`, `roles[]`) |
| GET | `/api/staff/{id}` | Get staff details |
| PUT | `/api/staff/{id}` | Update staff (body: `name`, `phone_number`, `is_active`) |
| POST | `/api/staff/{id}/roles` | Assign role (body: `role_name`) |
| DELETE | `/api/staff/{id}/roles/{role}` | Unassign role |

### Activity

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/activities` | List activity logs (query: `staff_id`, `company_code`, `from`, `to`) |
| GET | `/api/activities/sessions` | List computed work sessions (query: `company_code`, `from`, `to`) |

### Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard/stats` | Aggregated dashboard stats in JSON (query: `company_code`) |
| GET | `/dashboard` | HTML dashboard page |
| GET | `/staff` | HTML staff management page |
| GET | `/actions` | HTML action type configuration page |

## WhatsApp Commands

Workers send messages from their phone to a WhatsApp number. The external Waha gateway forwards these as webhook requests to the application.

### Command Format

```
{KEYWORD} [{ROLE}]
```

Keywords are case-insensitive.

### Examples

| Message | Action | Notes |
|---------|--------|-------|
| `IN CLEANING` | Check in for CLEANING role | Role required when staff has multiple roles |
| `IN` | Check in for only assigned role | Works when staff has exactly one role |
| `OUT` | Check out | Ends active session for any role |
| `BREAK` | Start break | Requires custom action type configuration |

### Default System Actions

| Action Type | Keyword | Description |
|-------------|---------|-------------|
| `CHECK_IN` | `IN` | Start a work session |
| `CHECK_OUT` | `OUT` | End a work session |

Companies can define custom action types (e.g., BREAK_START, OVERTIME_START) with their own keywords.

### Business Rules

- Workers are identified by phone number + company code
- Workers must be active and have at least one role assigned
- Check-in validates the role exists in the company catalog
- Check-out atomically validates there is an active check-in (prevents race conditions)
- Session cost = duration (hours) x role's hourly rate

## Database Schema

### Tables

| Table | Description | Key |
|-------|-------------|-----|
| `companies` | Company master data | `company_code` |
| `company_roles` | Role catalog per company (interleaved) | `(company_code, role_name)` |
| `company_action_types` | Configured action types per company (interleaved) | `(company_code, action_type)` |
| `staff` | Staff master data | `staff_id` |
| `staff_roles` | Role assignments (interleaved) | `(staff_id, role_name)` |
| `activity_logs` | Check-in/check-out events | `log_id` |

### Indexes

| Index | Columns | Purpose |
|-------|---------|---------|
| `staff_by_company` | `(company_code)` | Query staff by company |
| `staff_by_phone` | `(company_code, phone_number)` UNIQUE | Phone lookup per tenant |
| `activity_logs_by_staff` | `(staff_id, timestamp)` | Staff activity history |
| `activity_logs_by_company` | `(company_code, timestamp)` | Company activity timeline |
| `activity_logs_by_action` | `(company_code, action_type, timestamp)` | Action-type analytics |
| `company_action_types_by_keyword` | `(company_code, keyword)` UNIQUE | Keyword lookup for message parsing |

### Key Design Decisions

- **Interleaved tables**: `company_roles`, `company_action_types`, and `staff_roles` are interleaved in their parent tables for locality and cascade deletes.
- **Denormalized `company_code`** in `staff_roles` enables efficient interleaving and prevents cross-tenant role assignments.
- **SQL aggregations**: Dashboard stats are computed in SQL (not in application memory) to handle large datasets efficiently.
- **Atomic check-out**: Check-out operations use a `ReadWriteTransaction` to verify active check-in and create the log atomically, preventing double-check-out race conditions.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile server binary to `bin/server` |
| `make run` | Run server locally with `go run` |
| `make test` | Run all tests |
| `make migrate` | Run database migrations locally |
| `make docker-build` | Build Docker images |
| `make docker-up` | Start full Docker stack |
| `make docker-down` | Stop all services |
| `make docker-logs` | Follow container logs |
| `make docker-restart` | Down then up |

## Testing

```bash
make test
```

Tests are organized by layer:

| Layer | Test Files | Count |
|-------|-----------|-------|
| Company domain | `company_domain_test.go` | 6 tests |
| Company service | `company_service_test.go` | 3 tests |
| Staff domain | `staff_domain_test.go` | 6 tests |
| Staff service | `staff_service_test.go` | 5 tests |
| Activity domain | `activity_domain_test.go` | 4 tests |
| Activity service | `activity_service_test.go` | 3 tests |
| Dashboard service | `dashboard_service_test.go` | 1 test |

- **Domain tests**: Pure unit tests with no external dependencies.
- **Service tests**: Use mock repositories to verify orchestration logic.
- **Repository tests**: Integration tests against Spanner emulator (not yet implemented).
- **Handler tests**: HTTP tests with mock services (not yet implemented).

## Project Status

MVP complete -- production ready for initial deployment.

### Completed Features

- Company CRUD with role catalog management
- Staff CRUD with role assignment validation (against company catalog)
- WhatsApp webhook integration for check-in/check-out
- Activity log tracking with configurable action types
- Work session computation (pairing CHECK_IN with next CHECK_OUT)
- Cost calculation (duration x hourly rate)
- Dashboard with real-time stats (active staff, hours, costs)
- Action type configuration UI (custom keywords per company)
- Staff management UI
- Webhook authentication (X-Webhook-Secret)
- Atomic operations with race condition prevention
- SQL-based aggregations for performance
- Template caching (parsed once at startup)
- Graceful shutdown

### Known Limitations

- No authentication for dashboard/API access (MVP decision -- add authentication layer before production deployment)
- Overtime alert thresholds not yet implemented
- Only CHECK_IN/CHECK_OUT action types have built-in business logic (custom action types are stored but not processed)
- No CSV/PDF export
- No email notifications
- Handler and repository integration tests pending
- List operations have N+1 query patterns (optimization opportunity)

## Error Handling

Domain errors are defined in `internal/shared/errors.go` and map to HTTP status codes:

| Error | HTTP Status |
|-------|-------------|
| `ErrNotFound` | 404 Not Found |
| `ErrAlreadyExists` | 409 Conflict |
| `ErrInvalidInput` | 400 Bad Request |
| `ErrUnauthorized` | 401 Unauthorized |
| Internal/DB errors | 500 Internal Server Error |
