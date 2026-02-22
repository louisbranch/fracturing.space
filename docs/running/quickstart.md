---
title: "Quickstart"
parent: "Running"
nav_order: 1
---

# Quickstart (Docker)

Requires Docker + Docker Compose.

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml up --build
```

For a minimal-intervention bootstrap that creates `.env`, generates missing keys, and starts the stack:

```sh
make bootstrap
# or
./scripts/bootstrap.sh
```

Open `http://localhost:8080`.

Service URLs and Docker Compose details: [docker-compose.md](docker-compose.md).
If you want the Go dev workflow instead, see [local-dev.md](local-dev.md).

Notes:

- Dev-only join-grant keys are baked into `docker-compose.yml`. Replace for real deployments.
- Stop with `Ctrl+C`. To remove volumes: `docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml down -v`.
