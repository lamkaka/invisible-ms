# Refactor Root README into Overview + Canonical Docs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce `README.md` to a concise human-facing overview and getting-started guide, moving architecture rules, API details, WhatsApp behavior, database design decisions, testing strategy, and error handling to `docs/rules/` and cell `AGENTS.md`.

**Architecture:** Documentation-only restructure. README owns "what is this project and how do I run it"; `docs/rules/` own cross-cutting conventions; cell `AGENTS.md` files own domain specifics.

**Tech Stack:** Markdown documentation. No code or test changes.

---

## File Structure

| File | Current Responsibility | New Responsibility |
|---|---|---|
| `README.md` | Mixed overview + rules + endpoints + schema + testing + status | Concise overview, tech stack, quick start, env vars, Makefile, status, links |
| `apps/api/internal/activity/AGENTS.md` | Activity domain, schema, endpoints, message flow, business rules | Adds `## WhatsApp Commands` section from README |
| `docs/rules/03-database.md` | Database conventions | Adds `## Design Decisions` section from README |
| `docs/rules/05-testing.md` | Testing strategy by layer | Adds test counts/layer table from README |
| `docs/rules/06-development.md` | Error handling, config, performance, pitfalls | Adds `ErrUnauthorized` → 401 mapping if missing |

---

### Task 1: Move WhatsApp command details into `activity/AGENTS.md`

**Files:**
- Read: `README.md` lines 304–340, `apps/api/internal/activity/AGENTS.md`
- Modify: `apps/api/internal/activity/AGENTS.md`

- [ ] **Step 1: Read source files**

  Read the WhatsApp section of `README.md` and the current `activity/AGENTS.md`.

- [ ] **Step 2: Add `## WhatsApp Commands` section**

  Insert a new `## WhatsApp Commands` section after `## Message Processing Flow` and before `## Cell-Specific Business Rules`. Copy and adapt the following content from `README.md`:

  ```markdown
  ## WhatsApp Commands

  Workers send messages from their phone to a WhatsApp number. The external Waha gateway forwards these as webhook requests to the application.

  ### Command Format

  ```
  {KEYWORD} [{ROLE}]
  ```

  Keywords are case-insensitive.

  ### Examples

  | Message | Action | Notes |
  |---------|--------|-------|
  | `IN CLEANING` | Check in for CLEANING role | Role required when staff has multiple roles |
  | `IN` | Check in for only assigned role | Works when staff has exactly one role |
  | `OUT` | Check out | Ends active session for any role |
  | `BREAK` | Start break | Requires custom action type configuration |

  ### Default System Actions

  | Action Type | Keyword | Description |
  |-------------|---------|-------------|
  | `CHECK_IN` | `IN` | Start a work session |
  | `CHECK_OUT` | `OUT` | End a work session |

  Companies can define custom action types (e.g., `BREAK_START`, `OVERTIME_START`) with their own keywords.
  ```

- [ ] **Step 3: Merge business rules without duplication**

  The business rules under README's WhatsApp section (lines 336–340) are:
  - Workers are identified by phone number + company code
  - Workers must be active and have at least one role assigned
  - Check-in validates the role exists in the company catalog
  - Check-out atomically validates there is an active check-in
  - Session cost = duration (hours) × role's hourly rate

  Compare these to the existing `## Cell-Specific Business Rules` in `activity/AGENTS.md`. Add any missing bullets. Do not duplicate rules already present.

- [ ] **Step 4: Save, commit, and report**

  Save the file. Commit with message: `docs: move WhatsApp command details from README to activity/AGENTS.md`
  Report the exact lines changed.

---

### Task 2: Move testing details into `docs/rules/05-testing.md`

**Files:**
- Read: `README.md` lines 387–408, `docs/rules/05-testing.md`
- Modify: `docs/rules/05-testing.md`

- [ ] **Step 1: Read source files**

  Read the Testing section of `README.md` and the current `docs/rules/05-testing.md`.

- [ ] **Step 2: Append test counts**

  Append the following content to the end of `docs/rules/05-testing.md`:

  ```markdown
  ## Test Inventory

  | Layer | Test Files | Count |
  |-------|-----------|-------|
  | Company domain | `company_domain_test.go` | 6 tests |
  | Company service | `company_service_test.go` | 3 tests |
  | Staff domain | `staff_domain_test.go` | 6 tests |
  | Staff service | `staff_service_test.go` | 5 tests |
  | Activity domain | `activity_domain_test.go` | 4 tests |
  | Activity service | `activity_service_test.go` | 3 tests |
  | Dashboard service | `dashboard_service_test.go` | 1 test |

  - **Domain tests**: Pure unit tests with no external dependencies.
  - **Service tests**: Use mock repositories to verify orchestration logic.
  - **Repository tests**: Integration tests against Spanner emulator (not yet implemented).
  - **Controller tests**: HTTP tests with mock services (not yet implemented).
  ```

