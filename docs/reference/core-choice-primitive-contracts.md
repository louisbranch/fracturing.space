---
title: "Core Choice Primitive — Contracts"
parent: "Reference"
nav_order: 13
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Core Choice Primitive — Contracts

Detailed payload contracts, DSL expression, and scenario gap resolution for the [core choice primitive](../architecture/systems/core-choice-primitive.md).

## Payload Contracts

### `choice.present` → `choice.presented`

```go
type ChoicePresentPayload struct {
    ChoiceID   string           `json:"choice_id"`
    ChoiceType string           `json:"choice_type"`
    Label      string           `json:"label"`
    Options    []ChoiceOption   `json:"options"`
    GatePolicy ChoiceGatePolicy `json:"gate_policy"`
    Metadata   map[string]any   `json:"metadata,omitempty"`
}

type ChoiceOption struct {
    ID       string         `json:"id"`
    Label    string         `json:"label"`
    Effects  []ChoiceEffect `json:"effects,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
}

type ChoiceEffect struct {
    Type      string          `json:"type"`
    EntityType string         `json:"entity_type"`
    EntityID  string          `json:"entity_id"`
    SystemID  string          `json:"system_id,omitempty"`
    SystemVer string          `json:"system_version,omitempty"`
    PayloadJSON json.RawMessage `json:"payload"`
}

type ChoiceGatePolicy struct {
    Blocking bool `json:"blocking"`
}
```

### `choice.select` → `choice.selected`

```go
type ChoiceSelectPayload struct {
    ChoiceID   string `json:"choice_id"`
    SelectedID string `json:"selected_id"`
}

type ChoiceSelectedPayload struct {
    ChoiceID   string         `json:"choice_id"`
    SelectedID string         `json:"selected_id"`
    Effects    []ChoiceEffect `json:"effects"`
}
```

### Session fold state

```go
type OpenChoice struct {
    ChoiceID   string
    ChoiceType string
    Options    []ChoiceOption
    GateID     string // empty if non-blocking
}
```

### Causation chain

- `choice.presented` → `CausationID` points to triggering event (roll result, Fear spend).
- `choice.selected` → `CausationID` points to `choice.presented`.
- Consequence events → `CausationID` points to `choice.selected`; `CorrelationID` matches original trigger.

## DSL Expression

### `scn:present_choice`

```lua
scn:present_choice{
  actor = "GM",
  choice_type = "spawn_variant",
  label = "Choose reinforcement type",
  options = {
    { id = "rotted",    label = "Rotted zombie" },
    { id = "perfected", label = "Perfected undead" },
    { id = "legion",    label = "Legion swarm" },
  },
  blocking = true,
}
```

### `scn:select_choice`

```lua
scn:select_choice{ actor = "GM", selected = "rotted" }
```

Resolves against the most recent unresolved `present_choice` from the same actor, or an explicit `choice_id` can be provided.

### Stress-trade pattern

```lua
scn:present_choice{
  actor = "Frodo",
  choice_type = "stress_trade",
  label = "Mark Stress for extra detail?",
  options = {
    { id = "accept", label = "Accept one detail" },
    { id = "trade",  label = "Mark 1 Stress for all details" },
  },
}
scn:select_choice{ actor = "Frodo", selected = "trade" }
scn:assert_stress{ character = "Frodo", delta = 1 }
```

## Daggerheart Composition Examples

**Spawn-variant** (ossuary reinforcements):
```
choice.present → options: [rotted, perfected, legion]
choice.select  → selected: "rotted"
  └─ sys.daggerheart.adversary.spawn (rotted zombie template)
```

**Mechanical-benefit** (blasphemous might):
```
choice.present → options: [advantage, bonus_damage, relentless]
choice.select  → selected: "advantage"
  └─ sys.daggerheart.condition.change (attack advantage)
```

**Stress-trade** (rumor detail):
```
choice.present → options: [accept_one_rumor, stress_for_extra]
choice.select  → selected: "stress_for_extra"
  └─ sys.daggerheart.character_state.patch (stress +1)
  └─ sys.daggerheart.story_note.set (extra rumor detail)
```

## Scenario Gap Resolution

### Narrative branching gaps

| # | Scenario | Choice expression |
|---|----------|-------------------|
| 1 | `bree_outpost_rumors` | Rumor options; failure branch offers stress-trade for extra rumor. |
| 2 | `prancing_pony_talk` | Detail options; failure offers stress-trade for additional detail. |
| 3 | `moria_ossuary_they_keep_coming` | Spawn-variant options (rotted/perfected/legion); triggers `adversary.spawn`. |
| 4 | `prancing_pony_someone_comes_to_town` | NPC hook options; sets story notes with hook payload and agenda. |
| 5 | `bree_outpost_rival_party` | Rivalry-hook options; establishes persistent rivalry state. |
| 6 | `bree_outpost_broken_compass` | Social-pressure options; sets persistent pressure state. |
| 7 | `osgiliath_ruins_dead_ends` | Route options (detour/challenge/wait); updates route-state. |
| 8 | `isengard_ritual_blasphemous_might` | Mechanical-benefit options (advantage/bonus-damage/Relentless). |

### Stress/consequence gaps

| # | Scenario | Choice expression |
|---|----------|-------------------|
| 9 | `caradhras_pass_icy_winds` | Choice effect envelope applies stress; loop-reset in metadata. |
| 10 | `helms_deep_siege_collateral_damage` | Choice effects conditionally assign stress on success/failure. |
| 11 | `mirkwood_blight_grasping_vines` | Chains: Restrained + Vulnerable + follow-up damage + Hope loss. |
| 12 | `dark_tower_usurpation_ritual_nexus` | Stress-roll effect (1d4) via choice effect envelope. |
| 13 | `misty_ascent_fall` | State-dependent damage in choice metadata + effect payload. |
