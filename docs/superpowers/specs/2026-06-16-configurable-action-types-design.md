# Configurable Company Action Types

## Overview

Allow each company to configure the action types available for activity tracking. The system provides base action types (CHECK_IN, CHECK_OUT) that are always present, and companies can add custom action types on top. Companies can also customize the WhatsApp keywords for all action types, including system ones.

## Problem

Action types are currently hardcoded as Go constants in `activity_domain.go`. Only `IN` and `OUT` keywords are parsed from WhatsApp messages. Companies cannot define their own action types or customize keywords to match their workflows.

## Solution

### Data Model

#### New Table: `company_action_types`

```sql
CREATE TABLE company_action_types (
  company_code STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  keyword STRING(20) NOT NULL,
  is_system BOOL NOT NULL DEFAULT FALSE,
) PRIMARY KEY (company_code, action_type),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;

CREATE UNIQUE INDEX company_action_types_by_keyword 
  ON company_action_types(company_code, keyword);
```

#### Domain Value Object

```go
type CompanyActionType struct {
    ActionType string
    Keyword    string
    IsSystem   bool
}
```

#### Seeding

When a company is created, auto-seed two system action types in the same transaction:

| action_type | keyword | is_system |
|-------------|---------|-----------|
| `CHECK_IN`  | `IN`    | `true`    |
| `CHECK_OUT` | `OUT`   | `true`    |

#### Constraints

- `keyword` is unique per company (enforced by unique index)
- System action types (`CHECK_IN`, `CHECK_OUT`) cannot be deleted, but their keywords can be changed
- Custom action types cannot use keywords already taken by another action type in the same company
- `action_type` must be uppercase, alphanumeric + underscores only
- `keyword` must be non-empty, uppercase, alphanumeric + underscores only

### Message Parsing

#### Current Flow

```
Worker sends "IN CLEANING" → ParseMessage hardcodes "IN" → ActionCheckIn
```

#### New Flow

```
Worker sends "IN CLEANING" → ParseMessage looks up company's action types →
  finds keyword "IN" maps to action_type "CHECK_IN" → returns CHECK_IN
```

#### ParseMessage Signature Change

Current:
```go
func ParseMessage(message string, numWorkerRoles int) (ActionType, string, error)
```

New:
```go
func ParseMessage(message string, numWorkerRoles int, actionTypes []CompanyActionType) (string, string, error)
```

- Takes the company's configured action types as input
- Builds a keyword → action_type map from the provided action types
- Resolves the first word of the message against the keyword map
- Returns the `action_type` name as a string instead of the hardcoded `ActionType` enum
- If no keyword matches → `ErrUnknownAction`

#### Backward Compatibility

Default seeded keywords (`IN`, `OUT`) mean existing workers see no change. Only companies that customize keywords will see different behavior.

### API Endpoints

