# Decentralize Domain Rules into Cell AGENTS.md Files

**Date:** 2026-06-17  
**Status:** Pending Approval

## Problem Statement

`docs/rules/02-domain-model.md`, `03-database.md`, and `04-api-and-webhook.md` currently describe every domain in one place. An agent working on a single cell must read centralized docs that mix company, staff, activity, and dashboard specifics. This duplication makes the central rulebooks long and the cell guides incomplete.

## Solution

Move domain-specific details from the central rulebooks into the owning cell's `AGENTS.md`. Keep the central rulebooks focused on cross-cutting conventions only.

- **Central rules** describe *how* the project does things.
- **Cell AGENTS.md** describes *what* each cell owns.

## Decision: Central Rules Stay Pure

The central rulebooks will not name aggregates, tables, or endpoints.

### `docs/rules/02-domain-model.md`

- Aggregate/value-object conventions
- Session computation formula
- Cross-cell business rules
- Pointers to cell `AGENTS.md` for per-cell aggregates and rules

### `docs/rules/03-database.md`

- Migration conventions
- `ReadWriteTransaction` vs. single `Apply` guidance
- General query guidance
- Index usage principles
- Pointers to cell `AGENTS.md` for per-table schema

### `docs/rules/04-api-and-webhook.md`

- HTTP status-code mapping
- Webhook security requirements
- Controller layer responsibilities
- Pointers to cell `AGENTS.md` for per-cell endpoints

## Content Mapping

### `apps/api/internal/company/AGENTS.md`

Add:

- Full definitions for `Company`, `Role`, and `CompanyActionType` (from rule 02)
- DDL for `companies`, `company_roles`, and `company_action_types` (from rule 03)
- Canonical endpoint inventory for company/role/action-type routes (already present; verify against rule 04)

### `apps/api/internal/staff/AGENTS.md`

Add:

- Full definition for `Staff` aggregate (from rule 02)
- DDL for `staff` and `staff_roles` (from rule 03)
- Canonical endpoint inventory for staff routes (already present; verify against rule 04)

### `apps/api/internal/activity/AGENTS.md`

Add:

- Full definition for `ActivityLog` aggregate (from rule 02)
- Message parsing rules and check-in/check-out flow (from rule 02)
- DDL for `activity_logs` (from rule 03)
- Canonical endpoint inventory for webhook and activity routes (already present; verify against rule 04)

### `apps/api/internal/dashboard/AGENTS.md`

Add:

- Dashboard query patterns and relevant index guidance (from rule 03)
- Canonical endpoint inventory for dashboard JSON and HTML routes (already present; verify against rule 04)

### `apps/api/internal/shared/AGENTS.md`

No change. This cell has no domain-specific content.

## Execution Sequence

1. **Company cell** — move Company/Role/CompanyActionType definitions and DDL.
2. **Staff cell** — move Staff aggregate definition and DDL.
3. **Activity cell** — move ActivityLog aggregate, message parsing, check-in/check-out flow, and DDL.
4. **Dashboard cell** — move dashboard query patterns and index guidance.
5. **Rewrite central rules** — strip domain specifics from rules 02, 03, and 04; keep conventions only.
6. **Cross-link** — ensure central rules point to cells and cells point to central rules.
7. **Verify** — check links, confirm no domain-specific content remains in central rules, confirm no duplicated cross-cutting rules in cells.

## Acceptance Criteria

- [ ] `docs/rules/02-domain-model.md` contains only conventions and cross-cell rules; no per-aggregate definitions.
- [ ] `docs/rules/03-database.md` contains only conventions and patterns; no per-table DDL.
- [ ] `docs/rules/04-api-and-webhook.md` contains only conventions and security rules; no endpoint inventory.
- [ ] Each relevant cell `AGENTS.md` owns its aggregate definitions, DDL, and endpoint inventory.
- [ ] No duplicated cross-cutting rules in cell `AGENTS.md` files.
- [ ] All internal links between central rules and cell `AGENTS.md` files are valid relative paths.
- [ ] Root `AGENTS.md` quick-start mapping points to cell `AGENTS.md` for domain-specific tasks.

## Related Documents

- [docs/rules/02-domain-model.md](../../../../docs/rules/02-domain-model.md)
- [docs/rules/03-database.md](../../../../docs/rules/03-database.md)
- [docs/rules/04-api-and-webhook.md](../../../../docs/rules/04-api-and-webhook.md)
- [apps/api/internal/company/AGENTS.md](../../../../apps/api/internal/company/AGENTS.md)
- [apps/api/internal/staff/AGENTS.md](../../../../apps/api/internal/staff/AGENTS.md)
- [apps/api/internal/activity/AGENTS.md](../../../../apps/api/internal/activity/AGENTS.md)
- [apps/api/internal/dashboard/AGENTS.md](../../../../apps/api/internal/dashboard/AGENTS.md)
- [2026-06-17-agentic-rulebook-refactor-design.md](./2026-06-17-agentic-rulebook-refactor-design.md)
