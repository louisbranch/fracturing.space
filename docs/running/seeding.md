---
title: "Seeding"
parent: "Running"
nav_order: 6
---

# Seeding the Development Database

This guide covers how to populate the development database with demo data for local testing.

Seeding is a developer tool that calls the game service APIs; it is not a standalone service.

## Prerequisites

The game server must be running before seeding:

```bash
# Terminal 1: Start devcontainer + watcher-managed local services
make up

# If you run make seed from the host side (outside the devcontainer terminal),
# ensure 8090 and 8091 are forwarded in `.devcontainer/devcontainer.json`
# for social and listing.

# Terminal 2: Run seeding commands
make seed
```

Using direct Go commands:

```bash
# Terminal 1: Start the required services
go run ./cmd/game
go run ./cmd/auth
go run ./cmd/listing
go run ./cmd/social

# Terminal 2: Run seeding commands
make seed
```

Using Compose:

```bash
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml"

# Terminal 1: Start the required services
$COMPOSE up -d game auth listing social

# Terminal 2: Run seeding commands
$COMPOSE --profile tools run --rm seed
```

## Catalog Content Import

Use the catalog importer to load Daggerheart content into the SQLite catalog database.

```bash
make catalog-importer
```

Compose:

```bash
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml"
$COMPOSE --profile tools run --rm catalog-importer
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory containing locale subfolders | required |
| `-db-path` | Content database path | `data/game-content.db` |
| `-base-locale` | Base locale used for catalog data | `en-US` |
| `-dry-run` | Validate without writing to the database | false |

## Recommended local seeding flow (idempotent)

Use the declarative local-dev manifest by default. This is the recommended path for most dev workflows:

```bash
make seed        # Seed local-dev dataset (idempotent)
make seed-fresh  # Reset DB + reseed local-dev dataset
```

Compose:

```bash
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml"
$COMPOSE --profile tools run --rm seed
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-manifest` | Path to declarative manifest JSON (local-dev only) | `internal/tools/seed/manifests/local-dev.json` |
| `-seed-state` | Path to idempotent state file | `.tmp/seed-state/local-dev.state.json` |
| `-grpc-addr` | game server address | `game:8082` |
| `-auth-addr` | auth server address (uses `FRACTURING_SPACE_AUTH_ADDR` when set) | `auth:8083` |
| `-social-addr` | social server address | `social:8090` |
| `-listing-addr` | listing server address | `listing:8091` |
| `-v` | Verbose output | false |

For any non-local environments, avoid running `seed` against production services. The command is intentionally restricted to the local-dev manifest in this workflow.
## Manifest seeding entity coverage

The declarative seeder supports:

| Entity | Service |
|--------|---------|
| User identities | `auth.v1.AuthService` |
| Public profiles | `social.v1.SocialService` |
| Contacts | `social.v1.SocialService` |
| Campaigns | `game.v1.CampaignService` |
| Participants | `game.v1.ParticipantService` |
| Characters + controls | `game.v1.CharacterService` |
| Sessions | `game.v1.SessionService` |
| Forks | `game.v1.ForkService` |
| Listings | `listing.v1.CampaignListingService` |

Account profiles are intentionally excluded from seed manifests and are not written during declarative seeding.
