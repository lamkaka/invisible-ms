# Database Conventions

## Migration Conventions

- Migration files live in `apps/api/migrations/` with numeric prefix ordering.
- Each `.sql` file may contain DDL and DML.
- DDL is applied before DML.
- Existing DDL objects are skipped with a warning.

## Spanner Transaction Patterns

Use `ReadWriteTransaction` for:

- Multi-table operations
- Operations that must be atomic
- Updates that modify related entities

Use single `Apply` for:

- Single-table operations
- Read-only operations
- Simple inserts with no related entities

## Query Guidance

- Push aggregations (`SUM`, `COUNT`, `AVG`) to Spanner SQL rather than computing in Go.
- Use `TIMESTAMP_DIFF` for duration calculations.
- Pair sessions with correlated subqueries when computing read-side sessions in SQL.

## Index Usage Principles

- Add indexes to support lookup patterns, not speculative ones.
- Use unique indexes to enforce business constraints (e.g., phone number per company).
- Document the purpose of each index near the query that needs it.

## Per-Cell Schema

- Company, role, and action type schemas live in [`apps/api/internal/company/AGENTS.md`](../../apps/api/internal/company/AGENTS.md).
- Staff schema lives in [`apps/api/internal/staff/AGENTS.md`](../../apps/api/internal/staff/AGENTS.md).
- Activity log schema lives in [`apps/api/internal/activity/AGENTS.md`](../../apps/api/internal/activity/AGENTS.md).
- Dashboard query patterns live in [`apps/api/internal/dashboard/AGENTS.md`](../../apps/api/internal/dashboard/AGENTS.md).

## Design Decisions

- **Interleaved tables**: `company_roles`, `company_action_types`, and `staff_roles` are interleaved in their parent tables for locality and cascade deletes.
- **Denormalized `company_code`** in `staff_roles` enables efficient interleaving and prevents cross-tenant role assignments.
- **SQL aggregations**: Dashboard stats are computed in SQL (not in application memory) to handle large datasets efficiently.
- **Atomic check-out**: Check-out operations use a `ReadWriteTransaction` to verify active check-in and create the log atomically, preventing double-check-out race conditions.
