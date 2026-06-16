# IMS - Hourly Worker Management System Design Spec

**Date:** 2026-06-16  
**Status:** Approved for Implementation

## Problem Statement

Build a mini HR application to manage check-in/check-out of hourly workers (freelancers, contractors, part-time, shift workers) across multiple companies. The system must:
- Track worker activity via WhatsApp keyword commands
- Calculate hourly costs per role
- Provide a management dashboard with high-level stats

## Solution Overview

A multi-tenant Go application that receives webhooks from an external WhatsApp gateway (Waha), parses keyword commands to log worker activities, computes work sessions and costs, and exposes a server-rendered dashboard for management.

## Architecture Decisions

### Tech Stack
- **Backend:** Go (Golang)
- **Database:** Google Cloud Spanner
- **Frontend:** Server-rendered HTML + Alpine.js (CDN, no build step)
- **Messaging:** External webhook integration (WhatsApp/Waha layer is external)

### Architectural Patterns
- **Domain-Driven Design (DDD):** Bounded contexts as aggregates with clear business logic ownership
- **Clean Architecture:** Strict layer separation (Handler → Service → Domain ← Repository)
- **Cell-Based Architecture:** Each bounded context is a self-contained cell with its own domain, use cases, repositories, and handlers

### Why This Architecture?
- **DDD** ensures business logic (validation, rules, computations) lives in the domain layer, not scattered across handlers or services
- **Clean Architecture** enforces dependency rules: handlers call services, services orchestrate domain objects, repositories implement port interfaces
- **Cell-Based** keeps bounded contexts isolated; cells communicate through port interfaces, not shared state

## Domain Model

### Bounded Contexts (Cells)

#### 1. Company Cell
**Aggregate Root:** `Company`
- `company_code` (string, unique) — tenant identifier
- `company_name` (string)
- `roles` (collection of Role value objects)

**Value Object:** `Role`
- `name` (string) — e.g., "CLEANING", "DELIVERY"
- `hourly_rate` (decimal) — cost per hour for this role

**Business Rules:**
- Company code is immutable once created
- Roles can be added/removed, but removing a role that workers are assigned to should be prevented or require reassignment
- Hourly rate must be non-negative

#### 2. Worker Cell
**Aggregate Root:** `Worker`
- `worker_id` (string, UUID)
- `phone_number` (string) — unique within company
- `name` (string)
- `company_code` (string) — FK to Company
- `assigned_roles` ([]string) — list of role names from company's catalog
- `is_active` (bool)

**Business Rules:**
- Worker is identified by `phone_number` + `company_code`
- Phone number must be unique within a company
- Assigned roles must exist in the company's role catalog
- Inactive workers cannot check in

#### 3. Activity Cell
**Aggregate Root:** `ActivityLog`
- `log_id` (string, UUID)
- `worker_id` (string)
- `company_code` (string)
- `role` (string) — the role being worked
- `action_type` (enum) — CHECK_IN, CHECK_OUT, BREAK_START, BREAK_END, OVERTIME_START, etc.
- `timestamp` (timestamp)
- `metadata` (JSON, optional) — extra context for future action types

**Session Computation (Read Side):**
- A "work session" is derived by pairing the most recent CHECK_IN with the next CHECK_OUT for the same worker + role
- Duration = CHECK_OUT timestamp - CHECK_IN timestamp
- Cost = duration (in hours) × role's hourly rate

**Business Rules:**
- Message parsing is case-insensitive
- Format: `{ACTION} [ROLE]`
- Valid actions: `IN`, `OUT` (extensible for BREAK, OVERTIME, etc.)
- Role is optional if worker has only one assigned role; required if multiple
- For CHECK_OUT: worker must have an active CHECK_IN for this role
- Worker must exist and be active
- Role must be assigned to the worker

#### 4. Dashboard Cell (CQRS Read Side)
**Not an aggregate** — this is a read-only query service that aggregates data from activity, worker, and company cells.

**Stats Computed:**
- **Today's Overview:**
  - Who's currently working (active sessions)
  - Who checked in/out today
  - Total hours logged today
- **Cost Tracking:**
  - Total labor cost: today, this week, this month
  - Breakdown by company, role, or worker
