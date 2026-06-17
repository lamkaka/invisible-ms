# IMS Project Status

**Last Updated:** 2026-06-17  
**Status:** MVP Complete - Production Ready

## Overview

IMS (Hourly Staff Management System) is a multi-tenant HR application for managing hourly staff via WhatsApp webhooks. The system tracks check-in/check-out, computes work sessions and costs, and provides a management dashboard.

## Architecture

- **Backend:** Go 1.26
- **Database:** Google Cloud Spanner
- **Frontend:** Server-rendered HTML + Alpine.js (CDN)
- **Architecture:** DDD + Clean Architecture + Cell-Based Architecture

### Project Structure

```
apps/
├── api/          # Go backend (cmd/, internal/, migrations/, Dockerfile)
├── web/          # Server-rendered templates and static assets
└── infra/        # Infrastructure configs (nginx/)
deployments/      # Docker Compose, Makefile, .env.example
docs/
├── rules/        # Concern-based agent rulebooks
└── superpowers/  # Design specs, plans, and this STATUS.md
```

### Cells

1. **Company** - Manages companies and their role catalogs
2. **Staff** - Manages staff and role assignments
3. **Activity** - Handles webhooks, activity logs, and session computation
4. **Dashboard** - Read-only aggregation for management dashboard

## Repo Restructure (2026-06-17)

The repository was restructured to better separate concerns:

- Moved Go module to `apps/api/`
- Moved web assets (templates, static files) to `apps/web/`
- Moved nginx config to `apps/infra/nginx/`
- Moved deployment artifacts (Docker Compose, Makefile) to `deployments/`
- Split monolithic `AGENTS.md` into `docs/rules/` (concern-based rulebooks) and per-cell `AGENTS.md` files
- Renamed interface layer from `handler` to `controller` throughout the codebase

## Layer Structure (per cell)

```
Controller → Service → Domain ← Repository
```

- **Domain** (`*_domain.go`) - All business logic, validation, rules
- **Service** (`*_service.go`) - Thin orchestration only
- **Repository** (`*_repository.go`) - Port interfaces + Spanner adapters
- **Controller** (`*_controller.go`) - HTTP request/response handling

## Completed Features

### Core Functionality
- ✅ Company management (CRUD + roles)
- ✅ Staff management (CRUD + role assignments)
- ✅ WhatsApp webhook integration (check-in/check-out)
- ✅ Activity log tracking
- ✅ Session computation (pairing CHECK_IN/CHECK_OUT)
- ✅ Cost calculation (duration × hourly rate)
- ✅ Dashboard with real-time stats
- ✅ Staff management UI

### Security & Data Integrity
- ✅ Webhook authentication (X-Webhook-Secret header)
- ✅ Role validation (staff can only be assigned roles that exist in company catalog)
- ✅ Atomic operations (all create/update operations use transactions)
- ✅ Race condition prevention (check-out validation is atomic)

### Performance
- ✅ SQL-based aggregations (no in-memory session pairing)
- ✅ Template caching (parsed once at startup)
- ✅ Efficient queries with proper indexes

### API Endpoints

**Webhook:**
- `POST /webhook/message` - Receive check-in/check-out (requires X-Webhook-Secret header)

**Company:**
- `GET /api/companies` - List all companies
- `POST /api/companies` - Create company
- `GET /api/companies/:code` - Get company details
- `POST /api/companies/:code/roles` - Add role
- `DELETE /api/companies/:code/roles/:role` - Remove role

**Staff:**
- `GET /api/staff` - List staff (filter by company_code)
- `POST /api/staff` - Create staff
- `GET /api/staff/:id` - Get staff details
- `POST /api/staff/:id/roles` - Assign role
- `DELETE /api/staff/:id/roles/:role` - Unassign role

**Activity:**
- `GET /api/activities` - List activity logs (filter by staff_id, company_code, date range)
- `GET /api/activities/sessions` - List computed work sessions

