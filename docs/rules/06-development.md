# Development Guidelines

## Error Handling

- Domain errors are defined in each cell's `*_domain.go` or in `apps/api/internal/shared/errors.go`
- Services return domain errors; controllers translate to HTTP status codes
- Use Go 1.13+ error wrapping with `%w` for context
- **HTTP status code mapping:**
  - `shared.ErrNotFound` → 404 Not Found
  - `shared.ErrAlreadyExists` → 409 Conflict
  - `shared.ErrInvalidInput` → 400 Bad Request
  - `shared.ErrUnauthorized` → 401 Unauthorized
  - Internal/DB errors → 500 Internal Server Error

### Shared Error Types (`apps/api/internal/shared/errors.go`)
- `ErrNotFound` — resource not found
- `ErrAlreadyExists` — resource already exists
- `ErrInvalidInput` — invalid input provided
- `ErrUnauthorized` — unauthorized access
- `DomainError` — structured domain error with code, message, and wrapped error
- Helper functions: `IsNotFound()`, `IsAlreadyExists()`, `IsInvalidInput()`

## Configuration

- Environment variables for all config (Spanner project/instance/database, port, webhook secret, etc.)
- `.env.example` documents required variables
- `apps/api/internal/shared/config.go` loads and validates config
- `shared.GetEnv(key, defaultValue)` helper for safe env var reading

### Config fields:
| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `SPANNER_PROJECT_ID` | — | GCP project ID |
| `SPANNER_INSTANCE_ID` | — | Spanner instance ID |
| `SPANNER_DATABASE_ID` | — | Spanner database ID |
| `WEBHOOK_SECRET` | — | Secret for webhook auth |
| `CORS_ALLOWED_ORIGINS` | `*` | CORS origins |
| `TEMPLATES_PATH` | `../web/templates` | HTML templates dir |
| `STATIC_PATH` | `../web/static` | Static files dir |

## Performance Guidelines

- **Use SQL aggregations** instead of loading all records into memory
- **Avoid N+1 queries** — use JOINs and `IN UNNEST` when fetching related data
- **Parse templates once** at startup, not per-request
- **Use indexes** for frequently queried fields (see database schema)
- **Batch operations** when possible (e.g., insert multiple roles in one transaction)

## Common Pitfalls to Avoid

1. **Don't update parent without children** — Always use transactions for multi-table updates
2. **Don't skip role validation** — Always validate roles exist in company catalog
3. **Don't load all logs into memory** — Use SQL aggregations for dashboard stats
4. **Don't parse templates per-request** — Parse once at startup
5. **Don't ignore error types** — Map domain errors to appropriate HTTP status codes
6. **Don't allow concurrent check-outs** — Use atomic transactions for check-out validation
