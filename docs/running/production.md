---
title: "Production"
parent: "Running"
nav_order: 4
---

# Production (Docker + Caddy)

Use Docker Compose for deployment. Configure real domains, TLS, and secrets.

## Steps

1. Copy `.env.example` to `.env`.
2. Replace all dev secrets. Generate fresh keys and paste the output into `.env`:

```sh
go run ./cmd/hmac-key
go run ./cmd/join-grant-key
```

3. Set these values in `.env`:
   - `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`
   - `FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY`
   - `FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY`
4. Set routing and TLS:
   - `FRACTURING_SPACE_DOMAIN`
   - `FRACTURING_SPACE_PUBLIC_SCHEME=https`
   - `FRACTURING_SPACE_PUBLIC_PORT=` (empty)
   - `FRACTURING_SPACE_CADDY_AUTO_HTTPS=on`
5. Pull and run:

```sh
docker compose pull
docker compose up -d
```

See [configuration](configuration.md) for the full environment matrix.
