# IMS - Hourly Staff Management System

## Overview

IMS is a multi-tenant HR application for managing hourly staff (freelancers, contractors, part-time, shift staff). Workers check in/out via WhatsApp using keyword commands. The system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

## Tech Stack

- **Backend:** Go (Golang)
- **Database:** Google Cloud Spanner
- **Frontend:** Server-rendered HTML + Alpine.js (CDN, no build step)
- **Messaging:** External webhook integration (WhatsApp/Waha layer is external)
- **Architecture:** Domain-Driven Design (DDD) + Clean Architecture + Cell-Based Architecture

## Architecture Principles

### 1. Domain-Driven Design (DDD)

- **Bounded contexts** are implemented as independent cells: `company`, `staff`, `activity`, `dashboard`
- **Aggregates** are the consistency boundaries: `Company`, `Staff`, `ActivityLog`
- **Value objects** are immutable and have no identity: `Role` (name + hourly rate)
- **Domain events** capture significant occurrences: `StaffCheckedIn`, `StaffCheckedOut`

### 2. Clean Architecture Layers

Each cell follows clean architecture with strict dependency rules:

```
Handler → Service → Domain ← Repository
```

- **Domain layer** (`*_domain.go`): ALL business logic, validation, rules, computations
- **Service layer** (`*_service.go`): Thin orchestration only — coordinates repositories and domain objects
- **Repository layer** (`*_repository.go`): Port interfaces + Spanner adapters
- **Handler layer** (`*_handler.go`): HTTP request/response handling, no business logic

**Critical rule:** Business logic MUST live in the domain layer. Services only orchestrate.

### 3. Cell-Based Architecture

Each bounded context is a self-contained cell with its own:
- Domain models
- Use cases (services)
- Repositories
- HTTP handlers

Cells communicate through port interfaces, not by sharing internal state.

**Cell dependencies:**
- `company` → standalone
- `staff` → depends on `company` (validates company code and roles exist)
- `activity` → depends on `staff` (validates staff exists and has the role)
- `dashboard` → depends on `activity`, `staff`, `company` (read-only aggregation)

## File Naming Convention

Files use the pattern `{entity}_{role}.go`:

| Suffix | Purpose | Contents |
|--------|---------|----------|
| `_domain.go` | Domain layer | Entities, value objects, aggregates, business rules, validation, domain services, computations |
| `_service.go` | Application layer | Use case orchestration — load from repo, call domain methods, persist, return |
| `_repository.go` | Infrastructure layer | Port interface (what the cell needs) + Spanner adapter implementation |
| `_handler.go` | Interface layer | HTTP handlers — parse requests, call services, return responses |

**Examples:**
- `company_domain.go` — Company entity, Role value object, validation rules
- `activity_webhook_service.go` — Orchestration for webhook processing
- `dashboard_api_handler.go` — REST API endpoints for dashboard stats

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
│   ├── staff/
│   │   ├── staff_domain.go         # Staff aggregate, role assignment rules, validation
│   │   ├── staff_service.go        # Orchestration
│   │   ├── staff_repository.go     # Port interface + Spanner adapter
│   │   └── staff_handler.go        # REST endpoints for staff management
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
│   └── staff.html                 # Staff management page
│
├── migrations/
│   ├── 001_create_companies.sql
│   ├── 002_create_staff.sql
│   └── 003_create_activity_logs.sql
│
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

## Domain Model

### Company (Aggregate Root)
- `company_code` (string, unique) — tenant identifier
- `company_name` (string)
- `roles` (collection of Role value objects)

### Role (Value Object)
- `name` (string) — e.g., "CLEANING", "DELIVERY"
- `hourly_rate` (decimal) — cost per hour for this role

### Staff (Aggregate Root)
- `staff_id` (string, UUID)
- `phone_number` (string) — unique within company
- `name` (string)
- `company_code` (string) — FK to Company
- `assigned_roles` ([]string) — list of role names from company's catalog
- `is_active` (bool)

### ActivityLog (Aggregate Root)
- `log_id` (string, UUID)
- `staff_id` (string)
- `company_code` (string)
- `role` (string) — the role being worked
- `action_type` (enum) — CHECK_IN, CHECK_OUT, BREAK_START, BREAK_END, OVERTIME_START, etc.
- `timestamp` (timestamp)
- `metadata` (JSON, optional) — extra context for future action types

**Session computation:** A "work session" is derived by pairing the most recent CHECK_IN with the next CHECK_OUT for the same staff + role. Duration and cost are computed from the pair.

## Business Rules

### Staff Identification
- Workers are identified by `phone_number` + `company_code`
- The webhook payload includes both fields

### Check-in/Check-out Flow
1. Staff sends WhatsApp message (e.g., "IN CLEANING" or "OUT")
2. External gateway (Waha) sends webhook to `POST /webhook/message` with `{ phone, message, company_code }`
3. App parses the message:
   - Extracts action (IN/OUT) and optional role
   - If staff has only one role, "IN" is sufficient
   - If staff has multiple roles, role must be specified (e.g., "IN CLEANING")
4. App validates:
   - Staff exists and is active
   - Role is assigned to the staff
   - For CHECK_OUT: staff has an active CHECK_IN for this role
5. App creates an `ActivityLog` record with the appropriate action type
6. App responds with confirmation (optional, via webhook response)

### Message Parsing Rules
- Keywords are case-insensitive
- Format: `{ACTION} [ROLE]`
- Valid actions: `IN`, `OUT` (extensible for BREAK, OVERTIME, etc.)
- Role is optional if staff has only one assigned role
- Invalid messages return an error response

### Cost Calculation
- Duration = CHECK_OUT timestamp - CHECK_IN timestamp
- Cost = duration (in hours) × role's hourly rate
- Computed on-the-fly or cached in read model

