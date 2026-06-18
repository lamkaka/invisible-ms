# Architecture Principles

## 1. Domain-Driven Design (DDD)

- **Bounded contexts** are implemented as independent cells.
- **Aggregates** are the consistency boundaries within a cell.
- **Value objects** are immutable and have no identity.
- **Domain events** capture significant occurrences in the domain.

## 2. Clean Architecture Layers

Each cell follows clean architecture with strict dependency rules:

```
Controller → Service → Domain ← Repository
```

- **Domain layer** (`*_domain.go`): ALL business logic, validation, rules, computations
- **Service layer** (`*_service.go`): Thin orchestration only — coordinates repositories and domain objects
- **Repository layer** (`*_repository.go`): Port interfaces + database adapters
- **Controller layer** (`*_controller.go`): HTTP request/response handling, no business logic

**Critical rule:** Business logic MUST live in the domain layer. Services only orchestrate.

## 3. Cell-Based Architecture

Each bounded context is a self-contained cell with its own:
- Domain models
- Use cases (services)
- Repositories
- HTTP controllers

Cells communicate through port interfaces, not by sharing internal state.

Each cell is organized around one primary entity. By convention, the cell name and the primary entity name match, so files are named `<entity>_domain.go`, `<entity>_service.go`, etc.

**Cell dependency graph:**
- A cell may depend on other cells
- Dependent cells validate the existence and state of entities in the cells they depend on
- Read-only aggregation cells may depend on multiple cells for query purposes

## File Naming Convention

Files use the pattern `{entity}_{layer}.go`:

| Suffix | Purpose | Contents |
|--------|---------|----------|
| `_domain.go` | Domain layer | Entities, value objects, aggregates, business rules, validation, domain services, computations |
| `_service.go` | Application layer | Use case orchestration — load from repo, call domain methods, persist, return |
| `_repository.go` | Infrastructure layer | Port interface (what the cell needs) + database adapter implementation |
| `_controller.go` | Interface layer | HTTP controllers — parse requests, call services, return responses |

**Examples:**
- `<entity>_domain.go` — aggregate root, value objects, validation rules
- `<entity>_service.go` — orchestration for a use case
- `<entity>_controller.go` — REST API endpoints for the entity

## Code Organization

- All application code lives under `apps/api/cmd/` and `apps/api/internal/`
- Each cell is self-contained with clear boundaries
- Shared utilities (config, errors, middleware) live in `apps/api/internal/shared/`

## Dependency Injection

- `apps/api/cmd/server/main.go` wires all dependencies
- Repositories are instantiated with the database client
- Services are instantiated with repository interfaces
- Controllers are instantiated with service interfaces
