---
title: "Daggerheart Event Timeline Contract"
parent: "Project"
nav_order: 23
---

# Daggerheart Event Timeline Contract

This document maps high-traffic Daggerheart mechanics onto the canonical write path:

`request -> command -> decider -> event append -> projection apply policy`

Use this as the onboarding contract for new mechanics and for review of existing paths.

## Command/Event Timeline Map

| Mechanic | Command Type(s) | Emitted Event Type(s) | Projection Targets | Apply policy notes | Required invariants |
| --- | --- | --- | --- | --- | --- |
| Action roll resolution | `action.roll.resolve` | `action.roll_resolved` | Event journal (no direct Daggerheart projection mutation) | Request path records event; projection apply is skipped for this envelope | Campaign/session valid; roll payload valid; command must emit event |
| Roll outcome finalization | `action.outcome.apply` | `action.outcome_applied` | Event journal (plus follow-on Daggerheart/system commands) | Request path records outcome event; projection apply is skipped for this envelope | Roll event exists and matches session; no duplicate/bypass apply |
| Outcome-driven GM Fear update | `sys.daggerheart.gm_fear.set` | `sys.daggerheart.gm_fear_changed` | Daggerheart snapshot (`gm_fear`) | Inline apply depends on runtime mode; outbox mode must not inline apply in request path | Fear bounds and spend/gain checks enforced |
| Outcome-driven character state patch | `sys.daggerheart.character_state.patch` | `sys.daggerheart.character_state_patched` | Daggerheart character state | Inline apply mode-controlled | Patch payload must include meaningful deltas |
| Outcome-driven condition change | `sys.daggerheart.condition.change` | `sys.daggerheart.condition_changed` | Daggerheart character conditions | Inline apply mode-controlled | Normalized set diff; no empty/invalid conditions |
| Session gate for GM consequence | `session.gate_open`, `session.spotlight_set` | `session.gate_opened`, `session.spotlight_set` | Session gate + spotlight projections | Inline apply mode-controlled | One open gate at a time; request/session correlation |
| Character damage apply | `sys.daggerheart.damage.apply` | `sys.daggerheart.damage_applied` | Daggerheart character HP/armor | Inline apply mode-controlled | Campaign system is Daggerheart; damage payload valid; emits event |
| Adversary damage apply | `sys.daggerheart.adversary_damage.apply` | `sys.daggerheart.adversary_damage_applied` | Daggerheart adversary HP/armor | Inline apply mode-controlled | Adversary exists in session; payload valid; emits event |
| Rest | `sys.daggerheart.rest.take` | `sys.daggerheart.rest_taken` | Daggerheart snapshot and targeted character state | Inline apply mode-controlled | Rest type valid; campaign/session mutate gates pass |
| Downtime move | `sys.daggerheart.downtime_move.apply` | `sys.daggerheart.downtime_move_applied` | Daggerheart character state | Inline apply mode-controlled | Move is valid; resulting resource bounds valid |
| Temporary armor apply | `sys.daggerheart.character_temporary_armor.apply` | `sys.daggerheart.character_temporary_armor_applied` | Daggerheart temporary armor buckets and armor totals | Inline apply mode-controlled | Source/duration/amount validation; emits event |
| Loadout swap and associated resource mutation | `sys.daggerheart.loadout.swap`, `sys.daggerheart.stress.spend` | `sys.daggerheart.loadout_swapped`, `sys.daggerheart.character_state_patched` | Daggerheart character loadout-facing stress/state | Inline apply mode-controlled for Daggerheart events | Recall cost bounds; stress spend consistency |
| Character conditions apply endpoint | `sys.daggerheart.condition.change`, `sys.daggerheart.character_state.patch` (life state updates) | `sys.daggerheart.condition_changed`, `sys.daggerheart.character_state_patched` | Character conditions/life state | Inline apply mode-controlled | No-op updates rejected; roll correlation checked when provided |
| Adversary condition changes | `sys.daggerheart.adversary_condition.change` | `sys.daggerheart.adversary_condition_changed` | Adversary conditions | Inline apply mode-controlled | No-op updates rejected; normalized set required |
| Countdown create/update/delete | `sys.daggerheart.countdown.create`, `sys.daggerheart.countdown.update`, `sys.daggerheart.countdown.delete` | `sys.daggerheart.countdown_created`, `sys.daggerheart.countdown_updated`, `sys.daggerheart.countdown_deleted` | Daggerheart countdown projections | Inline apply mode-controlled | Countdown bounds/rules validated before command |
| Adversary create/update/delete | `sys.daggerheart.adversary.create`, `sys.daggerheart.adversary.update`, `sys.daggerheart.adversary.delete` | `sys.daggerheart.adversary_created`, `sys.daggerheart.adversary_updated`, `sys.daggerheart.adversary_deleted` | Daggerheart adversary projections | Inline apply mode-controlled | Session-scoped adversary integrity and payload validation |

## Non-Negotiable Handler Rules

1. Mutating request handlers must use shared orchestration (`executeAndApplyDomainCommand`).
2. Request handlers must not call direct event append APIs.
3. Request handlers must not call direct projection/storage mutation APIs for domain outcomes.
4. Every mutating command path must reject empty decision events unless explicitly audit-only.
5. Inline projection apply behavior must be controlled only by runtime mode policy.

## Required Guard Tests

Use these tests as baseline architecture guardrails:

- `internal/services/game/api/grpc/systems/daggerheart/write_path_arch_test.go`
- `internal/services/game/api/grpc/systems/daggerheart/domain_write_helper_test.go`
- `internal/services/game/api/grpc/game/domain_write_helper_test.go`

When adding a new mutating mechanic, update/add tests so bypass patterns fail fast.

## How To Add A New Daggerheart Mechanic

1. Add command and event registrations in the Daggerheart decider/registry.
2. Add a timeline row in this document before implementation.
3. Implement request handler using shared write orchestration only.
4. Implement/update adapter projection handling for emitted event types.
5. Add Red/Green tests:
   - command/event behavior
   - projection/apply behavior
   - architecture bypass guard where relevant
6. Validate runtime mode behavior (`inline_apply_only`, `outbox_apply_only`, `shadow_only`) is explicit for the new path.
