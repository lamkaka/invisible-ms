# API & Webhook Conventions

## HTTP Status Code Mapping

Domain errors are translated to HTTP status codes at the controller layer:

| Domain Error | HTTP Status Code |
|---|---|
| `shared.ErrNotFound` | 404 Not Found |
| `shared.ErrAlreadyExists` | 409 Conflict |
| `shared.ErrInvalidInput` | 400 Bad Request |
| Internal/DB errors | 500 Internal Server Error |

## Webhook Security

- All webhooks require `X-Webhook-Secret` header
- Secret is loaded from `WEBHOOK_SECRET` environment variable
- Controller validates secret using constant-time comparison before processing
- Returns 401 Unauthorized if secret is missing or invalid

## Controller Layer Responsibilities

Controllers (`*_controller.go`):
- Parse HTTP requests (query params, path vars, JSON body)
- Call the appropriate service method
- Translate domain errors to HTTP status codes
- Set response headers (Content-Type, status code)
- Encode response bodies (JSON or HTML template)

Controllers must NOT contain business logic.

## API Endpoint Inventory

### Webhook
- `POST /webhook/message` — receives `{ phone, message, company_code }`

### Company Management
- `GET /api/companies` — list all companies
- `POST /api/companies` — create company
- `GET /api/companies/{code}` — get company details
- `POST /api/companies/{code}/roles` — add role to company
- `DELETE /api/companies/{code}/roles/{role}` — remove role from company
- `GET /api/companies/{code}/action-types` — list action types
- `POST /api/companies/{code}/action-types` — create custom action type
- `PUT /api/companies/{code}/action-types/{action}` — update action type keyword
- `DELETE /api/companies/{code}/action-types/{action}` — delete custom action type

### Staff Management
- `GET /api/staff?company_code=` — list staff (company_code required)
- `POST /api/staff` — create staff
- `GET /api/staff/{id}` — get staff details
- `POST /api/staff/{id}/roles` — assign role to staff
- `DELETE /api/staff/{id}/roles/{role}` — unassign role from staff

### Activity
- `GET /api/activities?staff_id=&company_code=&from=&to=` — list activity logs
- `GET /api/activities/sessions?company_code=&from=&to=` — list computed work sessions

### Dashboard (API)
- `GET /api/dashboard/stats?company_code=` — aggregated JSON stats

### Dashboard (Web Pages)
- `GET /dashboard?company_code=` — HTML dashboard page
- `GET /staff` — HTML staff management page
- `GET /actions` — HTML action type management page
