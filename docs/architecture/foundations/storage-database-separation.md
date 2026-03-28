---
title: "Storage database separation"
parent: "Foundations"
nav_order: 8
---

# Storage Database Separation

The game service uses three separate SQLite databases, each managing a distinct
data concern. This separation is intentional and driven by operational,
integrity, and lifecycle requirements.

## The Three Databases

### Events (`game-events.db`)

The append-only event journal — the single source of truth for all domain state.

- **Write pattern:** append-only inserts of domain events.
- **Integrity:** chain-hash verification on boot ensures the journal has not
  been tampered with or truncated.
- **WAL mode:** enabled for concurrent read access during event replay.
- **Backup story:** this database alone is sufficient to reconstruct the full
  system state. Losing the other two databases is recoverable; losing events is
  not.

### Projections (`game-projections.db`)

Materialized read views derived from events, serving API queries.

- **Write pattern:** upserts driven by projection handlers
  (`projection.Applier`). Writes are coordinated through exactly-once
  idempotency checkpoints to prevent duplicate application.
- **Derivable:** can be rebuilt from scratch by replaying the event journal.
  This makes the projections database a cache of computed state, not a primary
  data store.
- **Schema evolution:** projection schema changes can be applied by dropping and
  rebuilding from events, which avoids complex migration logic for read models.

### Content (`game-content.db`)

System-specific reference data (e.g., Daggerheart catalog entries) that enriches
projection reads.

- **Write pattern:** seeded at startup or updated independently of domain
  events.
- **Lifecycle:** content data is not derived from events. It represents external
  reference material that game systems need for rendering (card catalogs,
  ability definitions, etc.).
- **Isolation:** keeping content separate prevents a content schema change from
  affecting event or projection availability.

## Why Separate Databases?

### Operational isolation

Each database has different backup, recovery, and lifecycle characteristics.
Events require point-in-time backup guarantees. Projections can be dropped and
rebuilt. Content can be reseeded. Mixing these concerns in one database would
force a single backup/recovery strategy that satisfies the strictest
requirement.

### Write contention

SQLite serializes writes within a single database. Separating event appends
(high-frequency, latency-sensitive) from projection upserts (batch-oriented)
and content seeding (startup-only) prevents write contention between unrelated
workloads.

### Schema independence

Event, projection, and content schemas evolve at different rates and for
different reasons. Separate databases allow each to migrate independently
without cross-concern coordination.

### Integrity boundaries

The event journal's chain-hash integrity verification is a database-level
concern. Keeping events isolated means integrity checks don't scan unrelated
tables, and the verification boundary is clean.

## Cross-Database Query Prevention

The three databases are opened as separate `*sql.DB` connections through
distinct backend types (`eventBackend`, `projectionBackend`, `contentBackend`)
in the `storageBundle`. Go's type system prevents accidentally passing the wrong
connection to a store constructor. There are no cross-database JOINs or
attached-database patterns.

## Configuration

Database paths default to `data/game-events.db`, `data/game-projections.db`,
and `data/game-content.db`. They can be overridden via environment variables:

- `FRACTURING_SPACE_GAME_EVENTS_DB_PATH`
- `FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`
- `FRACTURING_SPACE_GAME_CONTENT_DB_PATH`
