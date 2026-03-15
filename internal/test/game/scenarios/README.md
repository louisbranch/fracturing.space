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
local scn = Scenario.new("demo")
local dh = scn:system("DAGGERHEART")

scn:campaign({name = "Demo", system = "DAGGERHEART"})
dh:attack({actor = "Frodo", target = "Nazgul"})
```

Root alias convention:

- canonical semantic name: `scenario`
- preferred shorthand for scripts: `scn`
- avoid `scene` as the root alias to prevent collision with domain `scene` terminology

Core lifecycle steps remain on the root handle (`scn` above): campaign, participant/character setup, session lifecycle, and spotlight controls.

Interaction flow steps also live on the root handle. Use `as = "<participant alias>"`
inside a step table to run that write as a specific GM or player participant:

```lua
scn:interaction_start_player_phase({
  scene = "The Bridge",
  frame_text = "What do you do next?",
  characters = {"Aria", "Corin"},
  as = "Guide",
})
scn:interaction_post({
  as = "Rhea",
  summary = "Aria lunges for the rope.",
  characters = {"Aria"},
  yield = true,
})
```

Supported interaction root methods:

- `interaction_set_gm_authority`
- `interaction_set_active_scene`
- `interaction_start_player_phase`
- `interaction_post`
- `interaction_yield`
- `interaction_unyield`
- `interaction_accept_player_phase`
- `interaction_request_revisions`
- `interaction_end_player_phase`
- `interaction_pause_ooc`
- `interaction_post_ooc`
- `interaction_ready_ooc`
- `interaction_clear_ready_ooc`
- `interaction_resume_ooc`
- `interaction_expect`

Any scenario step may declare an expected failure and continue execution when
the runner sees a matching gRPC status:

```lua
scn:interaction_resume_ooc({
  as = "Guide",
  expect_error = {code = "FAILED_PRECONDITION", contains = "not paused"},
})
```

`code` is required. `contains` is optional and matches the gRPC status message.

`interaction_expect` now asserts scene player slots rather than the older
post/yield split. Slot entries may include `participant`, `summary`,
`characters`, `yielded`, `review_status`, `review_reason`, and
`review_characters`. Scene phase assertions also support `GM_REVIEW` for the
GM review/return step between player submission and the next GM-owned beat.

The same DSL shape is used for every system. Adding a new system should extend
the scenario system registry (`internal/tools/scenario/system_registry.go`)
rather than introducing a new top-level DSL object.

`scn:campaign` must declare `system` explicitly. Implicit default systems are
not supported.

Legacy root-level mechanic calls (for example `scene:attack(...)` or
`scn:attack(...)`) are intentionally rejected. Use
`scn:system("<SYSTEM_ID>"):attack(...)`.

## Comment Validation

When comment validation is enabled, each non-empty block that contains scenario step calls (for example, `scn:*` or `dh:*`) must start with a comment line (`-- ...`).

## Acceptance-first interaction specs

Acceptance-only scenario files are still allowed when the DSL does not yet
support a contract, but the current interaction corpus is expected to be
runtime-executable rather than placeholder-only.
