# Decentralize Domain Rules into Cell AGENTS.md Files — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move domain-specific content from `docs/rules/02-domain-model.md`, `03-database.md`, and `04-api-and-webhook.md` into the owning cell `AGENTS.md` files, leaving the central rulebooks as cross-cutting conventions only.

**Architecture:** Documentation-only restructure. Central rules describe *how* (conventions); cell `AGENTS.md` files describe *what* each cell owns (aggregates, tables, endpoints, rules). Cells link to central rules; central rules link to cells.

**Tech Stack:** Markdown documentation. No code or test changes.

---

## File Structure

| File | Current Responsibility | New Responsibility |
|---|---|---|
| `docs/rules/02-domain-model.md` | All aggregates, value objects, session computation, cross-cell rules | Cross-cutting domain conventions only |
| `docs/rules/03-database.md` | All table DDL, migration/transaction patterns, index guidance | Cross-cutting database conventions only |
| `docs/rules/04-api-and-webhook.md` | All endpoints, status codes, webhook security, controller rules | Cross-cutting API/webhook conventions only |
| `apps/api/internal/company/AGENTS.md` | Cell purpose, files, endpoints, business rules | Adds Company/Role/CompanyActionType definitions and DDL |
| `apps/api/internal/staff/AGENTS.md` | Cell purpose, files, endpoints, business rules | Adds Staff aggregate definition and DDL |
| `apps/api/internal/activity/AGENTS.md` | Cell purpose, files, endpoints, business rules | Adds ActivityLog definition, message parsing, flow, DDL |
| `apps/api/internal/dashboard/AGENTS.md` | Cell purpose, files, endpoints, business rules | Adds dashboard query patterns and index guidance |
| `AGENTS.md` | Landing page | Update quick-start mapping to point to cells for domain specifics |

---

### Task 1: Move Company domain details into `company/AGENTS.md`

**Files:**
- Read: `docs/rules/02-domain-model.md`, `docs/rules/03-database.md`, `docs/rules/04-api-and-webhook.md`
- Modify: `apps/api/internal/company/AGENTS.md`

- [ ] **Step 1: Read source docs**

  Read the three central rule files and the current `company/AGENTS.md`.

- [ ] **Step 2: Add aggregate/value object definitions**

  In `apps/api/internal/company/AGENTS.md`, under `## Owned Aggregates`, replace the brief bullets with the full definitions from `docs/rules/02-domain-model.md` lines 5–35:
  - `Company` aggregate root with `company_code`, `company_name`, `roles`
  - `Role` value object with `name`, `hourly_rate`
  - `CompanyActionType` value object with `action_type`, `keyword`, `is_system`

- [ ] **Step 3: Add database schema section**

  Add a new `## Database Schema` section before `## File Inventory` and copy the DDL from `docs/rules/03-database.md` lines 5–35:
  - `companies` table
  - `company_roles` table with interleave
  - `company_action_types` table with interleave and unique index

- [ ] **Step 4: Verify endpoint inventory**

  Confirm the existing endpoint table in `company/AGENTS.md` matches `docs/rules/04-api-and-webhook.md` lines 37–46. No additions expected; ensure no mismatch.

- [ ] **Step 5: Save and report**

  Save the file. Report the exact lines changed.

---

### Task 2: Move Staff domain details into `staff/AGENTS.md`

**Files:**
- Read: `docs/rules/02-domain-model.md`, `docs/rules/03-database.md`, `docs/rules/04-api-and-webhook.md`
- Modify: `apps/api/internal/staff/AGENTS.md`

- [ ] **Step 1: Read source docs**

  Read the three central rule files and the current `staff/AGENTS.md`.

- [ ] **Step 2: Add Staff aggregate definition**

  In `apps/api/internal/staff/AGENTS.md`, under `## Owned Aggregates`, replace the brief bullet with the full definition from `docs/rules/02-domain-model.md` lines 14–20:
  - `Staff` aggregate root with `staff_id`, `phone_number`, `name`, `company_code`, `assigned_roles`, `is_active`

- [ ] **Step 3: Add database schema section**

  Add a new `## Database Schema` section before `## File Inventory` and copy the DDL from `docs/rules/03-database.md` lines 37–59:
  - `staff` table
  - `staff_roles` table with interleave
  - Indexes `staff_by_company` and `staff_by_phone`

