# Docker Local Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Set up and run the IMS application locally using Docker Compose with the Cloud Spanner emulator.

**Architecture:** The existing `docker-compose.yml` orchestrates four services: the Spanner emulator, a one-shot migration job, the Go application, and an nginx reverse proxy. The migration job creates the Spanner instance, database, and applies all SQL migrations before the app starts.

**Tech Stack:** Docker, Docker Compose, Google Cloud Spanner emulator, Go 1.24, Alpine Linux, nginx

---

## File Structure

Files involved in this setup:

- `docker-compose.yml` — service orchestration (already exists)
- `Dockerfile` — Go application build (already exists)
- `nginx/Dockerfile` and `nginx/nginx.conf` — reverse proxy (already exist)
- `migrations/*.sql` — schema and seed migrations (already exist)
- `cmd/migrate/main.go` — migration runner (already exists)
- `cmd/server/main.go` — application entry point (already exists)
- `docs/superpowers/plans/2026-06-16-docker-local-setup.md` — this plan

No code changes are required unless validation reveals an issue.

---

## Task 1: Verify Docker Tooling

**Files:**
- None

- [x] **Step 1: Check Docker daemon**

Run: `docker --version && docker info`
Expected: Docker is installed and the daemon is reachable.

- [x] **Step 2: Check Docker Compose**

Run: `docker compose version`
Expected: Docker Compose v2 is installed.

---

## Task 2: Build and Start Services

**Files:**
- `docker-compose.yml`
- `Dockerfile`

- [x] **Step 1: Start all services in detached mode**

Run: `docker compose up --build -d`
Expected: All four services start; `migrate` exits successfully; `app` becomes healthy; `nginx` binds to host port `8888`.

- [x] **Step 2: Inspect service status**

Run: `docker compose ps`
Expected: `spanner-emulator` running, `migrate` exited (0), `app` healthy, `nginx` running.

---

## Task 3: Verify Migrations and Application Health

**Files:**
- `cmd/migrate/main.go`

- [x] **Step 1: Check migration logs**

Run: `docker compose logs migrate`
Expected: Logs show instance and database created, all DDL/DML statements applied successfully.

- [x] **Step 2: Check application logs**

Run: `docker compose logs app`
Expected: Server started on port 8080, no Spanner connection errors.

- [x] **Step 3: Health-check the dashboard API**

Run: `curl -s http://localhost:8888/api/dashboard/stats`
Expected: HTTP 200 with JSON stats (likely empty values on a fresh database).

---

## Task 4: Seed Test Data and Verify Endpoints

**Files:**
- None (uses existing API endpoints)

- [x] **Step 1: Create a test company and role**

Run:
```bash
curl -s -X POST http://localhost:8888/api/companies \
  -H "Content-Type: application/json" \
  -d '{"company_code":"ACME","company_name":"Acme Corp"}'

curl -s -X POST http://localhost:8888/api/companies/ACME/roles \
  -H "Content-Type: application/json" \
  -d '{"role_name":"CLEANING","hourly_rate":15.50}'
```
Expected: HTTP 201 for both requests.

- [x] **Step 2: Create a test staff member**

Run:
```bash
curl -s -X POST http://localhost:8888/api/staff \
  -H "Content-Type: application/json" \
  -d '{"staff_id":"staff-001","phone_number":"+1234567890","name":"John Doe","company_code":"ACME","roles":["CLEANING"]}'
```
Expected: HTTP 201 with staff JSON.

- [x] **Step 3: Send a test webhook check-in**

Run:
```bash
curl -s -X POST http://localhost:8888/webhook/message \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: test-secret" \
  -d '{"phone":"+1234567890","message":"IN CLEANING","company_code":"ACME"}'
```
Expected: HTTP 200/201 with confirmation.

- [x] **Step 4: Verify dashboard reflects activity**

Run: `curl -s http://localhost:8888/api/dashboard/stats`
Expected: JSON shows at least one active session and one staff member checked in today.

- [x] **Step 5: Open the dashboard in a browser (optional)**

URL: `http://localhost:8888/dashboard`
Expected: HTML dashboard page loads without errors.

---

## Spec Coverage

- Spanner emulator setup: Task 2
- Database/instance creation and migrations: Task 2 and Task 3
- Application startup and health: Task 2 and Task 3
- End-to-end smoke test via API and webhook: Task 4

## Placeholder Scan

No placeholders. All commands are exact and executable.

## Execution Notes

- **2026-06-16:** Plan executed inline.
- All services started successfully.
- One code fix was required during setup: `internal/dashboard/dashboard_repository.go` referenced `worker_id` instead of `staff_id` in `GetTotalHoursToday`, causing the dashboard health check to return HTTP 500 on a fresh database. Fixed to `staff_id`.
- `go test ./...` passes after the fix.
- End-to-end smoke test completed: company, role, staff, webhook check-in/out, and dashboard stats all work.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-16-docker-local-setup.md`.

Two execution options:

1. **Inline Execution** — I run the commands in this session and report results.
2. **Subagent-Driven** — I dispatch a subagent to execute and verify each task.

Given the operational nature of this setup, inline execution is recommended.
