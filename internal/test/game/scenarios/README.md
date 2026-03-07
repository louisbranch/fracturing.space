# Scenario Layout

Scenario scripts are grouped by system:

- `systems/<system_id>/*.lua`

Example:

- `systems/daggerheart/basic_flow.lua`

Manifests are stored under:

- `manifests/*.txt`

Manifest entries are path-based and must match scenario paths relative to this directory.

Example smoke entry:

- `systems/daggerheart/basic_flow.lua`

## DSL Rule

System mechanics must be invoked through an explicit system handle:

```lua
local scene = Scenario.new("demo")
local dh = scene:system("DAGGERHEART")

scene:campaign({name = "Demo", system = "DAGGERHEART"})
dh:attack({actor = "Frodo", target = "Nazgul"})
```

Core lifecycle steps remain on `scene` (campaign, participant/character setup, session lifecycle, spotlight controls).

The same DSL shape is used for every system. Adding a new system should extend
the scenario system registry (`internal/tools/scenario/system_registry.go`)
rather than introducing a new top-level DSL object.

`scene:campaign` must declare `system` explicitly. Implicit default systems are
not supported.

Legacy scene-level mechanic calls (for example `scene:attack(...)`) are
intentionally rejected. Use `scene:system("<SYSTEM_ID>"):attack(...)`.

## Comment Validation

When comment validation is enabled, each non-empty block that contains scenario step calls (for example, `scene:*` or `dh:*`) must start with a comment line (`-- ...`).