- [ ] **Step 4: Verify endpoint inventory**

  Confirm the existing endpoint table matches `docs/rules/04-api-and-webhook.md` lines 48–53.

- [ ] **Step 5: Save and report**

  Save the file. Report the exact lines changed.

---

### Task 3: Move Activity domain details into `activity/AGENTS.md`

**Files:**
- Read: `docs/rules/02-domain-model.md`, `docs/rules/03-database.md`, `docs/rules/04-api-and-webhook.md`
- Modify: `apps/api/internal/activity/AGENTS.md`

- [ ] **Step 1: Read source docs**

  Read the three central rule files and the current `activity/AGENTS.md`.

- [ ] **Step 2: Add ActivityLog aggregate definition**

  In `apps/api/internal/activity/AGENTS.md`, under `## Owned Aggregates`, replace the brief bullet with the full definition from `docs/rules/02-domain-model.md` lines 22–29:
  - `ActivityLog` aggregate root with `log_id`, `staff_id`, `company_code`, `role`, `action_type`, `timestamp`, `metadata`

- [ ] **Step 3: Add check-in/check-out flow and message parsing**

  Add a new `## Message Processing Flow` section before `## Cell-Specific Business Rules` and copy from `docs/rules/02-domain-model.md` lines 51–71:
  - WhatsApp message flow
  - Message parsing rules
  - Role inference rules

- [ ] **Step 4: Add database schema section**

  Add a new `## Database Schema` section before `## File Inventory` and copy the DDL from `docs/rules/03-database.md` lines 61–76:
  - `activity_logs` table
  - Indexes `activity_logs_by_staff`, `activity_logs_by_company`, `activity_logs_by_action`

- [ ] **Step 5: Verify endpoint inventory**

  Confirm the existing endpoint table matches `docs/rules/04-api-and-webhook.md` lines 34 and 55–57.

- [ ] **Step 6: Save and report**

  Save the file. Report the exact lines changed.

---

### Task 4: Move Dashboard query patterns into `dashboard/AGENTS.md`

**Files:**
- Read: `docs/rules/03-database.md`, `docs/rules/04-api-and-webhook.md`
- Modify: `apps/api/internal/dashboard/AGENTS.md`

- [ ] **Step 1: Read source docs**

  Read `docs/rules/03-database.md`, `docs/rules/04-api-and-webhook.md`, and the current `dashboard/AGENTS.md`.

- [ ] **Step 2: Add query patterns section**

  Add a new `## Query Patterns` section before `## Cell-Specific Business Rules` and copy from `docs/rules/03-database.md` lines 118–124:
  - Session pairing with correlated subqueries
  - Cost calculation via JOIN with `company_roles`
  - Aggregations in SQL
  - Time filtering with `TIMESTAMP_DIFF`

- [ ] **Step 3: Add index usage notes**

  Append the dashboard-relevant indexes from `docs/rules/03-database.md` lines 127–132 to the new `## Query Patterns` section:
  - `activity_logs_by_staff`
  - `activity_logs_by_company`
  - `activity_logs_by_action`

- [ ] **Step 4: Verify endpoint inventory**

  Confirm the existing endpoint table matches `docs/rules/04-api-and-webhook.md` lines 59–65.

- [ ] **Step 5: Save and report**

  Save the file. Report the exact lines changed.

---

### Task 5: Rewrite `docs/rules/02-domain-model.md` to conventions only

**Files:**
- Read: current `docs/rules/02-domain-model.md`
- Modify: `docs/rules/02-domain-model.md`

- [ ] **Step 1: Replace file contents**

  Replace the entire file with:

  ```markdown
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
  ```

- [ ] **Step 2: Save and report**

  Save the file. Report the new line count.

---

### Task 6: Rewrite `docs/rules/03-database.md` to conventions only

**Files:**
- Read: current `docs/rules/03-database.md`
- Modify: `docs/rules/03-database.md`

- [ ] **Step 1: Replace file contents**

  Replace the entire file with:

  ```markdown
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
  ```

- [ ] **Step 2: Save and report**

  Save the file. Report the new line count.

---

### Task 7: Rewrite `docs/rules/04-api-and-webhook.md` to conventions only

**Files:**
- Read: current `docs/rules/04-api-and-webhook.md`
- Modify: `docs/rules/04-api-and-webhook.md`

