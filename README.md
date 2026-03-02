# Fracturing.Space ![Coverage](../../raw/badges/coverage.svg)

Fracturing.Space is an open-source, server-authoritative engine for tabletop RPG campaigns modeled as deterministic, event-sourced state machines. It exposes gRPC and MCP interfaces for clients and automated agents.

## Start here

- Docs home: [docs/index.md](docs/index.md)
- Contribution workflow: [CONTRIBUTING.md](CONTRIBUTING.md)

## Quickstart (Docker)

Requires Docker + Docker Compose.

```sh
make bootstrap
```

Open `http://localhost:8080`.

For detailed runtime paths:

- [docs/running/quickstart.md](docs/running/quickstart.md)
- [docs/running/docker-compose.md](docs/running/docker-compose.md)
- [docs/running/local-dev.md](docs/running/local-dev.md)

## Onboarding paths

- User path (player/GM evaluation): [docs/audience/users.md](docs/audience/users.md)
- Contributor path: [docs/audience/contributors.md](docs/audience/contributors.md)
- Go developer path: [docs/audience/go-developers.md](docs/audience/go-developers.md)
- Integrator path (MCP/clients): [docs/audience/integrators.md](docs/audience/integrators.md)
- System developer path: [docs/audience/system-developers.md](docs/audience/system-developers.md)
- Translator path: [docs/audience/translators.md](docs/audience/translators.md)

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
