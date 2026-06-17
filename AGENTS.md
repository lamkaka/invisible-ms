# IMS — Hourly Staff Management System

IMS is a multi-tenant HR application for managing hourly staff (freelancers, contractors, part-time, shift workers). Workers check in/out via WhatsApp using keyword commands. The system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

## Tech Stack

- **Backend:** Go (Golang)
- **Database:** Google Cloud Spanner
- **Frontend:** Server-rendered HTML + Alpine.js (CDN, no build step)
- **Messaging:** External webhook integration (WhatsApp/Waha layer is external)
- **Architecture:** Domain-Driven Design (DDD) + Clean Architecture + Cell-Based Architecture
- **Router:** go-chi/chi/v5
- **Go version:** 1.22+

## Architecture & Conventions

| Document | What it covers |
|---|---|
| [docs/rules/01-architecture.md](docs/rules/01-architecture.md) | DDD, Clean Architecture, cell-based architecture, file naming (`*_controller.go`), code organization, DI |
| [docs/rules/02-domain-model.md](docs/rules/02-domain-model.md) | Aggregates, value objects, session computation, business rules |
| [docs/rules/03-database.md](docs/rules/03-database.md) | Spanner schema, migration conventions, transaction patterns, query patterns |
| [docs/rules/04-api-and-webhook.md](docs/rules/04-api-and-webhook.md) | HTTP status codes, webhook security (X-Webhook-Secret), endpoint inventory |
| [docs/rules/05-testing.md](docs/rules/05-testing.md) | Testing strategy by layer, mock conventions, file naming |
| [docs/rules/06-development.md](docs/rules/06-development.md) | Error handling, configuration, performance, pitfalls |

## Cell Guides

| Cell | AGENTS.md | Responsibility |
|---|---|---|
| `shared` | [apps/api/internal/shared/AGENTS.md](apps/api/internal/shared/AGENTS.md) | Config, errors, middleware, SQL utilities |
| `company` | [apps/api/internal/company/AGENTS.md](apps/api/internal/company/AGENTS.md) | Companies, roles, action type configuration |
| `staff` | [apps/api/internal/staff/AGENTS.md](apps/api/internal/staff/AGENTS.md) | Staff management, role assignment |
| `activity` | [apps/api/internal/activity/AGENTS.md](apps/api/internal/activity/AGENTS.md) | Activity logs, webhook processing, session computation |
| `dashboard` | [apps/api/internal/dashboard/AGENTS.md](apps/api/internal/dashboard/AGENTS.md) | Aggregated stats, HTML pages (CQRS read model) |

## Deployments

See [deployments/AGENTS.md](deployments/AGENTS.md) for Docker Compose, migrations, and local development setup.

## Quick-Start Mapping

| If you are editing... | Read first |
|---|---|
| A domain model | Cell `AGENTS.md` |
| A service or use case | Cell `AGENTS.md` + `docs/rules/01-architecture.md` |
| A controller / HTTP handler | Cell `AGENTS.md` + `docs/rules/04-api-and-webhook.md` |
| A repository / Spanner query | Cell `AGENTS.md` + `docs/rules/03-database.md` |
| A test file | Cell `AGENTS.md` + `docs/rules/05-testing.md` |
| The main server wiring | `apps/api/cmd/server/main.go` |
| A migration file | `docs/rules/03-database.md` + owning Cell `AGENTS.md` |
| Config or env vars | `docs/rules/06-development.md` |
| Multiple or cross-cutting | `docs/rules/01-architecture.md` first |

## MVP Scope

- No authentication (add later)
- WhatsApp integration via external webhook (Waha layer is external)
- Basic dashboard with today's stats
- Company and staff management via REST API
- Check-in/check-out via WhatsApp keywords

## Future Enhancements

- Authentication (Google OAuth or email/password)
- Additional action types (BREAK, OVERTIME, TASK_COMPLETE)
- Export reports (CSV, PDF)
- Email notifications
- Mobile app for managers
- Staff self-service portal (view own hours)
