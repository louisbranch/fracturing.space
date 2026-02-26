# Fracturing.Space ![Coverage](../../raw/badges/coverage.svg)

Fracturing.Space is an open-source, server-authoritative engine for tabletop RPG campaigns modeled as deterministic, event-sourced state machines. It exposes gRPC and MCP interfaces for clients and automated agents.

## Why this is interesting

Fracturing.Space is built for tables experimenting with AI GMs or mixed human/AI facilitation. The event-driven core makes every action an ordered, deterministic event, which means:

- AI or programmatic GMs can reason over a full, authoritative history
- Outcomes can be replayed and audited, not guessed or reinterpreted
- Branching a campaign is safe and explicit, enabling "what if" timelines without rewriting history

## Quickstart (Docker)

Requires Docker + Docker Compose.

```sh
make bootstrap
```

Open `http://localhost:8080`.

This uses dev-only join-grant keys baked into `docker-compose.yml`. Replace them for any real deployment.

Service URLs, explicit compose commands, and runtime variants:

- [docs/running/quickstart.md](docs/running/quickstart.md)
- [docs/running/docker-compose.md](docs/running/docker-compose.md)
- [docs/running/local-dev.md](docs/running/local-dev.md)

## New contributor path

Recommended first path:

1. Start at [docs/index.md](docs/index.md).
2. Follow [docs/audience/contributors.md](docs/audience/contributors.md).
3. Pick a first edit target from [docs/audience/contributor-map.md](docs/audience/contributor-map.md).
4. Apply contribution workflow checks in [CONTRIBUTING.md](CONTRIBUTING.md).

If you are evaluating architecture before coding, start with [docs/architecture/overview.md](docs/architecture/overview.md) and then return to the contributor path above.

## Docs map

- Docs home: [docs/index.md](docs/index.md)
- Audience routing: [docs/audience/index.md](docs/audience/index.md)
- Running and setup: [docs/running/index.md](docs/running/index.md)
- Project architecture and domain docs: [docs/architecture/index.md](docs/architecture/index.md)

## Documentation lifecycle

Contributor entrypoints prioritize canonical docs. Temporary implementation plans and phase trackers are intentionally kept out of `docs/`.

## Project status

Early and experimental. APIs and internal structure may change.

<details>
<summary>Coverage Treemap</summary>

![Coverage Treemap](../../raw/badges/coverage-treemap.svg)

</details>

## Systems, content, and licensing

This repository focuses on infrastructure and mechanics, not on distributing copyrighted game content.

- Rules systems are implemented as mechanics only, without copyrighted text
- Names and mechanics are referenced for compatibility and interoperability purposes
- Contributors are responsible for ensuring that any submitted content complies with applicable licenses
