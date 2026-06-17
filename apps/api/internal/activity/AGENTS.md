# Activity Cell

## Purpose
Records and tracks worker activity logs (check-in, check-out). Processes incoming WhatsApp webhook messages, parses them against company-configured keywords, and computes work sessions with duration and cost.

## Owned Aggregates

### ActivityLog (Aggregate Root)
- `log_id` (string, UUID)
- `staff_id` (string)
- `company_code` (string)
- `role` (string) — the role being worked
- `action_type` (enum) — CHECK_IN, CHECK_OUT, BREAK_START, BREAK_END, OVERTIME_START, etc.
- `timestamp` (timestamp)
- `metadata` (JSON, optional) — extra context for future action types

## Database Schema

```sql
CREATE TABLE activity_logs (
  log_id STRING(36) NOT NULL,
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  role STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  metadata JSON,
) PRIMARY KEY (log_id);

CREATE INDEX activity_logs_by_staff ON activity_logs(staff_id, timestamp);
CREATE INDEX activity_logs_by_company ON activity_logs(company_code, timestamp);
CREATE INDEX activity_logs_by_action ON activity_logs(company_code, action_type, timestamp);
```

## File Inventory

| File | Responsibility |
|---|---|
| `activity_domain.go` | ActivityLog aggregate, `NewActivityLog()` validator, `ParseMessage()` — keyword-based message parser, `CalculateSessionDuration()`, `CalculateSessionCost()` |
| `activity_webhook_service.go` | `WebhookService` — orchestrates end-to-end webhook processing: staff lookup, action type fetching, message parsing, role validation, atomic persist |
| `activity_session_service.go` | `SessionService` — queries activity logs, pairs check-in/check-out into sessions, computes duration and cost |
| `activity_repository.go` | `ActivityRepository` interface + `SpannerActivityRepository` adapter, including atomic `CheckOutWithValidation()` |
| `activity_controller.go` | `ActivityController` — webhook endpoint (with secret validation), activity list, session list |
| `activity_domain_test.go` | Unit tests for domain logic |
| `activity_service_test.go` | Service layer tests |
| `activity_controller_test.go` | Controller HTTP tests |

## Inbound Dependencies
- `internal/company` — `CompanyService.ListActionTypes()` (fetch keyword map), `CompanyService.GetCompany()` (role rate lookup)
- `internal/staff` — `WorkerServiceInterface` (phone-based staff lookup)
- `internal/shared` — error types

## Outbound Dependencies
- None (activity does not expose services to other cells; dashboard reads activity data directly via shared database)

## API Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/webhook/message` | Receive WhatsApp webhook (`X-Webhook-Secret` required) |
| GET | `/api/activities?staff_id=&company_code=&from=&to=` | List activity logs |
| GET | `/api/activities/sessions?company_code=&from=&to=` | List computed work sessions |

## Message Processing Flow

### Check-in/Check-out Flow
1. Worker sends WhatsApp message (e.g., "IN CLEANING" or "OUT")
2. External gateway (Waha) sends webhook to `POST /webhook/message` with `{ phone, message, company_code }`
3. App parses the message:
   - Extracts action (IN/OUT) and optional role
   - If worker has only one role, "IN" is sufficient
   - If worker has multiple roles, role must be specified (e.g., "IN CLEANING")
4. App validates:
   - Worker exists and is active
   - Role is assigned to the worker
   - For CHECK_OUT: worker has an active CHECK_IN for this role
5. App creates an `ActivityLog` record with the appropriate action type
6. App responds with confirmation (optional, via webhook response)

### Message Parsing Rules
- Keywords are case-insensitive (converted to uppercase for matching)
- Format: `{ACTION} [ROLE]`
- Valid actions are defined per-company via `CompanyActionType` configuration
- Default system keywords: `IN` → `CHECK_IN`, `OUT` → `CHECK_OUT`
- Role is optional if worker has only one assigned role
- Invalid messages return an error response

## Cell-Specific Business Rules
- Webhook requires `X-Webhook-Secret` header (constant-time comparison)
- Check-out validates an active check-in exists atomically (within ReadWriteTransaction)
- Double check-out (no active check-in) is rejected with `ErrNoActiveCheckIn`
- Session pairing: in-memory pairing of check-ins → check-outs by staff+role; unpaired check-ins are ignored
- Cost = duration hours × role hourly rate (fetched from company role catalog)
- Activity logs are immutable (no updates or deletes)

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
