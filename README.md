# Invisible Management System

Multi-tenant HR application for managing hourly staff (freelancers, contractors, part-time, shift staff). Staff check in and out via WhatsApp using keyword commands. The system tracks activity logs, computes hours and costs per role, and provides a management dashboard.

## Architecture

The application follows Cell-Based Domain-Driven Design (DDD) with Clean Architecture. Each bounded context is a self-contained cell with strict dependency rules.

```
                     +---------------+
                     |   Dashboard   |
                     |  (read-only)  |
                     +-------+-------+
                             |
            +----------------+----------------+
            |                |                |
     +------v------+  +-----v------+  +------v------+
     |   Company   |  |   Staff   |  |  Activity   |
     | (standalone)|  | -> company |  | -> staff   |
     +-------------+  +------------+  +-------------+
```

See [docs/rules/01-architecture.md](docs/rules/01-architecture.md) for the full architecture conventions.

## Tech Stack

- **Backend:** Go 1.26 with go-chi/chi/v5 router
- **Database:** Google Cloud Spanner (emulator for local development)
- **Frontend:** Server-rendered HTML + Alpine.js 3.x (CDN, no build step)
- **Messaging:** External webhook integration (WhatsApp via Waha gateway)
- **Containerization:** Docker Compose with Nginx reverse proxy
- **Testing:** Standard library testing with table-driven tests

## Quick Start (Docker)

The fastest way to get running is with Docker Compose, which starts a complete stack with Spanner emulator, database migrations, the Go API server, and Nginx reverse proxy.

```bash
git clone <repo-url>
cd invisible-ms
cp .env.example .env
make docker-up
```

The stack starts on **http://localhost:8888**. To stop:

```bash
make docker-down
```

### Build Docker Images

```bash
make docker-build
```

## Local Development (Without Docker)

1. Start the Spanner emulator:
   ```bash
   docker run -d --name spanner-emulator \
     -p 9010:9010 -p 9020:9020 \
     gcr.io/cloud-spanner-emulator/emulator
   ```

2. Set environment variables:
   ```bash
   export GCP_SPANNER_PROJECT_ID=invisible-ms-local
   export GCP_SPANNER_INSTANCE_ID=invisible-ms-instance
   export GCP_SPANNER_DATABASE_ID=invisible-ms-db
   export GCP_SPANNER_EMULATOR_HOST=localhost:9010
   export PORT=8080
   export WEBHOOK_SECRET=test-secret
   ```

3. Run migrations:
   ```bash
   make migrate
   ```

4. Start the server:
   ```bash
   make run
   ```

5. Seed test data (optional):
   ```bash
   cd apps/api && go run ./cmd/setup
   ```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile server binary to `bin/server` |
| `make run` | Run server locally with `go run` |
| `make test` | Run all tests |
| `make migrate` | Run database migrations locally |
| `make docker-build` | Build Docker images |
| `make docker-up` | Start full Docker stack |
| `make docker-down` | Stop all services |
| `make docker-logs` | Follow container logs |
| `make docker-restart` | Down then up |

## Project Status

MVP complete -- production ready for initial deployment.

### Completed Features

- Company CRUD with role catalog management
- Staff CRUD with role assignment validation
- WhatsApp webhook integration for check-in/check-out
- Activity log tracking with configurable action types
- Work session computation and cost calculation
- Dashboard with real-time stats
- Action type configuration UI and staff management UI
- Webhook authentication and atomic operations

### Known Limitations

- No authentication for dashboard/API access
- Overtime alert thresholds not yet implemented
- Only CHECK_IN/CHECK_OUT action types have built-in business logic
- No CSV/PDF export or email notifications
- Controller and repository integration tests pending
- List operations have N+1 query patterns (optimization opportunity)

## Documentation

- [AGENTS.md](AGENTS.md) — Agent instructions, cell guides, and quick-start mapping
- [docs/rules/01-architecture.md](docs/rules/01-architecture.md) — Architecture principles
- [docs/rules/02-domain-model.md](docs/rules/02-domain-model.md) — Domain model conventions
- [docs/rules/03-database.md](docs/rules/03-database.md) — Database conventions
- [docs/rules/04-api-and-webhook.md](docs/rules/04-api-and-webhook.md) — API and webhook conventions
- [docs/rules/05-testing.md](docs/rules/05-testing.md) — Testing strategy
- [docs/rules/06-development.md](docs/rules/06-development.md) — Development guidelines

For the full API endpoint inventory and webhook details, see [docs/rules/04-api-and-webhook.md](docs/rules/04-api-and-webhook.md).