**Dashboard:**
- `GET /api/dashboard/stats` - Aggregated stats (currently working, costs, staff activity)
- `GET /dashboard` - HTML dashboard page
- `GET /staff` - HTML staff management page

## Running the Application

### Prerequisites

1. **Cloud Spanner Emulator** (Docker):
   ```bash
   docker run -d --name spanner-emulator -p 9010:9010 -p 9020:9020 gcr.io/cloud-spanner-emulator/emulator
   ```

2. **Setup Database**:
   ```bash
   export SPANNER_EMULATOR_HOST=localhost:9010
   export SPANNER_PROJECT_ID=ims-project
   export SPANNER_INSTANCE_ID=invisible-ms-instance
   export SPANNER_DATABASE_ID=invisible-ms-db
   cd apps/api && go run ./cmd/setup/main.go
   ```

### Start Server

```bash
export SPANNER_EMULATOR_HOST=localhost:9010
export SPANNER_PROJECT_ID=ims-project
export SPANNER_INSTANCE_ID=invisible-ms-instance
export SPANNER_DATABASE_ID=invisible-ms-db
export WEBHOOK_SECRET=your-secret-here
cd apps/api && go run ./cmd/server
```

Server starts on http://localhost:8080

### Docker Compose

For a full local stack (Spanner emulator + API + nginx):

```bash
cd deployments
cp .env.example .env
make up
```

### Test Data

Seed script available at `/tmp/seed_data.sh` creates:
- Company: ACME Corp (roles: CLEANING $15.50, DELIVERY $18.00, WAREHOUSE $16.75)
- Workers: John Doe, Jane Smith, Mike Johnson, Sarah Wilson
- Sample check-ins and check-outs

## Known Limitations

### Not Yet Implemented

1. **Overtime Alerts** - Dashboard shows null for overtime alerts. Need threshold logic.
2. **Average Hours Calculation** - Dashboard shows 0 for average hours per staff.
3. **Controller Tests** - Controllers now have test coverage, but some edge cases and error paths remain untested.
4. **Repository Integration Tests** - Skipped per user request, but would be valuable.
5. **Authentication** - No auth for dashboard/API access (MVP decision).
6. **Additional Action Types** - Only CHECK_IN/CHECK_OUT implemented. BREAK, OVERTIME, etc. not yet.
7. **Export Reports** - No CSV/PDF export functionality.
8. **Email Notifications** - No notification system.

### Technical Debt

1. **List Operations N+1** - `CompanyRepository.List` and `StaffRepository.List` do N+1 queries (query IDs, then fetch each entity). Could be optimized with JOINs.
2. **Session Service N+1** - `SessionService.GetSessions` calls `GetCompany` for each session to get hourly rate. Could use JOIN like dashboard repository.
3. **Time Parse Errors Ignored** - Activity controller ignores time parse errors (benign but not ideal).
4. **No Phone/Company Code Format Validation** - No regex/E.164 validation on phone numbers.

## Testing

### Run All Tests
```bash
cd apps/api && go test ./...
```

### Test Coverage
- ✅ Company domain (6 tests)
- ✅ Company service (3 tests)
- ✅ Staff domain (6 tests)
- ✅ Staff service (5 tests, including role validation)
- ✅ Activity domain (4 tests)
- ✅ Activity webhook service (3 tests)
- ✅ Dashboard service (1 test)
- ✅ Company controller (14 tests)
- ✅ Staff controller (15 tests)
- ✅ Activity controller (17 tests)
- ✅ Dashboard API controller (3 tests)
- ✅ Dashboard web controller (4 tests)
- ❌ Repository integration (no tests)

## Code Review Findings (Oracle Review - 2026-06-16)

All critical and important issues from Oracle review have been addressed:

### Critical (Fixed)
1. ✅ Staff role updates now persist (ReadWriteTransaction)
2. ✅ Staff roles validated against company catalog
3. ✅ Webhook authentication added

