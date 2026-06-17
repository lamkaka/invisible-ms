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

## Run migrations only

```bash
make migrate
```
