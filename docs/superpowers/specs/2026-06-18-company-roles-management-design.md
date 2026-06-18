# Company Roles Management

## Overview

Add a dedicated way for managers to view, add, edit, and delete roles within a company. The feature closes gaps in the existing role API and adds a server-rendered HTML page that follows the current dashboard pattern.

## Goals

- Let managers manage roles through a web UI at `/roles?company_code=`.
- Add the missing REST endpoints the UI needs:
  - `GET /api/companies/{code}/roles` — list roles.
  - `PUT /api/companies/{code}/roles/{role}` — update a role's hourly rate.
- Prevent accidental deletion of a role that is assigned to staff.
- Enforce consistent role name validation (uppercase alphanumeric + underscores).

## Non-Goals

- Do not make `Role` a separate aggregate; it stays a value object inside `Company`.
- Do not apply rate changes retroactively; updated rates affect only future sessions.
- Do not block deletion based on activity logs or open check-ins.

## Background

The `company` cell already supports adding and removing roles via the REST API:

- `POST /api/companies/{code}/roles`
- `DELETE /api/companies/{code}/roles/{role}`

The dashboard cell already serves `/staff` and `/actions` HTML pages. There is no dedicated `/roles` page, and there are no endpoints to list roles or update a role's hourly rate.

## Architecture

The changes stay inside the `company` and `dashboard` cells.

### Company Cell Changes

| File | Change |
|------|--------|
| `company_domain.go` | Add `UpdateRole(name, hourlyRate)` and `ErrRoleAssigned`; validate role name format. |
| `company_service.go` | Add `ListRoles()` and `UpdateRole()`; guard `RemoveRole()` with staff-assignment check. |
| `company_repository.go` | Add `IsRoleAssigned(ctx, companyCode, roleName) bool`. |
| `company_controller.go` | Add handlers for `GET /api/companies/{code}/roles` and `PUT /api/companies/{code}/roles/{role}`. |

### Dashboard Cell Changes

| File | Change |
|------|--------|
| `dashboard_web_controller.go` | Add `GET /roles?company_code=` handler that renders `roles.html`. |
| `apps/web/templates/roles.html` | New template shell; Alpine.js fetches roles and calls the company endpoints. |
| `apps/web/templates/layout.html` | Add a "Roles" link to the navbar. |

## API Design

### List Roles

```
GET /api/companies/{code}/roles
```

Response `200 OK`:

```json
[
  { "name": "CLEANING", "hourly_rate": 15.0 },
  { "name": "DELIVERY", "hourly_rate": 18.5 }
]
```

### Update Role Rate

```
PUT /api/companies/{code}/roles/{role}
```

Request body:

```json
{ "hourly_rate": 20.0 }
```

Response `204 NoContent` on success.

### Delete Role (existing, with new guard)

```
DELETE /api/companies/{code}/roles/{role}
```

Response `204 NoContent` on success. Returns `409 Conflict` if the role is assigned to staff.

### Add Role (existing)

```
POST /api/companies/{code}/roles
```

Request body:

```json
{ "role_name": "SECURITY", "hourly_rate": 22.0 }
```

## Web UI

The `/roles` page follows the same pattern as `/staff` and `/actions`:

- The server renders a template shell with Alpine.js.
- Alpine.js fetches roles from `GET /api/companies/{code}/roles` and calls `POST`, `PUT`, and `DELETE` endpoints.
- Table of roles with inline edit and delete actions.
- Modal form to add a new role.
- Errors surface inline near the relevant action.

The navbar in `layout.html` gains a "Roles" link.

## Business Rules

- Role names must be non-empty, uppercase alphanumeric, and may contain underscores.
- Hourly rates must be zero or positive.
- Role names must be unique within a company.
- A role cannot be deleted while it is assigned to one or more staff members.
- Updating a role's hourly rate changes only the stored rate; past sessions keep their original cost.
- These rules apply to both the existing `AddRole` and the new `UpdateRole` paths.

## Error Handling

| Scenario | HTTP Status |
|----------|-------------|
| Invalid role name format | `400 Bad Request` |
| Negative hourly rate | `400 Bad Request` |
| Company not found | `404 Not Found` |
| Role not found | `404 Not Found` |
| Role already exists | `409 Conflict` |
| Role assigned to staff on delete | `409 Conflict` |

## Data Flow

### List Roles

1. Controller extracts `code` from URL.
2. Service loads the company via the repository.
3. Service returns the role list from `company.Roles`.

### Update Role Rate

1. Controller decodes `hourly_rate` from body.
2. Service loads the company.
3. Domain validates and applies the rate change.
4. Service persists the company via `repo.Update`.

### Delete Role

1. Controller extracts `code` and `role` from URL.
2. Service checks `repo.IsRoleAssigned`.
3. If assigned, return `ErrRoleAssigned` (`409 Conflict`).
4. Otherwise load company, call `company.RemoveRole`, and persist.

## Testing

- Domain tests for `UpdateRole` and role-name validation.
- Service tests for `ListRoles`, `UpdateRole`, and guarded `RemoveRole`.
- Controller tests for the new endpoints and the guarded delete.
- Dashboard web controller test for `/roles` page rendering.

## Dependencies

- The `company` cell exposes the new endpoints.
- The `dashboard` cell consumes the role endpoints for the web page.
- The `staff` cell is unaffected; it continues to validate roles through `CompanyService`.

## Open Questions

None. All scope decisions were confirmed during brainstorming.
