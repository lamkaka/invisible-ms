# API & Webhook Conventions

## HTTP Status Code Mapping

| Domain Error | HTTP Status Code |
|---|---|
| `shared.ErrNotFound` | 404 Not Found |
| `shared.ErrAlreadyExists` | 409 Conflict |
| `shared.ErrInvalidInput` | 400 Bad Request |
| Internal/DB errors | 500 Internal Server Error |

## Webhook Security

- All webhooks require the `X-Webhook-Secret` header.
- The secret is loaded from the `WEBHOOK_SECRET` environment variable.
- Validate the secret with constant-time comparison.
- Return 401 Unauthorized if the secret is missing or invalid.

## Controller Responsibilities

Controllers (`*_controller.go`) handle:

- Parsing HTTP requests (query params, path variables, JSON body)
- Calling the appropriate service method
- Translating domain errors to HTTP status codes
- Setting response headers
- Encoding response bodies (JSON or HTML template)

Controllers must not contain business logic.

## Per-Cell Endpoints

- Company, role, and action type endpoints live in [`apps/api/internal/company/AGENTS.md`](../../apps/api/internal/company/AGENTS.md).
- Staff endpoints live in [`apps/api/internal/staff/AGENTS.md`](../../apps/api/internal/staff/AGENTS.md).
- Webhook and activity endpoints live in [`apps/api/internal/activity/AGENTS.md`](../../apps/api/internal/activity/AGENTS.md).
- Dashboard endpoints live in [`apps/api/internal/dashboard/AGENTS.md`](../../apps/api/internal/dashboard/AGENTS.md).
