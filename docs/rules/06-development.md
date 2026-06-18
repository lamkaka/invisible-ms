# Development Guidelines

## Error Handling

- Domain errors are defined in each cell's domain layer or in shared error definitions.
- Services return domain errors; controllers translate to HTTP status codes.
- Use Go 1.13+ error wrapping with `%w` for context.

### HTTP Status Code Mapping
- `shared.ErrNotFound` → 404 Not Found
- `shared.ErrAlreadyExists` → 409 Conflict
- `shared.ErrInvalidInput` → 400 Bad Request
- `shared.ErrUnauthorized` → 401 Unauthorized
- Internal/DB errors → 500 Internal Server Error

### Shared Error Types
- `ErrNotFound` — resource not found
- `ErrAlreadyExists` — resource already exists
- `ErrInvalidInput` — invalid input provided
- `ErrUnauthorized` — unauthorized access
- Helper functions to check error types

## Configuration

- Environment variables for all config.
- `.env.example` documents required variables.
- A central config loader validates required values at startup.
- Use safe env var helpers for optional values with defaults.

## Performance Guidelines

- **Use SQL aggregations** instead of loading all records into memory.
- **Avoid N+1 queries** — use JOINs and `IN UNNEST` when fetching related data.
- **Parse templates once** at startup, not per-request.
- **Use indexes** for frequently queried fields.
- **Batch operations** when possible.

## Common Pitfalls to Avoid

1. **Don't update related entities without a transaction** — Always use transactions for multi-table updates.
2. **Don't skip dependency validation** — Always validate referenced entities exist in their owning cell.
3. **Don't load large datasets into memory** — Use database aggregations and pagination.
4. **Don't parse templates per-request** — Parse once at startup.
5. **Don't ignore error types** — Map domain errors to appropriate HTTP status codes.
6. **Don't allow concurrent state transitions** — Use atomic transactions for validate-then-mutate operations.
