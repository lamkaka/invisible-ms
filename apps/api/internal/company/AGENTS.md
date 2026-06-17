# Company Cell

## Purpose
Manages tenant companies and their role catalog. Companies are the top-level organizational unit. This cell also manages action type configuration (WhatsApp keyword → action type mappings).

## Owned Aggregates
- **Company** (aggregate root): `company_code` (PK), `company_name`, `roles` (map of Role value objects)
- **Role** (value object): `name`, `hourly_rate`
- **CompanyActionType** (value object): `action_type`, `keyword`, `is_system`

## File Inventory

| File | Responsibility |
|---|---|
| `company_domain.go` | Company aggregate, Role value object, CompanyActionType value object, validation/business rules |
| `company_service.go` | Orchestration: CRUD operations for companies, role management, action type management |
| `company_repository.go` | `CompanyRepository` interface + `SpannerCompanyRepository` adapter (companies + roles tables) |
| `company_action_type_repository.go` | `CompanyActionTypeRepository` interface + `SpannerCompanyActionTypeRepository` adapter (`company_action_types` table), including `SeedDefaults()` |
| `company_controller.go` | REST endpoints for company, role, and action type management |
| `company_domain_test.go` | Unit tests for domain logic |
| `company_service_test.go` | Service layer tests |
| `company_controller_test.go` | Controller HTTP tests |

## Inbound Dependencies
- `internal/shared` — error types, config

## Outbound Dependencies
- Exposes `CompanyService` to `staff` cell (role validation)
- Exposes `CompanyService.ListActionTypes()` to `activity` cell (keyword map building)
- Exposes `CompanyService.GetCompany()` to `activity` cell (role rate lookup for session cost)

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/companies` | List all companies |
| POST | `/api/companies` | Create company (seeds default action types) |
| GET | `/api/companies/{code}` | Get company details with roles |
| POST | `/api/companies/{code}/roles` | Add role to company |
| DELETE | `/api/companies/{code}/roles/{role}` | Remove role from company |
| GET | `/api/companies/{code}/action-types` | List action types |
| POST | `/api/companies/{code}/action-types` | Create custom action type |
| PUT | `/api/companies/{code}/action-types/{action}` | Update action type keyword |
| DELETE | `/api/companies/{code}/action-types/{action}` | Delete custom action type |

## Cell-Specific Business Rules
- `company_code` must be non-empty (used as tenant identifier)
- Role names and action type names must be uppercase alphanumeric with underscores
- Hourly rate cannot be negative
- System action types (`CHECK_IN`, `CHECK_OUT`) cannot be deleted; their keywords can only be changed
- Keywords must be unique within a company
- New companies automatically get default system action types seeded
- Roles are stored as `company_roles` child rows (interleaved in `companies`)

## Links
- Architecture conventions: [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