- **Worker Activity:**
  - Most active workers
  - Average hours per worker
  - Overtime alerts (configurable threshold)

## Webhook Flow

1. Worker sends WhatsApp message (e.g., "IN CLEANING" or "OUT")
2. External gateway (Waha) sends webhook to `POST /webhook/message` with payload:
   ```json
   {
     "phone": "+1234567890",
     "message": "IN CLEANING",
     "company_code": "ACME"
   }
   ```
3. App parses the message:
   - Extracts action (IN/OUT) and optional role
   - If worker has only one role, "IN" is sufficient
   - If worker has multiple roles, role must be specified (e.g., "IN CLEANING")
4. App validates:
   - Worker exists and is active
   - Role is assigned to the worker
   - For CHECK_OUT: worker has an active CHECK_IN for this role
5. App creates an `ActivityLog` record with the appropriate action type
6. App responds with confirmation (optional, via webhook response)

## API Endpoints

### Webhook
- `POST /webhook/message` — receives `{ phone, message, company_code }`

### Company Management
- `GET /api/companies` — list all companies
- `POST /api/companies` — create company
- `GET /api/companies/:code` — get company details
- `PUT /api/companies/:code` — update company
- `POST /api/companies/:code/roles` — add role to company
- `DELETE /api/companies/:code/roles/:role` — remove role from company

### Worker Management
- `GET /api/workers` — list workers (filterable by company)
- `POST /api/workers` — create worker
- `GET /api/workers/:id` — get worker details
- `PUT /api/workers/:id` — update worker
- `POST /api/workers/:id/roles` — assign role to worker
- `DELETE /api/workers/:id/roles/:role` — unassign role from worker

### Activity
- `GET /api/activities` — list activity logs (filterable by worker, company, date range)
- `GET /api/activities/sessions` — list computed work sessions

### Dashboard
- `GET /api/dashboard/stats` — aggregated stats for dashboard
- `GET /dashboard` — HTML dashboard page
- `GET /workers` — HTML worker management page

## Database Schema (Cloud Spanner)

### Companies Table
```sql
CREATE TABLE companies (
  company_code STRING(50) NOT NULL,
  company_name STRING(200) NOT NULL,
) PRIMARY KEY (company_code);
```

### Company Roles Table
```sql
CREATE TABLE company_roles (
  company_code STRING(50) NOT NULL,
  role_name STRING(50) NOT NULL,
  hourly_rate FLOAT64 NOT NULL,
) PRIMARY KEY (company_code, role_name),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;
```

### Workers Table
```sql
CREATE TABLE workers (
  worker_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT TRUE,
) PRIMARY KEY (worker_id);

CREATE INDEX workers_by_company ON workers(company_code);
CREATE UNIQUE INDEX workers_by_phone ON workers(company_code, phone_number);
```

### Worker Roles Table
```sql
CREATE TABLE worker_roles (
  worker_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL,  -- denormalized for interleaving
) PRIMARY KEY (worker_id, role_name),
  INTERLEAVE IN PARENT workers ON DELETE CASCADE;
```

### Activity Logs Table
```sql
CREATE TABLE activity_logs (
  log_id STRING(36) NOT NULL,
  worker_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  role STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  metadata JSON,
) PRIMARY KEY (log_id);

CREATE INDEX activity_logs_by_worker ON activity_logs(worker_id, timestamp);
CREATE INDEX activity_logs_by_company ON activity_logs(company_code, timestamp);
CREATE INDEX activity_logs_by_action ON activity_logs(company_code, action_type, timestamp);
```

## Project Structure

