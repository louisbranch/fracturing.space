---
title: "Production"
parent: "Running"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Production (Image-only Compose + Caddy)

Remote deployment uses [`docker-compose.production.yml`](../../docker-compose.production.yml).
It is image-only: the host pulls tagged images, Caddy is the only public entry
point, and runtime state lives in named Docker volumes instead of repo bind
mounts or host-relative paths.

## Deployment shape

- Caddy publishes `80` and `443` and terminates TLS.
- Internal services stay on the private Compose network.
- SQLite data lives under `/data` in the shared named volume.
- The Daggerheart reference corpus is seeded from a separate image into a named volume and mounted into `ai` at `/reference`.
- OpenViking runs through the repo-owned `openviking-sidecar` image and stores mirrored data in a named volume instead of `~/.openviking`.

## Prepare the env file

Start from the production template and generate the local-only secrets:

```sh
make prod-env
```

That command creates `.env.production` from
[`.env.production.example`](../../.env.production.example)
when needed and fills these values locally without requiring Compose tool
profiles on the server:

- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`
- `FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY`
- `FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY`
- `FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY`
- `FRACTURING_SPACE_AI_ENCRYPTION_KEY`
- `FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY`
- `FRACTURING_SPACE_OAUTH_RESOURCE_SECRET`

Fill the remaining required operator values in `.env.production`:

- `FRACTURING_SPACE_IMAGE_TAG`
- `FRACTURING_SPACE_DOMAIN`
- `FRACTURING_SPACE_DAGGERHEART_REFERENCE_IMAGE`
- `FRACTURING_SPACE_JAEGER_BASIC_AUTH`
- `FRACTURING_SPACE_OPENVIKING_OPENAI_API_KEY`

Optional:

- `FRACTURING_SPACE_CADDY_EMAIL`

`FRACTURING_SPACE_DAGGERHEART_REFERENCE_IMAGE` must point at an image that
contains the corpus under `/reference-src` with `index.json` at the root and
provides `sh` plus `cp` so the init container can seed the named volume.

`FRACTURING_SPACE_JAEGER_BASIC_AUTH` must be a full Caddy directive, not just a
username/password pair.

## Validate and start

Validate the rendered stack before the first deploy:

```sh
docker compose --env-file .env.production -f docker-compose.production.yml config
```

Start the stack:

```sh
docker compose --env-file .env.production -f docker-compose.production.yml pull
docker compose --env-file .env.production -f docker-compose.production.yml up -d
```

Or use the convenience target:

```sh
make bootstrap-prod
```

## Path policy

The production deployment path should not depend on checkout-relative or
user-home-relative filesystem layout.

Allowed in production:

- named Docker volumes
- fixed container paths such as `/data`, `/reference`, and `/openviking-data`

Not allowed in production:

- `./...` bind mounts
- `${HOME}`-based runtime paths
- repo checkout paths inside service configuration

See [configuration](configuration.md) for the full environment matrix.
