# Domain Model Conventions

## Aggregate and Value Object Conventions

- Every aggregate root has a stable primary key. Use UUID (`STRING(36)`) for synthetic keys unless a natural tenant identifier exists.
- Value objects are immutable and have no identity; they are identified by their property values.
- Aggregates are the consistency boundary for business rules and persistence.

## Session Computation

A work session pairs the most recent `CHECK_IN` with the next `CHECK_OUT` for the same staff member and role:

- `Duration = CHECK_OUT timestamp - CHECK_IN timestamp`
- `Cost = duration in hours × role hourly_rate`

Session computation may happen in domain code or in SQL, depending on the read/write context.

## Cross-Cell Business Rules

- Workers are identified by `phone_number + company_code`.
- Check-in and check-out messages flow through the external WhatsApp gateway to `POST /webhook/message`.
- Role validation must verify the role exists in the company's role catalog before assignment or activity logging.
- Invalid or malformed messages return a parse error; no activity log is created.

## Per-Cell Details

- Company, Role, and CompanyActionType definitions live in [`apps/api/internal/company/AGENTS.md`](../../apps/api/internal/company/AGENTS.md).
- Staff aggregate definition lives in [`apps/api/internal/staff/AGENTS.md`](../../apps/api/internal/staff/AGENTS.md).
- ActivityLog aggregate, message parsing, and check-in/check-out flow live in [`apps/api/internal/activity/AGENTS.md`](../../apps/api/internal/activity/AGENTS.md).
