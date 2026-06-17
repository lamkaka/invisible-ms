# Architecture Principles

## 1. Domain-Driven Design (DDD)

- **Bounded contexts** are implemented as independent cells: `company`, `staff`, `activity`, `dashboard`
- **Aggregates** are the consistency boundaries: `Company`, `Staff`, `ActivityLog`
- **Value objects** are immutable and have no identity: `Role` (name + hourly rate)
- **Domain events** capture significant occurrences: `StaffCheckedIn`, `StaffCheckedOut`

## 2. Clean Architecture Layers

Each cell follows clean architecture with strict dependency rules:

```
Controller → Service → Domain ← Repository
```

- **Domain layer** (`*_domain.go`): ALL business logic, validation, rules, computations
- **Service layer** (`*_service.go`): Thin orchestration only — coordinates repositories and domain objects
- **Repository layer** (`*_repository.go`): Port interfaces + Spanner adapters
- **Controller layer** (`*_controller.go`): HTTP request/response handling, no business logic

**Critical rule:** Business logic MUST live in the domain layer. Services only orchestrate.

## 3. Cell-Based Architecture

Each bounded context is a self-contained cell with its own:
- Domain models
- Use cases (services)
- Repositories
- HTTP controllers

Cells communicate through port interfaces, not by sharing internal state.

**Cell dependency graph:**
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
| `_controller.go` | Interface layer | HTTP controllers — parse requests, call services, return responses |

**Examples:**
- `company_domain.go` — Company entity, Role value object, validation rules
- `activity_webhook_service.go` — Orchestration for webhook processing
- `dashboard_api_controller.go` — REST API endpoints for dashboard stats

## Code Organization

- All application code lives under `apps/api/cmd/` and `apps/api/internal/`
- Each cell is self-contained with clear boundaries
- Shared utilities (config, errors, middleware) live in `apps/api/internal/shared/`

## Dependency Injection

- `apps/api/cmd/server/main.go` wires all dependencies
- Repositories are instantiated with Spanner client
- Services are instantiated with repository interfaces
- Controllers are instantiated with service interfaces