#### Company Action Type Management

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/api/companies/:code/action-types` | List all action types for a company |
| `POST` | `/api/companies/:code/action-types` | Add a custom action type |
| `PUT` | `/api/companies/:code/action-types/:action` | Update keyword for an action type |
| `DELETE` | `/api/companies/:code/action-types/:action` | Delete a custom action type (system types blocked) |

#### Request/Response Examples

**Create custom action type:**
```json
POST /api/companies/acme/action-types
{
  "action_type": "BREAK_START",
  "keyword": "BREAK"
}
```

**Update keyword (including system types):**
```json
PUT /api/companies/acme/action-types/CHECK_IN
{
  "keyword": "CLOCK_IN"
}
```

**Delete (custom only):**
```json
DELETE /api/companies/acme/action-types/BREAK_START
```
Returns 400 Bad Request if `action_type` is a system type.

#### Validation Rules

- `action_type`: required, uppercase alphanumeric + underscores, unique per company
- `keyword`: required, uppercase alphanumeric + underscores, unique per company
- Cannot delete system types
- Cannot change `action_type` name (only keyword) — delete + recreate instead

### Service & Repository Changes

#### Company Cell

- `CompanyService.CreateCompany()` — after creating company, seed `CHECK_IN`/`OUT` with keywords `IN`/`OUT` in the same transaction
- New `CompanyActionTypeRepository` port + Spanner adapter in the `company` cell
- CRUD methods for action types with validation (system type protection, keyword uniqueness)

#### Activity Cell

- `WebhookService.ProcessWebhook()`:
  1. Look up worker → get `company_code`
  2. Fetch company's action types from `CompanyActionTypeRepository`
  3. Pass action types to `ParseMessage()`
  4. Rest of flow unchanged — creates `ActivityLog` with resolved `action_type` string

- `ActivityLog` entity: change `ActionType` field from the hardcoded `ActionType` enum to `string`
- Keep well-known string constants for system action types in `activity_domain.go` for reference (e.g., `const ActionCheckIn = "CHECK_IN"`), but do not enforce them — they are documentation, not validation

#### Session Pairing

- Session service still pairs by `action_type == "CHECK_IN"` and `action_type == "CHECK_OUT"` — these string values are stable because system types can't be deleted
- The keyword changes, but the `action_type` name stored in `activity_logs` stays the same

#### Dependency

- `activity` cell gains a dependency on `company` cell's `CompanyActionTypeRepository` port (similar to how it already depends on company for role validation)

### Dashboard Changes

#### Current Dashboard Queries

Hardcode `WHERE action_type IN ('CHECK_IN', 'CHECK_OUT')` — these continue to work unchanged since system action type names are stable.

#### New Dashboard Insights

Add a new section to surface custom action type data:

- **Action type breakdown**: count of each action type today (system + custom)
- **Per-worker custom actions**: which workers are triggering custom actions and how often
- **Timeline view**: custom actions shown alongside check-in/out in activity feeds

#### Implementation

- New query in `dashboard_repository.go`: `GetActionTypeBreakdown(companyCode, dateRange)` — groups by `action_type`, returns counts
- Dashboard HTML template: add a card/section showing action type distribution
- No changes to session/cost queries — custom types are informational only

### Migration & Backward Compatibility

#### Migration

New migration file: `004_create_company_action_types.sql`

```sql
CREATE TABLE company_action_types (
  company_code STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  keyword STRING(20) NOT NULL,
  is_system BOOL NOT NULL DEFAULT FALSE,
) PRIMARY KEY (company_code, action_type),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;

CREATE UNIQUE INDEX company_action_types_by_keyword 
  ON company_action_types(company_code, keyword);
```

#### Backward Compatibility

- Existing companies: a data migration script inserts `CHECK_IN` (keyword `IN`) and `CHECK_OUT` (keyword `OUT`) as system action types for each existing company in the `companies` table
- Existing `activity_logs` records: no changes needed — `action_type` column already stores strings like `CHECK_IN`
- Existing workers: default keywords (`IN`/`OUT`) mean no behavior change unless a company customizes

### Files Changed Summary

| File | Change |
|------|--------|
| `migrations/004_create_company_action_types.sql` | New table |
| `company_domain.go` | Add `CompanyActionType` value object, validation |
| `company_repository.go` | Add `CompanyActionTypeRepository` port + adapter |
| `company_service.go` | Seed action types on company creation, CRUD for action types |
| `company_handler.go` | New API endpoints for action type management |
| `activity_domain.go` | Change `ActionType` to `string`, update `ParseMessage` signature |
| `activity_webhook_service.go` | Fetch company action types, pass to parser |
| `activity_session_service.go` | Use string comparison instead of enum |
| `activity_repository.go` | Use string comparison in queries |
| `dashboard_repository.go` | Add action type breakdown query |
| `dashboard_web_handler.go` / templates | Show custom action type insights |

## Scope Boundaries

### In Scope

- Company-level action type configuration (CRUD)
- Custom WhatsApp keywords for all action types (including system)
- System action types (CHECK_IN, CHECK_OUT) always present, not deletable
- Custom action types logged as activity records
- Dashboard insights for custom action types (counts, breakdowns)
- Backward compatibility with existing data

### Out of Scope

- Custom action types affecting session pairing or cost calculations
- Per-role action type configuration (all roles in a company share the same set)
- Action type categories or behavior configuration (break, overtime, etc.)
- Worker-facing UI for viewing available action types
