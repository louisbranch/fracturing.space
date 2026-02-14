# Testability Practices

## Intent

Code that can't be tested can't be safely changed. This document codifies the dependency injection and constructor patterns already working in the well-tested domain layer (95%+ coverage), so new code follows them from the start.

## Accept dependencies, don't create them

The single most important pattern: constructors accept their dependencies rather than creating them internally.

**Good** — domain layer does this already:

```go
// Applier accepts store interfaces; tests pass in-memory fakes.
type Applier struct {
    Campaign    CampaignStore
    Daggerheart DaggerheartStore
}
```

**Production wiring** — creates real dependencies internally (fine for production, but not testable in isolation):

```go
// NewRunner dials a real gRPC server — production code calls this.
func NewRunner(ctx context.Context, cfg Config) (*Runner, error) {
    conn, err := grpc.NewClient(cfg.GRPCAddr, ...)
    // ... creates all clients from conn
}
```

To make this testable, add a separate internal constructor that accepts pre-built dependencies (see below).

## Constructor patterns

Use two constructors when a type needs complex setup for production but simple injection for tests:

```go
// Production constructor — creates real dependencies.
func NewRunner(ctx context.Context, cfg Config) (*Runner, error) {
    conn, err := grpc.NewClient(cfg.GRPCAddr, ...)
    env := scenarioEnv{
        campaignClient: gamev1.NewCampaignServiceClient(conn),
        // ...
    }
    return newRunner(env, auth, cfg), nil
}

// Internal constructor — accepts pre-built dependencies.
func newRunner(env scenarioEnv, auth *MockAuth, cfg Config) *Runner {
    return &Runner{env: env, auth: auth, ...}
}
```

Tests call the internal constructor directly with fakes. The production constructor is thin wiring that calls the internal one.

## Interface boundaries

Accept interfaces at dependency boundaries. Concrete types stay behind them.

**Stores**: Define small interfaces for what you need. The projection layer demonstrates this well — `CampaignStore`, `DaggerheartStore`, etc. are each a handful of methods.

**gRPC clients**: Protoc-generated clients are already interfaces (`gamev1.CampaignServiceClient`). Use them directly — no need to wrap.

**Simple dependencies**: Use function injection when the dependency is a single function:

```go
type Runner struct {
    clock func() time.Time   // injectable, defaults to time.Now
    rand  func(int) int      // injectable, defaults to math/rand
}
```

## Fake stores in test files

Keep test doubles in `*_test.go` files, scoped to the package under test.

```go
type fakeCampaignStore struct {
    campaigns map[string]campaign.Campaign
}

func newFakeCampaignStore() *fakeCampaignStore {
    return &fakeCampaignStore{campaigns: map[string]campaign.Campaign{}}
}

func (f *fakeCampaignStore) Get(ctx context.Context, id string) (campaign.Campaign, error) {
    c, ok := f.campaigns[id]
    if !ok {
        return campaign.Campaign{}, storage.ErrNotFound
    }
    return c, nil
}
```

Guidelines:
- **Minimal**: Only implement methods the tests actually call. Return `unimplemented` for the rest.
- **In-memory maps**: Simple and fast. No real IO.
- **Configurable errors**: Use function fields for methods where tests need to control the error path.

## What not to test

Not all code needs unit tests. Skip:

- **Production wiring** — Thin functions that just plug dependencies together (`main()`, `NewServer(cfg)` that dials and returns). Verified by integration tests.
- **Generated code** — Proto stubs, sqlc queries, templ output. Excluded from coverage via `COVER_EXCLUDE_REGEX` in `Makefile`.
- **CLI entrypoints** — `func main()` and flag parsing. Test the logic they call, not the wiring.

Focus unit tests on logic: validation, state transitions, request building, error handling.

## Related docs

- [Testing Coverage Policy](testing-coverage.md) — TDD workflow and CI non-regression gate
- [Architecture](architecture.md) — Service boundaries and layers