## Dashboard Requirements

The management dashboard displays:

### Today's Overview
- Who's currently working (active sessions)
- Who checked in/out today
- Total hours logged today

### Cost Tracking
- Total labor cost: today, this week, this month
- Breakdown by company, role, or staff

### Staff Activity
- Most active staff
- Average hours per staff
- Overtime alerts (configurable threshold)

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

### Staff Management
- `GET /api/staff` — list staff (filterable by company)
- `POST /api/staff` — create staff
- `GET /api/staff/:id` — get staff details
- `PUT /api/staff/:id` — update staff
- `POST /api/staff/:id/roles` — assign role to staff
- `DELETE /api/staff/:id/roles/:role` — unassign role from staff

### Activity
- `GET /api/activities` — list activity logs (filterable by staff, company, date range)
- `GET /api/activities/sessions` — list computed work sessions

### Dashboard
- `GET /api/dashboard/stats` — aggregated stats for dashboard
- `GET /dashboard` — HTML dashboard page
- `GET /staff` — HTML staff management page

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
CREATE TABLE staff (
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT TRUE,
) PRIMARY KEY (staff_id);

CREATE INDEX staff_by_company ON staff(company_code);
CREATE UNIQUE INDEX staff_by_phone ON staff(company_code, phone_number);
```

### Staff Roles Table
```sql
CREATE TABLE staff_roles (
  staff_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL,  -- denormalized for interleaving
) PRIMARY KEY (staff_id, role_name),
  INTERLEAVE IN PARENT staff ON DELETE CASCADE;
```

### Activity Logs Table
```sql
CREATE TABLE activity_logs (
  log_id STRING(36) NOT NULL,
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  role STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  metadata JSON,
) PRIMARY KEY (log_id);

CREATE INDEX activity_logs_by_staff ON activity_logs(staff_id, timestamp);
CREATE INDEX activity_logs_by_company ON activity_logs(company_code, timestamp);
CREATE INDEX activity_logs_by_action ON activity_logs(company_code, action_type, timestamp);
```

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
- **HTTP status code mapping:**
  - `shared.ErrNotFound` → 404 Not Found
  - `shared.ErrAlreadyExists` → 409 Conflict
  - `shared.ErrInvalidInput` → 400 Bad Request
  - Internal/DB errors → 500 Internal Server Error

### Spanner Transaction Patterns

**Use ReadWriteTransaction for:**
- Multi-table operations (e.g., insert parent + children)
- Operations that must be atomic (e.g., check-out validation)
- Update operations that modify related entities (e.g., staff + roles)

**Example pattern:**
```go
_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    // Delete existing child records
    txn.BufferWrite(spanner.Delete("child_table", ...))
    
    // Update parent
    txn.BufferWrite(spanner.Update("parent_table", ...))
    
    // Insert new child records
    for _, child := range children {
        txn.BufferWrite(spanner.Insert("child_table", ...))
    }
    
    return nil
})
```

**Use single Apply for:**
- Single-table operations
- Read-only operations
- Simple inserts with no related entities

### Webhook Security
- All webhooks require `X-Webhook-Secret` header
- Secret is loaded from `WEBHOOK_SECRET` environment variable
- Handler validates secret before processing
- Returns 401 Unauthorized if secret is missing or invalid

### Role Validation
- Workers can only be assigned roles that exist in the company's `company_roles` table
- `StaffService` depends on `CompanyService` to validate roles
- Validation happens in `CreateStaff` and `AssignRole` methods
- Prevents phantom roles that would break cost calculations

### Testing
- Domain layer: unit tests with no external dependencies
- Service layer: mock repositories
- Repository layer: integration tests against Spanner emulator (skipped for MVP)
- Handler layer: HTTP tests with mock services (not yet implemented)

### Configuration
- Environment variables for all config (Spanner project/instance/database, port, webhook secret, etc.)
- `.env.example` documents required variables
- `internal/shared/config.go` loads and validates config

### Performance Guidelines
- **Use SQL aggregations** instead of loading all records into memory
- **Avoid N+1 queries** - use JOINs when fetching related data
- **Parse templates once** at startup, not per-request
- **Use indexes** for frequently queried fields (see database schema)
- **Batch operations** when possible (e.g., insert multiple roles in one transaction)

### Dashboard Query Patterns
- Session pairing: Use correlated subqueries to pair CHECK_IN with next CHECK_OUT
- Cost calculation: JOIN with `company_roles` to get hourly_rate in same query
- Aggregations: Use `SUM`, `COUNT`, `AVG` in SQL, not in Go code
- Time-based filtering: Use `TIMESTAMP_DIFF` for duration calculations

### Common Pitfalls to Avoid
1. **Don't update parent without children** - Always use transactions for multi-table updates
2. **Don't skip role validation** - Always validate roles exist in company catalog
3. **Don't load all logs into memory** - Use SQL aggregations for dashboard stats
4. **Don't parse templates per-request** - Parse once at startup
5. **Don't ignore error types** - Map domain errors to appropriate HTTP status codes
6. **Don't allow concurrent check-outs** - Use atomic transactions for check-out validation

## MVP Scope

For the initial MVP:
- No authentication (add later)
- WhatsApp integration via external webhook (Waha layer is external)
- Basic dashboard with today's stats
- Company and staff management via REST API
- Check-in/check-out via WhatsApp keywords

## Future Enhancements

- Authentication (Google OAuth or email/password)
- Additional action types (BREAK, OVERTIME, TASK_COMPLETE)
- Export reports (CSV, PDF)
- Email notifications
- Mobile app for managers
- Staff self-service portal (view own hours)