```
ims/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point, wires all cells
│
├── internal/
│   ├── shared/
│   │   ├── config.go                # Environment variables, Spanner client init
│   │   ├── errors.go                # Shared error types
│   │   └── middleware.go            # HTTP middleware (logging, etc.)
│   │
│   ├── company/
│   │   ├── company_domain.go        # Company aggregate, Role value object, business rules
│   │   ├── company_service.go       # Orchestration: CRUD operations
│   │   ├── company_repository.go    # Port interface + Spanner adapter
│   │   └── company_handler.go       # REST endpoints for company management
│   │
│   ├── worker/
│   │   ├── worker_domain.go         # Worker aggregate, role assignment rules, validation
│   │   ├── worker_service.go        # Orchestration
│   │   ├── worker_repository.go     # Port interface + Spanner adapter
│   │   └── worker_handler.go        # REST endpoints for worker management
│   │
│   ├── activity/
│   │   ├── activity_domain.go       # ActivityLog aggregate, ActionType enum, session pairing logic,
│   │   │                            # duration/cost calculation, message parsing rules, domain services
│   │   ├── activity_webhook_service.go  # Orchestration: parse webhook → call domain → persist
│   │   ├── activity_session_service.go  # Orchestration: query logs → call domain session logic
│   │   ├── activity_repository.go   # Port interface + Spanner adapter
│   │   └── activity_handler.go      # Webhook endpoint + REST endpoints for activity queries
│   │
│   └── dashboard/
│       ├── dashboard_domain.go      # Stats value objects, aggregation rules, computed metrics
│       ├── dashboard_service.go     # Orchestration: query repos → call domain aggregation
│       ├── dashboard_repository.go  # Read-side Spanner queries (CQRS read model)
│       ├── dashboard_api_handler.go # GET /api/dashboard/stats
│       └── dashboard_web_handler.go # HTML dashboard pages
│
├── web/
│   └── static/
│       ├── css/
│       │   └── style.css
│       └── js/
│           └── app.js               # Alpine.js logic
│
├── templates/
│   ├── layout.html                  # Base HTML template
│   ├── dashboard.html               # Dashboard page
│   └── workers.html                 # Worker management page
│
├── migrations/
│   ├── 001_create_companies.sql
│   ├── 002_create_workers.sql
│   └── 003_create_activity_logs.sql
│
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

## File Naming Convention

Files use the pattern `{entity}_{role}.go`:

| Suffix | Purpose | Contents |
|--------|---------|----------|
| `_domain.go` | Domain layer | Entities, value objects, aggregates, business rules, validation, domain services, computations |
| `_service.go` | Application layer | Use case orchestration — load from repo, call domain methods, persist, return |
| `_repository.go` | Infrastructure layer | Port interface (what the cell needs) + Spanner adapter implementation |
| `_handler.go` | Interface layer | HTTP handlers — parse requests, call services, return responses |

**Critical Rule:** ALL business logic MUST live in `_domain.go` files. Services only orchestrate.

## Cell Dependencies

- `company` → standalone
- `worker` → depends on `company` (validates company code and roles exist)
- `activity` → depends on `worker` (validates worker exists and has the role)
- `dashboard` → depends on `activity`, `worker`, `company` (read-only aggregation)

Cells communicate through port interfaces, not by sharing internal state.

## Development Guidelines

### Code Organization
- All application code lives under `internal/` to prevent external imports
- Each cell is self-contained with clear boundaries
- Shared utilities (config, errors, middleware) live in `internal/shared/`

### Dependency Injection
- `cmd/server/main.go` wires all dependencies
- Repositories are instantiated with Spanner client
- Services are instantiated with repository interfaces
- Handlers are instantiated with service interfaces

### Error Handling
- Domain errors are defined in `internal/shared/errors.go`
- Services return domain errors; handlers translate to HTTP status codes
- Use Go 1.13+ error wrapping with `%w` for context

### Testing
- Domain layer: unit tests with no external dependencies
- Service layer: mock repositories
- Repository layer: integration tests against Spanner emulator
- Handler layer: HTTP tests with mock services

### Configuration
- Environment variables for all config (Spanner project/instance/database, port, etc.)
- `.env.example` documents required variables
- `internal/shared/config.go` loads and validates config

## MVP Scope

For the initial MVP:
- No authentication (add later)
- WhatsApp integration via external webhook (Waha layer is external)
- Basic dashboard with today's stats
- Company and worker management via REST API
- Check-in/check-out via WhatsApp keywords

## Future Enhancements

- Authentication (Google OAuth or email/password)
- Additional action types (BREAK, OVERTIME, TASK_COMPLETE)
- Export reports (CSV, PDF)
- Email notifications
- Mobile app for managers
- Worker self-service portal (view own hours)
