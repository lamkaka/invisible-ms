# Dashboard Cell

## Purpose
Provides aggregated read-only views of company activity: today's overview, cost tracking, staff activity metrics, and overtime alerts. Implements CQRS with a dedicated read-side repository using SQL aggregations.

## Owned Aggregates
None (read-only aggregation). Consumes data from `activity_logs`, `staff`, and `company_roles` tables via SQL queries.

## File Inventory

| File | Responsibility |
|---|---|
| `dashboard_domain.go` | Stats value objects: `DashboardStats`, `TodayOverview`, `ActiveStaff`, `CostTracking`, `StaffActivity`, `StaffStats`, `OvertimeAlert`, `ActionTypeCount` |
| `dashboard_service.go` | `DashboardService` — orchestrates multiple repository calls, assembles the `DashboardStats` response |
| `dashboard_repository.go` | `DashboardRepository` interface + `SpannerDashboardRepository` adapter — SQL aggregation queries against all relevant tables |
| `dashboard_api_controller.go` | `DashboardAPIController` — `GET /api/dashboard/stats` JSON endpoint |
| `dashboard_web_controller.go` | `DashboardWebController` — server-rendered HTML pages (`/dashboard`, `/staff`, `/actions`) using Go templates |
| `dashboard_api_controller_test.go` | HTTP tests for API controller |
| `dashboard_web_controller_test.go` | HTTP tests for web controller |
| `dashboard_service_test.go` | Service layer tests |

## Inbound Dependencies
- No Go-level dependencies on other cells (reads data directly from shared database via SQL queries)
- `internal/shared` — error types (indirectly)

## Outbound Dependencies
- Reads from: `activity_logs`, `staff`, `company_roles`, `company_action_types` tables (via SQL queries, not Go services)

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/dashboard/stats?company_code=` | Aggregated JSON dashboard stats |
| GET | `/dashboard?company_code=` | HTML dashboard page |
| GET | `/staff` | HTML staff management page |
| GET | `/actions` | HTML action type management page |

## Query Patterns

The dashboard cell relies on SQL aggregation queries against `activity_logs`, `staff`, and `company_roles` tables:

- Session pairing: Use correlated subqueries to pair CHECK_IN with next CHECK_OUT
- Cost calculation: JOIN with `company_roles` to get hourly_rate in same query
- Aggregations: Use `SUM`, `COUNT`, `AVG` in SQL, not in Go code
- Time-based filtering: Use `TIMESTAMP_DIFF` for duration calculations

### Index Usage

- `activity_logs_by_staff` — querying activity per worker with time range
- `activity_logs_by_company` — querying activity per company with time range
- `activity_logs_by_action` — querying open check-ins per company

## Cell-Specific Business Rules
- Templates are parsed once at controller creation time (not per-request)
- All queries use Spanner `Single()` read (no transactions needed for read-only)
- Session pairing is done in SQL using correlated subqueries (not in Go)
- Overtime threshold is configurable; default is 8 hours per day
- Cost calculation joins with `company_roles` for hourly rates within the SQL query
- Currently-working detection: latest CHECK_IN per staff+role with no following CHECK_OUT
- `company_code` query parameter is required for most queries (multi-tenant isolation)
- All queries filter by company_code to prevent cross-tenant data leakage

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
