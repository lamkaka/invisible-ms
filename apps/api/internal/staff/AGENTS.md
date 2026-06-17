# Staff Cell

## Purpose
Manages workers (staff members) within companies. Handles staff creation, role assignment, and identification via phone number.

## Owned Aggregates
- **Staff** (aggregate root): `staff_id` (UUID, PK), `phone_number`, `name`, `company_code` (FK to Company), `assigned_roles` ([]string), `is_active`

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

| Method | Path | Description |
|---|---|---|
| GET | `/api/staff?company_code=` | List staff (company_code required) |
| POST | `/api/staff` | Create staff with optional roles |
| GET | `/api/staff/{id}` | Get staff details |
| POST | `/api/staff/{id}/roles` | Assign role to staff |
| DELETE | `/api/staff/{id}/roles/{role}` | Unassign role from staff |

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