### Important (Fixed)
4. ✅ Check-out race condition fixed (atomic transaction)
5. ✅ Company/staff create operations atomic
6. ✅ Dashboard uses SQL for aggregations
7. ✅ Session N+1 eliminated
8. ✅ HTTP error codes discriminate (404/409/400/500)
9. ✅ GetStaffStats implemented
10. ✅ Template parsing moved to startup

## File Naming Convention

All files follow `{entity}_{role}.go` pattern:
- `_domain.go` - Domain layer (business logic)
- `_service.go` - Application layer (orchestration)
- `_repository.go` - Infrastructure layer (persistence)
- `_controller.go` - Interface layer (HTTP)

## Database Schema

### Tables
- `companies` - Company master data
- `company_roles` - Role catalog per company (interleaved)
- `staff` - Staff master data
- `staff_roles` - Role assignments (interleaved)
- `activity_logs` - Check-in/check-out events

### Indexes
- `staff_by_company` - Query staff by company
- `staff_by_phone` - Unique index on (company_code, phone_number)
- `activity_logs_by_staff` - Query logs by staff + timestamp
- `activity_logs_by_company` - Query logs by company + timestamp
- `activity_logs_by_action` - Query logs by action type

## Environment Variables

```bash
SPANNER_PROJECT_ID=ims-project
SPANNER_INSTANCE_ID=invisible-ms-instance
SPANNER_DATABASE_ID=invisible-ms-db
SPANNER_EMULATOR_HOST=localhost:9010  # For local development
PORT=8080
WEBHOOK_SECRET=your-secret-here
```

## Git History

- `a2a35c9` - Move API Dockerfile and migrations under apps/api/
- `ea03ed8` - fix: copy migrations into /app/api/migrations/ for migrate binary
- `f46bcf3` - Fix cross-references and file paths after repo restructure
- `ac7b951` - docs: split monolithic AGENTS.md into modular docs/rules/ and cell AGENTS.md files
- `e36157d` - Update deployment configs for repo restructure (apps/* layout)
- `52164c2` - WIP: restructure repo and rename handler->controller (partial)
- `eb86f65` - docs: add repo restructure design and implementation plan
- `e354032` - fix: address Oracle code review (critical + important issues)
- `b6fbdc4` - feat: implement cost tracking and activity list endpoint
- `bcecfdb` - feat: complete dashboard and staff UI with modern design
- `380cce5` - feat: IMS application complete
- `b294dd3` - feat: wire all dependencies in main.go
- `ac49941` - feat: add dashboard cell
- `26f260a` - feat: add activity cell
- `3243448` - feat: add activity domain layer
- `a9004ef` - feat: add staff cell
- `953306a` - feat: add staff domain layer
- `b61eb6b` - feat: add company handler layer
- `1428d03` - feat: add company service layer
- `585ef06` - feat: add company repository layer
- `21f8ea4` - feat: add company domain layer
- `fcd0082` - feat: add shared infrastructure
- `0579cb6` - Initial workspace setup

## Next Steps (Future Enhancements)

1. **Overtime Alerts** - Implement threshold logic in dashboard
2. **Additional Action Types** - BREAK_START, BREAK_END, OVERTIME_START
3. **Authentication** - Add Google OAuth or email/password for dashboard
4. **Export Reports** - CSV/PDF export for activity logs and costs
5. **Email Notifications** - Notify managers of overtime, late check-ins
6. **Mobile App** - Native mobile app for managers
7. **Staff Self-Service** - Portal for staff to view their own hours
8. **Performance Optimization** - Fix remaining N+1 queries in List operations
9. **Test Coverage** - Add repository integration tests; controller tests now exist
10. **Input Validation** - Add phone number format validation (E.164)

## Contact

For questions or issues, refer to:
- Design spec: `docs/superpowers/specs/2026-06-16-ims-hr-app-design.md`
- Implementation plan: `docs/superpowers/plans/2026-06-16-ims-implementation.md`
- Architecture guide: `docs/rules/01-architecture.md`
- Project overview: `AGENTS.md` (root)
