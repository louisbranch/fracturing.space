---
title: "Production"
parent: "Running"
nav_order: 4
---

# Production (Docker + Caddy)

Use Docker Compose for deployment. Configure real domains, TLS, and secrets.

## Steps

1. Copy `.env.production.example` to `.env` (or run `ENV_EXAMPLE=.env.production.example make bootstrap`).
2. Replace all dev secrets. Generate fresh keys and paste the output into `.env`:

```sh
docker compose --profile tools run --rm hmac-key
docker compose --profile tools run --rm join-grant-key
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
   - `FRACTURING_SPACE_WEBAUTHN_RP_ID` (must match `FRACTURING_SPACE_DOMAIN`)
   - `FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS` (e.g., `https://example.com`)
5. Pull and run:

```sh
docker compose pull
docker compose up -d
```

Minimal-intervention bootstrap: run `make bootstrap-prod` to copy the production template, generate missing keys, and start Compose. If you already have `.env`, use `make bootstrap`.

See [configuration](configuration.md) for the full environment matrix.