- [ ] **Step 1: Replace file contents**

  Replace the entire file with:

  ```markdown
  # API & Webhook Conventions

  ## HTTP Status Code Mapping

  | Domain Error | HTTP Status Code |
  |---|---|
  | `shared.ErrNotFound` | 404 Not Found |
  | `shared.ErrAlreadyExists` | 409 Conflict |
  | `shared.ErrInvalidInput` | 400 Bad Request |
  | Internal/DB errors | 500 Internal Server Error |

  ## Webhook Security

  - All webhooks require the `X-Webhook-Secret` header.
  - The secret is loaded from the `WEBHOOK_SECRET` environment variable.
  - Validate the secret with constant-time comparison.
  - Return 401 Unauthorized if the secret is missing or invalid.

  ## Controller Responsibilities

  Controllers (`*_controller.go`) handle:

  - Parsing HTTP requests (query params, path variables, JSON body)
  - Calling the appropriate service method
  - Translating domain errors to HTTP status codes
  - Setting response headers
  - Encoding response bodies (JSON or HTML template)

  Controllers must not contain business logic.

  ## Per-Cell Endpoints

  - Company, role, and action type endpoints live in [`apps/api/internal/company/AGENTS.md`](../../apps/api/internal/company/AGENTS.md).
  - Staff endpoints live in [`apps/api/internal/staff/AGENTS.md`](../../apps/api/internal/staff/AGENTS.md).
  - Webhook and activity endpoints live in [`apps/api/internal/activity/AGENTS.md`](../../apps/api/internal/activity/AGENTS.md).
  - Dashboard endpoints live in [`apps/api/internal/dashboard/AGENTS.md`](../../apps/api/internal/dashboard/AGENTS.md).
  ```

- [ ] **Step 2: Save and report**

  Save the file. Report the new line count.

---

### Task 8: Update root `AGENTS.md` quick-start mapping

**Files:**
- Read: `AGENTS.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Update mapping table**

  In the "If you are editing..." table, change the second column for these rows:
  - `A domain model` → `Cell AGENTS.md`
  - `A controller / HTTP handler` → `Cell AGENTS.md + docs/rules/04-api-and-webhook.md`
  - `A repository / Spanner query` → `Cell AGENTS.md + docs/rules/03-database.md`
  - `A migration file` → `docs/rules/03-database.md + owning Cell AGENTS.md`

  Keep other rows unchanged.

- [ ] **Step 2: Save and report**

  Save the file. Report the changed lines.

---

### Task 9: Verify cross-links and no leftover domain specifics

**Files:**
- Read: all modified files

- [ ] **Step 1: Check central rules**

  Confirm that `docs/rules/02-domain-model.md`, `03-database.md`, and `04-api-and-webhook.md` do not contain any of the following strings:
  - `Company (`
  - `Staff (`
  - `ActivityLog (`
  - `Role (`
  - `CompanyActionType (`
  - `CREATE TABLE`
  - `GET /api/companies`
  - `POST /api/staff`
  - `POST /webhook/message`
  - `GET /dashboard`

- [ ] **Step 2: Check relative links**

  Confirm all links from central rules to cell `AGENTS.md` files and vice versa use valid relative paths and resolve to existing files.

- [ ] **Step 3: Check for duplicate cross-cutting rules**

  Confirm that cell `AGENTS.md` files do not duplicate the HTTP status code mapping, webhook security requirements, migration conventions, or transaction pattern guidance.

- [ ] **Step 4: Report results**

  Report pass/fail for each check. If any check fails, fix the offending file and re-run.

---

## Spec Coverage

| Spec Requirement | Task |
|---|---|
| Rule 02 contains only conventions and cross-cell rules | Task 5 |
| Rule 03 contains only conventions and patterns | Task 6 |
| Rule 04 contains only conventions and security rules | Task 7 |
| Cell AGENTS.md files own aggregate definitions, DDL, endpoints | Tasks 1–4 |
| No duplicated cross-cutting rules in cells | Task 9 step 3 |
| Valid relative links | Task 9 step 2 |
| Root AGENTS.md points to cells for domain specifics | Task 8 |

## Placeholder Scan

No placeholders are used. Every task names exact files, exact source lines, and exact replacement content.