- [ ] **Step 3: Save, commit, and report**

  Save the file. Commit with message: `docs: move testing inventory from README to 05-testing.md`
  Report the exact lines changed.

---

### Task 3: Move database design decisions into `docs/rules/03-database.md`

**Files:**
- Read: `README.md` lines 366–371, `docs/rules/03-database.md`
- Modify: `docs/rules/03-database.md`

- [ ] **Step 1: Read source files**

  Read the "Key Design Decisions" subsection of `README.md` and the current `docs/rules/03-database.md`.

- [ ] **Step 2: Add `## Design Decisions` section**

  Append the following content to the end of `docs/rules/03-database.md`:

  ```markdown
  ## Design Decisions

  - **Interleaved tables**: `company_roles`, `company_action_types`, and `staff_roles` are interleaved in their parent tables for locality and cascade deletes.
  - **Denormalized `company_code`** in `staff_roles` enables efficient interleaving and prevents cross-tenant role assignments.
  - **SQL aggregations**: Dashboard stats are computed in SQL (not in application memory) to handle large datasets efficiently.
  - **Atomic check-out**: Check-out operations use a `ReadWriteTransaction` to verify active check-in and create the log atomically, preventing double-check-out race conditions.
  ```

- [ ] **Step 3: Save, commit, and report**

  Save the file. Commit with message: `docs: move database design decisions from README to 03-database.md`
  Report the exact lines changed.

---

### Task 4: Ensure `ErrUnauthorized` mapping is in `docs/rules/06-development.md`

**Files:**
- Read: `README.md` lines 441–451, `docs/rules/06-development.md`
- Modify: `docs/rules/06-development.md` (if needed)

- [ ] **Step 1: Read source files**

  Read the Error Handling section of `README.md` and the current `docs/rules/06-development.md`.

- [ ] **Step 2: Check for `ErrUnauthorized` HTTP mapping**

  The README maps `ErrUnauthorized` → 401 Unauthorized. Check if `docs/rules/06-development.md` already maps `ErrUnauthorized` to HTTP 401.

  If missing, update the HTTP status code mapping in `docs/rules/06-development.md` to include:
  - `shared.ErrUnauthorized` → 401 Unauthorized

- [ ] **Step 3: Save, commit if changed, and report**

  If you made a change, commit with message: `docs: add ErrUnauthorized mapping to 06-development.md`
  Report whether a change was needed.

---

### Task 5: Rewrite `README.md`

**Files:**
- Read: current `README.md`
- Modify: `README.md`

