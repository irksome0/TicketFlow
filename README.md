# TicketFlow
Diploma project

## Vercel deployment

Frontend application is located in `ticketflow-web`.

Recommended Vercel project settings:

- Framework Preset: Next.js
- Root Directory: `ticketflow-web`
- Install Command: `npm install`
- Build Command: `npm run build`
- Output Directory: `.next`

Required frontend environment variable:

```env
NEXT_PUBLIC_API_URL=https://<render-service>.onrender.com/api/v1
```

For local frontend development, use `ticketflow-web/.env.local` based on
`ticketflow-web/.env.local.example`.
