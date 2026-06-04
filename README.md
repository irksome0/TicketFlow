# TicketFlow
Diploma project

## Vercel deployment

Frontend application is located in `ticketflow-web`.

Recommended Vercel project settings:

- Framework Preset: Next.js
- Root Directory: `ticketflow-web`
- Install Command: `npm install`
- Build Command: `npm run build`
- Output Directory: leave empty and let Vercel detect the Next.js output

Required frontend environment variable:

```env
NEXT_PUBLIC_API_URL=https://<render-service>.onrender.com/api/v1
```

For local frontend development, use `ticketflow-web/.env.local` based on
`ticketflow-web/.env.local.example`.

## Local Docker Run

For manual local testing, the whole MVP can be started from the repository root:

```bash
docker compose up --build
```

This starts PostgreSQL, the Go API, and the Next.js frontend with local defaults:

- frontend: `http://localhost:3000`;
- backend API: `http://localhost:8080/api/v1`;
- backend health check: `http://localhost:8080/health`;
- PostgreSQL: `localhost:5432`.

The root `.env` file is optional for local Docker usage. Create it from
`.env.example` only when ports, database credentials, or deployment-like settings
must be overridden.
