# Domain Model Conventions

## Aggregate and Value Object Conventions

- Every aggregate root has a stable primary key. Use UUID (`STRING(36)`) for synthetic keys unless a natural tenant identifier exists.
- Value objects are immutable and have no identity; they are identified by their property values.
- Aggregates are the consistency boundary for business rules and persistence.

## Cross-Cell Business Rules

- Entities that span cells are identified by a stable composite key or foreign reference.
- External events or messages enter the system through adapters and route to the owning cell.
- Related value objects and catalogs must exist in their owning cell before another cell can reference them.
- Invalid or malformed input returns a parse error; no state change occurs.

## Per-Cell Details

- Each cell's concrete aggregate, value object, and business rule definitions live in the cell's `AGENTS.md`.
