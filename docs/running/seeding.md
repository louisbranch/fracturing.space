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
# Terminal 1: Start watcher-managed local services
make up

# Terminal 2: Run seeding commands
make seed
```

Using direct Go commands:

```bash
# Terminal 1: Start the game server
go run ./cmd/game

# Terminal 2: Run seeding commands
make seed
```

Using Compose:

```bash
# Terminal 1: Start the game + auth services
docker compose up -d game auth

# Terminal 2: Run seeding commands
docker compose --profile tools run --rm seed
```

## Catalog Content Import

Use the catalog importer to load Daggerheart content into the SQLite catalog database.

```bash
make catalog-importer
```

Compose:

```bash
docker compose --profile tools run --rm catalog-importer
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Directory containing locale subfolders | required |
| `-db-path` | Content database path | `data/game-content.db` |
| `-base-locale` | Base locale used for catalog data | `en-US` |
| `-dry-run` | Validate without writing to the database | false |

## Static Fixtures (JSON-based)

Run predefined scenarios from JSON fixture files:

```bash
make seed        # Run all scenarios with verbose output
make seed-fresh  # Reset DB and reseed
```

Compose:

```bash
docker compose --profile tools run --rm seed
docker compose --profile tools run --rm seed -- -scenario example
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-grpc-addr` | game server address | `localhost:8080` |
| `-auth-addr` | auth server address (uses `FRACTURING_SPACE_AUTH_ADDR` when set) | `localhost:8083` |
| `-scenario` | Run specific scenario | all |
| `-list` | List available scenarios | - |
| `-v` | Verbose output | false |

### Adding Scenarios

Create JSON files in `internal/test/integration/fixtures/seed/`.

## Dynamic Generation

Generate diverse, randomized test data with reproducible seeds:

```bash
make seed-generate         # Demo preset (rich single campaign)
make seed-variety          # 8 campaigns with varied statuses/modes
make seed-generate-fresh   # Reset DB and generate demo data
```

Compose:

```bash
docker compose --profile tools run --rm seed -- -generate -preset=demo -v
docker compose --profile tools run --rm seed -- -generate -preset=variety -v
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-generate` | Enable dynamic generation mode | false |
| `-preset` | Generation preset | `demo` |
| `-campaigns` | Override number of campaigns | preset default |
| `-seed` | RNG seed for reproducibility (0 = random) | 0 |
| `-grpc-addr` | game server address | `localhost:8080` |
| `-auth-addr` | auth server address (uses `FRACTURING_SPACE_AUTH_ADDR` when set) | `localhost:8083` |
| `-v` | Verbose output | false |

### Presets

| Preset | Campaigns | Description |
|--------|-----------|-------------|
| `demo` | 1 | Rich single campaign with 3 players, 5-6 characters, 1 active session, 10-20 events |
| `variety` | 8 | Mixed statuses (DRAFT/ACTIVE/COMPLETED/ARCHIVED) and GM modes (HUMAN/AI/HYBRID) |
| `session-heavy` | 2 | Full parties with 5 sessions each, 50+ events |
| `stress-test` | 50 | Minimal campaigns for load testing |

### Examples

```bash
# Generate 3 campaigns using variety preset settings
go run ./cmd/seed -generate -preset=variety -campaigns=3 -v

# Generate with a specific seed for reproducibility
go run ./cmd/seed -generate -preset=demo -seed=12345 -v

# Re-run the same seed to get identical data
go run ./cmd/seed -generate -preset=demo -seed=12345 -v
```

### Reproducibility

The generator uses a seeded random number generator. Running with the same `-seed` value produces identical data. If no seed is specified, a random seed is chosen and printed to stderr for later reproduction:

```
Using seed: 1707234567890123456
```

Emails are uniquified within a run to satisfy auth email uniqueness constraints. When duplicates occur, the generator appends a numeric suffix (for example, `alex@example.com` -> `alex-2@example.com`).

### Entity Variations

The dynamic generator creates diverse test data:

| Entity | Variations |
|--------|------------|
| Campaign | DRAFT, ACTIVE, COMPLETED, ARCHIVED statuses; HUMAN/AI/HYBRID GM modes |
| Participant | GM + Players; HUMAN/AI controllers (20% AI chance) |
| Character | PC/NPC kinds; PCs assigned to player participants |
| Session | ACTIVE/ENDED statuses; named with sequence numbers |
| Event | NOTE_ADDED events with random content |
