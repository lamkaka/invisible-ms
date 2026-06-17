# Activity Cell

## Purpose
Records and tracks worker activity logs (check-in, check-out). Processes incoming WhatsApp webhook messages, parses them against company-configured keywords, and computes work sessions with duration and cost.

## Owned Aggregates
- **ActivityLog** (aggregate root): `log_id` (UUID, PK), `staff_id`, `company_code`, `role`, `action_type`, `timestamp`, `metadata`

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

## Cell-Specific Business Rules
- Webhook requires `X-Webhook-Secret` header (constant-time comparison)
- Message keywords are case-insensitive, resolved via company-configured keyword map
- Messages with more than 2 words are rejected
- Unknown keywords return parse error
- Role is required if staff has multiple assigned roles; inferred if staff has exactly one
- Check-out validates an active check-in exists atomically (within ReadWriteTransaction)
- Double check-out (no active check-in) is rejected with `ErrNoActiveCheckIn`
- Session pairing: in-memory pairing of check-ins → check-outs by staff+role; unpaired check-ins are ignored
- Cost = duration hours × role hourly rate (fetched from company role catalog)
- Activity logs are immutable (no updates or deletes)

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
