# Fracturing.Space ![Coverage](../../raw/badges/coverage.svg)

Fracturing.Space is an open-source, server-authoritative engine for tabletop RPG campaigns modeled as deterministic, event-sourced state machines. It exposes gRPC and MCP interfaces for clients and automated agents.

## Why this is interesting

Fracturing.Space is built for tables experimenting with AI GMs or mixed human/AI facilitation. The event-driven core makes every action an ordered, deterministic event, which means:

- AI or programmatic GMs can reason over a full, authoritative history
- Outcomes can be replayed and audited, not guessed or reinterpreted
- Branching a campaign is safe and explicit, enabling “what if” timelines without rewriting history

## Quickstart (Docker)

Requires Docker + Docker Compose.

```sh
docker compose up --build
```

Open `http://localhost:8080`.

This uses dev-only join-grant keys baked into `docker-compose.yml`. Replace them for any real deployment.

Service URLs and Docker Compose details: [docs/running/docker-compose.md](docs/running/docker-compose.md).

## Docs map

- Docs home: [docs/index.md](docs/index.md)
- Audience guides: [docs/audience/index.md](docs/audience/index.md)
- Contributors: [docs/audience/contributors.md](docs/audience/contributors.md)
- Operators: [docs/audience/operators.md](docs/audience/operators.md)
- Clients and tooling: [docs/audience/clients.md](docs/audience/clients.md)
- System designers: [docs/audience/system-designers.md](docs/audience/system-designers.md)
- Project overview: [docs/project/overview.md](docs/project/overview.md)

## Project status

Early and experimental. APIs and internal structure may change.

## Systems, content, and licensing

This repository focuses on infrastructure and mechanics, not on distributing copyrighted game content.

- Rules systems are implemented as mechanics only, without copyrighted text
- Names and mechanics are referenced for compatibility and interoperability purposes
- Contributors are responsible for ensuring that any submitted content complies with applicable licenses
