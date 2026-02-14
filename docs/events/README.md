# Event Catalog

## Regenerating the catalog
Run the generator from the repo root:

```bash
go generate ./internal/services/game/domain/campaign/event
```

This writes the [event catalog](event-catalog.md) using the Go source of core and Daggerheart events.

## CI check
The Go tests workflow regenerates the catalog and fails if the [event catalog](event-catalog.md) is out of date.
