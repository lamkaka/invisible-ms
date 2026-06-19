# GitHub Actions PR Test Workflow

## Goal

Run automated checks on every pull request: a fast unit-test job and a slower end-to-end (e2e) job that exercises the full stack against a Spanner emulator.

## Trigger

The workflow runs on pull request events:

- `opened`
- `synchronize`
- `reopened`

## Workflow file

`.github/workflows/pr-tests.yml`

## Job 1: unit-tests

Runs on `ubuntu-latest` with `working-directory: apps/api`.

Steps:

1. Check out the repository.
2. Set up Go using `go-version-file: apps/api/go.mod` (currently Go 1.26.4).
3. Run `go build ./...`.
4. Run `go vet ./...`.
5. Run `go test ./...`.

The existing tests are mocked and do not need a Spanner instance.

## Job 2: e2e

Runs on `ubuntu-latest` after `unit-tests` succeeds.

### Service container

- Image: `gcr.io/cloud-spanner-emulator/emulator`
- Port: `9010`

### Environment variables

```text
GCP_SPANNER_PROJECT_ID=invisible-ms-local
GCP_SPANNER_INSTANCE_ID=invisible-ms-instance
GCP_SPANNER_DATABASE_ID=invisible-ms-db
GCP_SPANNER_EMULATOR_HOST=localhost:9010
WEBHOOK_SECRET=test-secret
PORT=8080
TEMPLATES_PATH=../web/templates
STATIC_PATH=../web/static
```

### Steps

1. Check out the repository.
2. Set up Go using `apps/api/go.mod`.
3. Run migrations: `go run ./cmd/migrate`.
4. Start the server in the background: `go run ./cmd/server`.
5. Wait until `http://localhost:8080` responds.
6. Run `go test ./e2e/...`.

## E2E test package

Location: `apps/api/e2e/e2e_test.go` (package `e2e`).

The test skips itself when `GCP_SPANNER_EMULATOR_HOST` is empty, so `go test ./...` in the unit-test job does not try to run it. The test reads its target URL from `E2E_BASE_URL` and defaults to `http://localhost:8080`.

### CRUD cycle coverage

**Company**

- POST `/api/companies` to create company `E2E`.
- GET `/api/companies/E2E`.
- GET `/api/companies`.

**Roles**

- POST `/api/companies/E2E/roles` to add `CLEANING`.
- PUT `/api/companies/E2E/roles/CLEANING` to update the hourly rate.
- Add a second role, assign and unassign it from staff, then DELETE the role.

**Action types**

- POST `/api/companies/E2E/action-types` to create `BREAK_START` with keyword `BREAK`.
- PUT `/api/companies/E2E/action-types/BREAK_START` to change the keyword.
- DELETE `/api/companies/E2E/action-types/BREAK_START`.

**Staff**

- POST `/api/staff` to create a staff member with role `CLEANING`.
- GET `/api/staff/{id}`.
- GET `/api/staff?company_code=E2E`.
- POST `/api/staff/{id}/roles` to assign a second role.
- DELETE `/api/staff/{id}/roles/{role}` to unassign it.

**Activity**

- POST `/webhook/message` with header `X-Webhook-Secret` to send `IN`.
- POST `/webhook/message` to send `OUT`.
- GET `/api/activities`.
- GET `/api/activities/sessions`.

**Dashboard**

- GET `/api/dashboard/stats?company_code=E2E`.
- GET `/dashboard?company_code=E2E` and assert a 200 HTML response.

## Out of scope

- Race detection or coverage reporting.
- Multi-OS or multi-Go-version matrix builds.
- Browser/UI automation with Playwright.
- Notifications or status posting.

## Future possibilities

- Add `-race` to unit tests.
- Upload coverage reports.
- Run the workflow on pushes to `main` in addition to pull requests.
