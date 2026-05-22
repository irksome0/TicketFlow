# TicketFlow API

Backend REST API for the TicketFlow diploma project.

## Stack

- Go
- Gin
- GORM
- PostgreSQL
- JWT
- bcrypt
- Local filesystem storage for attachments

## Environment

Copy `.env.example` to `.env` for local development and set production values in the hosting provider environment.

Required variables:

| Variable | Purpose |
| --- | --- |
| `SERVER_PORT` | HTTP port, usually `8080` locally or provider-defined in production. |
| `GIN_MODE` | Use `release` in production. |
| `FRONTEND_ORIGIN` | Allowed frontend origin for CORS. Multiple origins can be comma-separated. |
| `DB_HOST` | PostgreSQL host. |
| `DB_PORT` | PostgreSQL port. |
| `DB_USER` | PostgreSQL user. |
| `DB_PASSWORD` | PostgreSQL password. |
| `DB_NAME` | PostgreSQL database name. |
| `DB_SSLMODE` | Use `require` for managed production PostgreSQL when required by provider. |
| `DB_TIMEZONE` | Use `UTC` for SLA consistency. |
| `JWT_SECRET` | Long random secret for signing JWT tokens. |
| `JWT_TTL_HOURS` | Access token lifetime in hours. |
| `SLA_TIMEZONE` | IANA timezone for SLA business calendar, for example `Europe/Kyiv`. |
| `SLA_BUSINESS_START` | Start of business hours in `HH:MM` format. |
| `SLA_BUSINESS_END` | End of business hours in `HH:MM` format. |
| `SLA_HOLIDAYS` | Optional comma-separated dates in `YYYY-MM-DD` format that are excluded from SLA time. |
| `SLA_HIGH_HOURS` | High-priority SLA limit in business hours. |
| `SLA_MEDIUM_HOURS` | Medium-priority SLA limit in business hours. |
| `SLA_LOW_HOURS` | Low-priority SLA limit in business hours. |

## Local Run

```bash
go mod download
go run ./cmd
```

## Docker Run

From the repository root:

```bash
cp .env.example .env
docker compose up --build
```

This starts:

- PostgreSQL on `localhost:5432`;
- TicketFlow API on `localhost:8080`;
- a local `ticketflow-api/uploads` directory mounted into the API container.

The root `.env` file is used only by Docker Compose and must not be committed.

Stop containers:

```bash
docker compose down
```

Remove the local PostgreSQL volume when a clean database is needed:

```bash
docker compose down -v
```

Health check:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## Deploy Notes

The API is deploy-ready for a single-instance MVP with PostgreSQL and local file storage.

Production settings:

- set `GIN_MODE=release`;
- set `JWT_SECRET` to a long random value;
- set `FRONTEND_ORIGIN` to the deployed Next.js frontend URL;
- set `DB_SSLMODE=require` if the PostgreSQL provider requires SSL;
- keep `DB_TIMEZONE=UTC`;
- configure persistent storage for the `uploads/` directory if the hosting platform does not preserve local disk between restarts.

## Security Notes

Implemented:

- JWT authentication with expiration;
- token invalidation through `token_version` after role changes;
- organization-based multi-tenant checks in handlers;
- role-based access control;
- SLA engine based on priority policies, business hours, weekends, holidays, status history, and pause statuses;
- attachment size and extension validation;
- path-safe attachment storage;
- health endpoint for deployment checks.

For a production B2B system, PostgreSQL Row Level Security can be added as an additional database-level defense. The current diploma MVP keeps tenant isolation in the application layer and documents RLS as a recommended production hardening measure.
