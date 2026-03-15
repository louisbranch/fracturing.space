# SQLC Workflow

The project uses [sqlc](https://sqlc.dev/) (v1.30.0+) to generate type-safe Go
code from SQL queries. This document describes the workflow for modifying
database queries or schema.

## Directory Layout

```
sqlc.yaml                                          # Config (repo root)
internal/services/game/storage/sqlite/
  migrations/
    events/       # Event store DDL (append-only journal)
    projections/  # Projection read-model DDL
    content/      # System-specific reference data DDL
  queries/        # SQL query files (*.sql)
    campaigns.sql
    characters.sql
    events.sql
    participants.sql
    sessions.sql
    invites.sql
    daggerheart.sql
    audit.sql
    snapshot.sql
    statistics.sql
  db/             # Generated output (DO NOT EDIT)
    db.go
    models.go
    *.sql.go
```

## Adding or Modifying Queries

1. **Edit the `.sql` query file** in `queries/`. Each query uses a sqlc
   annotation comment to control code generation:

   ```sql
   -- name: GetCampaign :one
   SELECT id, name, locale, ...
   FROM campaigns
   WHERE id = ?;

   -- name: ListCampaigns :many
   SELECT id, name, locale, ...
   FROM campaigns
   WHERE id > ?
   ORDER BY id
   LIMIT ?;

   -- name: PutCampaign :exec
   INSERT OR REPLACE INTO campaigns (id, name, locale, ...)
   VALUES (?, ?, ?, ...);
   ```

   Supported directives: `:one`, `:many`, `:exec`, `:execresult`.

2. **Run sqlc generate:**

   ```sh
   sqlc generate
   ```

   This regenerates all files in `db/`. Never hand-edit generated files.

3. **Update consuming code** if query signatures changed. The generated code
   lives in `db/` and is consumed by store implementations in sibling packages
   (`coreprojection/`, `eventjournal/`, `daggerheartprojection/`, etc.).

## Adding Schema Migrations

1. **Create a new `.sql` file** in the appropriate `migrations/` subdirectory.
   Use a numeric prefix for ordering (e.g., `004_add_column.sql`).

2. **Update queries** that reference new columns or tables.

3. **Run `sqlc generate`** to pick up the schema changes.

4. Migration files are applied at startup by the storage layer. The event store,
   projection store, and content store each manage their own migration set
   independently.

## Configuration

The `sqlc.yaml` at the repo root defines two SQL engines — one for the game
service and one for the auth service. Each engine specifies:

- `queries` -- directory of `.sql` query files
- `schema` -- one or more directories of DDL migration files
- `gen.go.package` -- output Go package name (`db`)
- `gen.go.out` -- output directory
- `emit_json_tags: true` -- generates JSON struct tags on model types
- `emit_empty_slices: true` -- initializes slice fields to `[]T{}` instead of
  `nil`

## Testing

Generated code in `db/` is excluded from unit test requirements and coverage
floors. The store packages that consume the generated code have their own
integration tests using in-memory SQLite databases.
