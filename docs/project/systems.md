# Systems: From Zero to Playable

This guide summarizes the system-level work required to bring a new ruleset into Fracturing.Space, from initial definition to a playable mechanical core. It is system-agnostic and focused on what/why, not implementation details.

## 1) Define the System Surface
- **Ruleset identity**: name, version, and canonical dice model.
- **Outcome taxonomy**: the result categories players/GM must reason about.
- **Resource model**: player and GM currencies, caps, and default values.
- **State scope**: what belongs in profile (static) vs state (dynamic) vs snapshot (campaign-level).

## 2) Deterministic Resolution Core
- **Deterministic rolls**: seedable randomness with explicit input/output.
- **Outcome evaluation**: pure functions that map inputs to results.
- **Explainability**: a rules explanation surface for debugging and audit.

## 3) Profiles, State, and Projections
- **Profile schema**: traits, thresholds, and static modifiers.
- **State schema**: mutable resources and combat state.
- **Snapshots**: campaign-level state and GM resources.
- **Projections**: derive all state from append-only events.

## 4) Core Combat Mechanics
- **Attack resolution**: hit/avoid flow and difficulty targets.
- **Damage system**: damage rolls, thresholds, and severity mapping.
- **Mitigation**: resistance, immunity, and armor rules.
- **Critical rules**: critical hit/defense effects.

## 5) Recovery and Downtime
- **Rest cadence**: short/long rest semantics and interruption rules.
- **Downtime moves**: recovery moves and project progression.
- **Refresh model**: per-rest and per-long-rest feature refresh.

## 6) Ability Modules and Loadouts
- **Ability types**: spells/abilities/techniques and their common fields.
- **Loadout rules**: active vs vaulted capacity.
- **Swap constraints**: recall costs or swap rules outside rest.

## 7) Validation and Guardrails
- **Caps and ranges**: enforce system constraints at domain and projection layers.
- **Event safety**: reject invalid payloads in projections.
- **Versioning**: system version compatibility for future rulesets.

## 8) Exposed Surfaces
- **Internal domain API** first (for mechanics).
- **Transport APIs** after mechanics stabilize.
- **Interface layers** (e.g., MCP) last.

## Success Criteria
- All deterministic mechanics are reproducible with seeded inputs.
- Profiles and state can be rebuilt solely from event history.
- Core combat, recovery, and ability loadout rules are mechanically complete.
