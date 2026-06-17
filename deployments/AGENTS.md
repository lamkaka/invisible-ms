# Deployments

This folder contains everything needed to run IMS locally with Docker Compose.

## Files

- `docker-compose.yml` — local stack: Spanner emulator, API, nginx
- `../apps/api/Dockerfile` — API application image (in apps/api)
- `../apps/api/migrations/` — Spanner DDL migrations (in apps/api)
- `Makefile` — common commands
- `.env.example` — required environment variables

## Quick start

```bash
cd deployments
cp .env.example .env
make up
```

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `GCP_SPANNER_PROJECT_ID` | `invisible-ms-local` | Yes | GCP project or emulator project ID |
| `GCP_SPANNER_INSTANCE_ID` | `invisible-ms-instance` | Yes | Spanner instance name |
| `GCP_SPANNER_DATABASE_ID` | `invisible-ms-db` | Yes | Spanner database name |
| `GCP_SPANNER_EMULATOR_HOST` | (empty) | For emulator | Spanner emulator host:port (e.g., localhost:9010) |
| `PORT` | `8080` | No | HTTP server port |
| `WEBHOOK_SECRET` | (empty) | For webhooks | Secret value for webhook authentication header |

## Run migrations only

```bash
make migrate
```
