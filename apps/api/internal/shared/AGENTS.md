# Shared Cell

This guide covers the shared cell only. It provides cross-cutting utilities used by all other cells and contains no business logic. For project-wide conventions and task routing, read the root AGENTS.md.

## Purpose
Shared utilities, error types, configuration, and middleware used by all other cells. This cell has no business logic and no dependencies on other cells.

## Owned Aggregates
None. This cell provides value types and utilities only.

## File Inventory

| File | Responsibility |
|---|---|
| `config.go` | Environment variable loading, Config struct, Spanner client initialization |
| `errors.go` | Shared error sentinels (`ErrNotFound`, `ErrAlreadyExists`, `ErrInvalidInput`, `ErrUnauthorized`), `DomainError` struct, helper functions |
| `middleware.go` | HTTP middleware: `LoggingMiddleware` (request logging), `CORSMiddleware` (CORS headers) |
| `sql.go` | `SplitSQLStatements()` — utility for parsing SQL files by semicolons |

## Inbound Dependencies
None (standalone cell).

## Outbound Dependencies
None (provides shared types used by all cells).

## API Endpoints
None. The middleware is used by the router in `apps/api/cmd/server/main.go`.

## Cell-Specific Business Rules
- Config is loaded once at startup via `shared.LoadConfig()`
- Errors use Go 1.13+ wrapping; check with `errors.Is()` and helper functions

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
