# Staff Cell

This guide covers the staff cell only. It defines staff management and role assignment. For project-wide conventions and task routing, read the root AGENTS.md.

## Purpose
Manages workers (staff members) within companies. Handles staff creation, role assignment, and identification via phone number.

## Owned Aggregates

### Staff (Aggregate Root)
- `staff_id` (string, UUID)
- `phone_number` (string) — unique within company
- `name` (string)
- `company_code` (string) — FK to Company
- `assigned_roles` ([]string) — list of role names from company's catalog
- `is_active` (bool)

## Database Schema

### Staff Table
```sql
CREATE TABLE staff (
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT TRUE,
) PRIMARY KEY (staff_id);

CREATE INDEX staff_by_company ON staff(company_code);
CREATE UNIQUE INDEX staff_by_phone ON staff(company_code, phone_number);
```

### Staff Roles Table
```sql
CREATE TABLE staff_roles (
  staff_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL,  -- denormalized for interleaving
) PRIMARY KEY (staff_id, role_name),
  INTERLEAVE IN PARENT staff ON DELETE CASCADE;
```

## File Inventory

| File | Responsibility |
|---|---|
| `staff_domain.go` | Staff aggregate, role assignment/unassignment validation, activate/deactivate |
| `staff_service.go` | Orchestration: CRUD operations, phone-based lookup, role validation against CompanyService |
| `staff_repository.go` | `StaffRepository` interface + `SpannerStaffRepository` adapter (staff + staff_roles tables) |
| `staff_controller.go` | REST endpoints for staff management |
| `staff_domain_test.go` | Unit tests for domain logic |
| `staff_service_test.go` | Service layer tests |
| `staff_controller_test.go` | Controller HTTP tests |

## Inbound Dependencies
- `internal/company` — `CompanyService.GetCompany()` and `CompanyService.HasRole()` for role validation
- `internal/shared` — error types

## Outbound Dependencies
- Exposes `StaffService` to `activity` cell (via `WorkerServiceInterface` for phone-based lookup)
- `StaffService` implements `WorkerServiceInterface` (contract: `GetStaffByPhone(ctx, phone, companyCode)`)

## API Endpoints

API endpoints for this cell are documented in [docs/openapi.json](../../../../docs/openapi.json).

## Cell-Specific Business Rules
- Staff are identified by `phone_number + company_code` (unique constraint)
- Roles assigned to staff must exist in the company's role catalog (validated via `CompanyService`)
- Staff with only one role can use bare "IN" keyword (role inferred automatically)
- Staff with multiple roles must specify the role (e.g., "IN CLEANING")
- Staff can be deactivated (prevents check-in but preserves history)
- `staff_id` is auto-generated as UUID if not provided on creation
- Repository uses 2-query pattern (staff query + IN UNNEST roles) to avoid N+1

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
- Domain model: [docs/rules/02-domain-model.md](../../../../docs/rules/02-domain-model.md) (cross-cell role validation rules)
- Database conventions: [docs/rules/03-database.md](../../../../docs/rules/03-database.md) (transaction patterns for multi-table staff+roles writes)