- [ ] **Step 1: Replace file contents**

  Replace the entire file with:

  ```markdown
  # IMS - Hourly Staff Management System

  Multi-tenant HR application for managing hourly staff (freelancers, contractors, part-time, shift staff). Workers check in and out via WhatsApp using keyword commands. The system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

  ## Tech Stack

  - **Backend:** Go 1.26 with gorilla/mux router
  - **Database:** Google Cloud Spanner (emulator for local development)
  - **Frontend:** Server-rendered HTML + Alpine.js 3.x (CDN, no build step)
  - **Messaging:** External webhook integration (WhatsApp via Waha gateway)
  - **Containerization:** Docker Compose with Nginx reverse proxy
  - **Testing:** Standard library testing with table-driven tests

  ## Quick Start (Docker)

  The fastest way to get running is with Docker Compose, which starts a complete stack with Spanner emulator, database migrations, the Go API server, and Nginx reverse proxy.

  ```bash
  git clone <repo-url> ims
  cd ims
  cp .env.example .env
  make docker-up
  ```

  The stack starts on **http://localhost:8888**. To stop:

  ```bash
  make docker-down
  ```

  ### Build Docker Images

  ```bash
  make docker-build
  ```

  ## Local Development (Without Docker)

  1. Start the Spanner emulator:
     ```bash
     docker run -d --name spanner-emulator \
       -p 9010:9010 -p 9020:9020 \
       gcr.io/cloud-spanner-emulator/emulator
     ```

  2. Set environment variables:
     ```bash
     export SPANNER_PROJECT_ID=invisible-ms-local
     export SPANNER_INSTANCE_ID=invisible-ms-instance
     export SPANNER_DATABASE_ID=invisible-ms-db
     export SPANNER_EMULATOR_HOST=localhost:9010
     export PORT=8080
     export WEBHOOK_SECRET=test-secret
     ```

  3. Run migrations:
     ```bash
     make migrate
     ```

  4. Start the server:
     ```bash
     make run
     ```

  5. Seed test data (optional):
     ```bash
     cd apps/api && go run ./cmd/setup
     ```

  ## Environment Variables

  | Variable | Default | Required | Description |
  |----------|---------|----------|-------------|
  | `SPANNER_PROJECT_ID` | `invisible-ms-local` | Yes | GCP project or emulator project ID |
  | `SPANNER_INSTANCE_ID` | `invisible-ms-instance` | Yes | Spanner instance name |
  | `SPANNER_DATABASE_ID` | `invisible-ms-db` | Yes | Spanner database name |
  | `SPANNER_EMULATOR_HOST` | (empty) | For emulator | Spanner emulator host:port |
  | `PORT` | `8080` | No | HTTP server port |
  | `WEBHOOK_SECRET` | (empty) | For webhooks | Secret value for webhook authentication header |

  ## Makefile Targets

  | Target | Description |
  |--------|-------------|
  | `make build` | Compile server binary to `bin/server` |
  | `make run` | Run server locally with `go run` |
  | `make test` | Run all tests |
  | `make migrate` | Run database migrations locally |
  | `make docker-build` | Build Docker images |
  | `make docker-up` | Start full Docker stack |
  | `make docker-down` | Stop all services |
  | `make docker-logs` | Follow container logs |
  | `make docker-restart` | Down then up |

  ## Project Status

  MVP complete -- production ready for initial deployment.

  ### Completed Features

  - Company CRUD with role catalog management
  - Staff CRUD with role assignment validation
  - WhatsApp webhook integration for check-in/check-out
  - Activity log tracking with configurable action types
  - Work session computation and cost calculation
  - Dashboard with real-time stats
  - Action type configuration UI and staff management UI
  - Webhook authentication and atomic operations

  ### Known Limitations

  - No authentication for dashboard/API access
  - Overtime alert thresholds not yet implemented
  - Only CHECK_IN/CHECK_OUT action types have built-in business logic
  - No CSV/PDF export or email notifications
  - Controller and repository integration tests pending
  - List operations have N+1 query patterns (optimization opportunity)

  ## Documentation

  - [AGENTS.md](AGENTS.md) — Agent instructions, cell guides, and quick-start mapping
  - [docs/rules/01-architecture.md](docs/rules/01-architecture.md) — Architecture principles
  - [docs/rules/02-domain-model.md](docs/rules/02-domain-model.md) — Domain model conventions
  - [docs/rules/03-database.md](docs/rules/03-database.md) — Database conventions
  - [docs/rules/04-api-and-webhook.md](docs/rules/04-api-and-webhook.md) — API and webhook conventions
  - [docs/rules/05-testing.md](docs/rules/05-testing.md) — Testing strategy
  - [docs/rules/06-development.md](docs/rules/06-development.md) — Development guidelines
  ```

- [ ] **Step 2: Save, commit, and report**

  Save the file. Commit with message: `docs: rewrite README.md as overview and getting-started guide`
  Report the new line count.

---

### Task 6: Verify refactor completeness

**Files:**
- Read: all modified files

- [ ] **Step 1: Check README.md scope**

  Confirm `README.md` does not contain any of the following strings:
  - `Architecture` (as a section heading)
  - `Project Structure`
  - `API Endpoints`
  - `WhatsApp Commands`
  - `Database Schema`
  - `CREATE TABLE`
  - `GET /api/companies`
  - `POST /webhook/message`
  - `Testing` (as a section heading with layer table)
  - `Error Handling` (as a section heading with status table)

  Allowed: links to these topics, and the string `WhatsApp` in the description/status.

- [ ] **Step 2: Check content destinations**

  Confirm:
  - `apps/api/internal/activity/AGENTS.md` contains a `## WhatsApp Commands` section.
  - `docs/rules/05-testing.md` contains a `## Test Inventory` section.
  - `docs/rules/03-database.md` contains a `## Design Decisions` section.
  - `docs/rules/06-development.md` maps `ErrUnauthorized` → 401.

- [ ] **Step 3: Check links**

  Confirm all internal links in the new `README.md` resolve to existing files.

- [ ] **Step 4: Check for duplication**

  Confirm the new README content does not duplicate details already in `docs/rules/` or cell `AGENTS.md` files.

- [ ] **Step 5: Report results**

  Report pass/fail for each check. If any check fails, fix the offending file and re-run.

---

## Spec Coverage

| Spec Requirement | Task |
|---|---|
| WhatsApp command details live in `activity/AGENTS.md` | Task 1 |
| Testing strategy details live in `docs/rules/05-testing.md` | Task 2 |
| Database design decisions live in `docs/rules/03-database.md` | Task 3 |
| `ErrUnauthorized` mapping lives in `docs/rules/06-development.md` | Task 4 |
| `README.md` is concise overview + getting started | Task 5 |
| No content lost, no duplication | Task 6 |

## Placeholder Scan

No placeholders are used. Every task names exact files, exact source lines, and exact replacement content.
