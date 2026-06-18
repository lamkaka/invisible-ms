# Database Conventions

## Migration Conventions

- Migration files live in `apps/api/migrations/` with numeric prefix ordering.
- Each `.sql` file may contain DDL and DML.
- DDL is applied before DML.
- Existing DDL objects are skipped with a warning.

## Transaction Patterns

Use read-write transactions for:

- Multi-table operations
- Operations that must be atomic
- Updates that modify related entities

Use single operations for:

- Single-table operations
- Read-only operations
- Simple inserts with no related entities

## Query Guidance

- Push aggregations (`SUM`, `COUNT`, `AVG`) to the database rather than computing in application memory.
- Use database-native functions for duration, date, and numeric calculations.
- Pair related events or records with correlated subqueries when computing read-side views in SQL.

## Index Usage Principles

- Add indexes to support lookup patterns, not speculative ones.
- Use unique indexes to enforce business constraints.
- Document the purpose of each index near the query that needs it.

## Per-Cell Schema

- Each cell's concrete schema, table relationships, and query patterns live in the cell's `AGENTS.md`.

## Design Decisions

- **Interleaved tables**: Use interleaving for parent-child relationships that require locality and cascade deletes.
- **Denormalized tenant keys**: Duplicate tenant identifiers where they enable efficient querying and prevent cross-tenant references.
- **SQL aggregations**: Compute aggregate statistics in SQL rather than in application memory to handle large datasets efficiently.
- **Atomic state transitions**: Operations that validate and mutate related state use read-write transactions to prevent race conditions.
