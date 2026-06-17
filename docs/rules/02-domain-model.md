# Domain Model

## Aggregates and Value Objects

### Company (Aggregate Root)
- `company_code` (string, unique) — tenant identifier
- `company_name` (string)
- `roles` (collection of Role value objects)

### Role (Value Object)
- `name` (string) — e.g., "CLEANING", "DELIVERY"
- `hourly_rate` (decimal) — cost per hour for this role

### Staff (Aggregate Root)
- `staff_id` (string, UUID)
- `phone_number` (string) — unique within company
- `name` (string)
- `company_code` (string) — FK to Company
- `assigned_roles` ([]string) — list of role names from company's catalog
- `is_active` (bool)

### ActivityLog (Aggregate Root)
- `log_id` (string, UUID)
- `staff_id` (string)
- `company_code` (string)
- `role` (string) — the role being worked
- `action_type` (enum) — CHECK_IN, CHECK_OUT, BREAK_START, BREAK_END, OVERTIME_START, etc.
- `timestamp` (timestamp)
- `metadata` (JSON, optional) — extra context for future action types

### CompanyActionType (Value Object)
- `action_type` (string) — stable identifier stored in `activity_logs.action_type`
- `keyword` (string) — WhatsApp keyword (e.g., "IN", "OUT")
- `is_system` (bool) — system action types cannot be deleted

## Session Computation

A "work session" is derived by pairing the most recent CHECK_IN with the next CHECK_OUT for the same staff + role. Duration and cost are computed from the pair:

- Duration = CHECK_OUT timestamp - CHECK_IN timestamp
- Cost = duration (in hours) × role's hourly rate

Session computation happens in `activity_session_service.go` and is also embedded in SQL queries in the dashboard repository.

## Cross-Cell Business Rules

### Staff Identification
- Workers are identified by `phone_number` + `company_code`
- The webhook payload includes both fields

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

### Role Validation
- Workers can only be assigned roles that exist in the company's `company_roles` table
- `StaffService` depends on `CompanyService` to validate roles
- Validation happens in `CreateStaff` and `AssignRole` methods
- Prevents phantom roles that would break cost calculations

### Cost Calculation
- Duration = CHECK_OUT timestamp - CHECK_IN timestamp
- Cost = duration (in hours) × role's hourly rate
- Computed on-the-fly or cached in read model
