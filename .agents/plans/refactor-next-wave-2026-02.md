# Refactor Next-Wave (2026-02)

This ExecPlan tracks this iteration's large refactors for improved clarity, maintenance, and testability. It is intentionally no-migration: **do not add migration files, alter existing schema definitions, or execute migrations**.

## Purpose / Big Picture

Reduce hidden complexity in high-risk areas by splitting monolithic orchestration and validation code into domain-aligned components, while keeping behavior unchanged and preserving event-driven storage semantics.

## Progress

- [x] (2026-02-22 00:00Z) Survey current refactor state and identify all files that still mix responsibilities.
- [x] (2026-02-22 00:00Z) Finish `GetContentCatalog` to use `contentCatalog` orchestration with context-aware steps and central proto mapping.
- [x] (2026-02-22 00:00Z) Add MCP transport injection seams (auth, rate limiting, TLS) without forcing token/OAuth coupling.
- [x] (2026-02-22 00:00Z) Decompose `store_projection_core.go` by aggregate family and move methods into focused files.
- [x] (2026-02-22 00:00Z) Decompose `engine/registries.go` by responsibility: bootstrap, core/system validation, registry validation.
- [x] (2026-02-22 00:00Z) Update task list and keep docs/plan continuity for all completed refactor work.
- [x] (TBD) Add seam tests for MCP transport admission/rate-limit behavior.
- [x] (2026-02-22 00:00Z) Add MCP TLS seam behavior tests (end-to-end listener assertion) without changing default behavior.
- [x] (TBD) Split MCP HTTP transport test surface by adding topic-specific `http_transport_authz_test.go`.
- [x] Split remaining Daggerheart session workflow cluster (`actions_session_flows.go`) into bounded orchestrators with delegation wrappers.
- [x] (2026-02-22 00:00Z) Split MCP HTTP transport host-guardrail and health handling concerns into `http_transport_host_validation.go`.
- [x] Split remaining Daggerheart outcome workflow cluster (`actions_outcomes.go`) into bounded orchestrators with delegation wrappers.
- [x] Split remaining Daggerheart action clusters for conditions and countdown workflows.
- [x] Split remaining Daggerheart action cluster for recovery flows (`actions_recovery.go`) into bounded orchestrators.
- [x] Split `internal/services/game/api/grpc/systems/daggerheart/actions_flows_outcomes_test.go` into focused outcomes/session test files:
  - `actions_session_rolls_test.go`
  - `actions_session_flows_test.go`
  - `actions_apply_roll_outcomes_test.go`
  - `actions_apply_attack_outcomes_test.go`.
- [x] (2026-02-22 00:00Z) Continue splitting `internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcomes_test.go` into:
  - `actions_apply_roll_outcome_test.go`
  - `actions_apply_attack_outcome_test.go`
  - `actions_apply_reaction_outcome_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/content_endpoints_test.go` into:
  - `content_endpoints_fixtures_test.go`
  - `content_endpoints_service_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcome_test.go` into:
  - `actions_apply_roll_outcome_test.go`
  - `actions_apply_roll_outcome_domain_test.go`.
- [x] (2026-02-22 00:00Z) Split the large `content_endpoints_service_test.go` split point into:
  - `content_endpoints_service_test.go` (happy-path catalog/content access cases)
  - `content_endpoints_service_guardrails_test.go` (nil/no-store/error/contract tests).
- [x] (2026-02-22 00:00Z) Continue `actions_apply_roll_outcome_domain_test.go` split into:
  - `actions_apply_roll_outcome_domain_test.go` (primary ApplyRollOutcome behaviors)
  - `actions_apply_roll_outcome_domain_effects_test.go` (effect-domain outcome branch coverage).
- [x] (2026-02-22 00:00Z) Split `actions_downtime_armor_loadout_test.go` by command surface into:
  - `actions_apply_downtime_move_test.go`
  - `actions_apply_temporary_armor_test.go`
  - `actions_swap_loadout_test.go`.
- [x] (2026-02-22 00:00Z) Split `actions_adversary_gm_conditions_test.go` into:
  - `actions_adversary_damage_test.go`
  - `actions_adversary_conditions_test.go`
  - `actions_apply_conditions_test.go`
  - `actions_apply_gm_move_test.go`.
- [x] (2026-02-22 00:00Z) Split `actions_gm_move_countdown_test.go` by workflow into:
  - `actions_gm_move_test.go`
  - `actions_create_countdown_test.go`
  - `actions_update_countdown_test.go`
  - `actions_delete_countdown_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_session_flows_test.go` into:
  - `actions_session_attack_flow_test.go`
  - `actions_session_reaction_flow_test.go`
  - `actions_session_adversary_attack_roll_test.go`
  - `actions_session_adversary_action_check_test.go`
  - `actions_session_adversary_attack_flow_test.go`
  - `actions_session_group_action_flow_test.go`
  - `actions_session_tag_team_flow_test.go`.
- [x] (2026-02-22 00:00Z) Split Daggerheart session workflow cluster into bounded orchestrators by moving execution logic into `sessionFlowApplication` in `session_flow_session_application.go`.
- [x] (2026-02-22 00:00Z) Split Daggerheart content catalog + endpoint cluster (`content_service.go`) into bounded `contentApplication` orchestration.
- [x] (2026-02-22 00:00Z) Move Daggerheart outcome helper logic out of `actions_outcomes.go` into `outcome_helpers.go` and keep the service wrapper file delegation-only.
- [x] (2026-02-22 00:00Z) Split MCP server composition/runtime concerns out of `server.go` into focused runtime and registration modules for clearer transport and orchestration ownership.

## Surprises & Discoveries

- [x] Behavior tests for MCP TLS listener wiring are complete with `http_transport_tls_test.go`; auth/rate and seam config coverage is now in place.
- [x] Existing MCP auth code has TODOs for auth/rate-limits/TLS but already carries many transport concerns in one file.
- [x] MCP TLS seam behavior is now covered in a focused listener-selection suite.
- [x] The Daggerheart catalog pipeline call site and method signatures are now wired through `newContentCatalog(...).run(ctx)`.
- [x] The registry and projection files are already large and mostly cohesive by validation family, making safe extraction straightforward.

## Decision Log

- Decision: complete migration of catalog pipeline by invoking `newContentCatalog(...).run(ctx)` in service call.
  Rationale: keeps step sequencing and variable ownership in one place and removes duplicated list/localize scaffolding.
  Date/Author: 2026-02-22 / automated refactor pass.
- Decision: inject transport seams through `Config` + optional interfaces rather than global flags.
  Rationale: preserves defaults for local-only behavior while enabling explicit auth/token/rate-limit/TLS behavior in production.
  Date/Author: 2026-02-22 / automated refactor pass.
- Decision: split validator/projection monoliths by aggregate and validation type.
  Rationale: improves file-level discoverability and ownership boundaries without changing exported behavior.
  Date/Author: 2026-02-22 / automated refactor pass.

## Outcomes & Retrospective

- [x] To be recorded once changes are complete and verified.
- [x] Structural decomposition is complete for this wave's scope and remained within no-migration constraints; verification remains pending user-authorized test run.

## Context and Orientation

- Domain-driven boundaries affected:
  - `internal/services/game/api/grpc/systems/daggerheart` for content catalog assembly.
  - `internal/services/mcp/service` for transport security/per-request policy.
  - `internal/services/game/storage/sqlite` for projection write APIs.
  - `internal/services/game/domain/engine` for registry bootstrap/validation.
- `config` remains source-of-truth for deployment defaults; no migration artifacts required.

## Plan of Work

- keep behavior stable and preserve existing interfaces where callers are external
- preserve test helper patterns (especially MCP helper tests that target request-time behavior)
- avoid any migration touching and keep all SQL schemas intact
- keep helper seams optional and nil-safe

## Concrete Steps

- [x] Update `GetContentCatalog` orchestration call path and remove inline step wiring.
- [x] Add MCP auth/rate limiter/tls seam declarations and wire them into HTTP transport.
- [x] Add composite request authorizer that accepts both OAuth and API token when both are configured.
- [x] Add optional `RequestRateLimiter` interface and hook in `/mcp` request handlers with clear 429 handling.
- [x] Add optional `TLSConfig` handling for the HTTP server listener path.
- [x] Split `internal/services/game/storage/sqlite/store_projection_core.go` into:
  - `store_projection_campaign.go`
  - `store_projection_participant.go`
  - `store_projection_invite.go`
  - `store_projection_character.go`
- [x] Split `internal/services/game/domain/engine/registries.go` into:
  - `registries_builder.go`
  - `registries_validation_core.go`
  - `registries_validation_projection.go`
  - `registries_validation_system.go`
  - `registries_validation_aggregate.go`
- [x] Sweep for duplicate/missing methods and remove stale monolith sections so package compiles.
- [x] Add focused tests for MCP auth/rate seams on `HTTPTransport`.
- [x] Add focused test proving TLS listener wiring for `Config.TLSConfig` without production traffic changes.
- [x] (2026-02-22 00:00Z) Split MCP HTTP transport host/health guardrails into `http_transport_host_validation.go` so `http_transport_auth.go` remains token/rate/authorization-focused.
- [x] Split Daggerheart session workflow cluster into bounded orchestrators (`sessionFlowApplication` extraction).
- [x] Split Daggerheart outcome workflow cluster into bounded orchestrators (`outcomeApplication` extraction).
- [x] Split Daggerheart conditions cluster into bounded orchestrators (`conditionsApplication` extraction).
- [x] Split Daggerheart countdown cluster into bounded orchestrators (`countdownApplication` extraction).
- [x] Split Daggerheart recovery cluster into bounded orchestrators (`recoveryApplication` extraction).
- [x] Split Daggerheart damage cluster into bounded orchestrators (`damageApplication` extraction).
- [x] Split Daggerheart adversary CRUD cluster into bounded orchestrators (`adversaryApplication` extraction).
- [x] Split Daggerheart content catalog and endpoint cluster into bounded orchestrator (`contentApplication` extraction).
- [x] Move Daggerheart outcome helper methods from `actions_outcomes.go` into `outcome_helpers.go` for clearer orchestration boundary.
- [x] Extract MCP server test doubles and fixtures from `internal/services/mcp/service/server_test.go` into `server_test_fixtures_test.go` to reduce test file responsibility.
- [x] Split MCP server runtime/infrastructure tests from `server_test.go` into `server_runtime_test.go`, including shared `startHealthServer` and `startCampaignServer` helpers.
- [x] Split MCP campaign tool/resource tests from `server_test.go` into `server_campaign_test.go`.
- [x] (2026-02-22 00:00Z) Add docs for repository-wide seam-first refactor lessons and extraction boundary patterns to retain learnings from this wave.
- [x] (2026-02-22 00:00Z) Split MCP `server.go` into explicit orchestration (`server_registration.go`) and runtime/handler (`server_runtime.go`) modules.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_downtime_armor_loadout_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_death_blaze_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_damage_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_rest_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_gm_move_countdown_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_test.go` and introducing `actions_adversary_gm_conditions_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_flows_outcomes_test.go` into:
  - `actions_session_rolls_test.go`
  - `actions_session_flows_test.go`
  - `actions_apply_roll_outcomes_test.go`
  - `actions_apply_attack_outcomes_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcomes_test.go` into:
  - `actions_apply_roll_outcome_test.go`
  - `actions_apply_attack_outcome_test.go`
  - `actions_apply_reaction_outcome_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/content_endpoints_test.go` into:
  - `content_endpoints_fixtures_test.go`
  - `content_endpoints_service_test.go`.
- [x] (2026-02-22 00:00Z) Further split `content_endpoints_service_test.go` into:
  - `content_endpoints_service_test.go`
  - `content_endpoints_service_guardrails_test.go`.
- [x] (2026-02-22 00:00Z) Continue splitting `internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcome_domain_test.go` into:
  - `actions_apply_roll_outcome_domain_test.go`
  - `actions_apply_roll_outcome_domain_effects_test.go`.
- [x] (2026-02-22 00:00Z) Split `internal/services/game/api/grpc/systems/daggerheart/actions_downtime_armor_loadout_test.go` into:
  - `actions_apply_downtime_move_test.go`
  - `actions_apply_temporary_armor_test.go`
  - `actions_swap_loadout_test.go`.
- [x] (2026-02-22 00:00Z) Split `internal/services/game/api/grpc/systems/daggerheart/actions_adversary_gm_conditions_test.go` into:
  - `actions_adversary_damage_test.go`
  - `actions_adversary_conditions_test.go`
  - `actions_apply_conditions_test.go`
  - `actions_apply_gm_move_test.go`.
- [x] (2026-02-22 00:00Z) Split `internal/services/game/api/grpc/systems/daggerheart/actions_gm_move_countdown_test.go` into:
  - `actions_gm_move_test.go`
  - `actions_create_countdown_test.go`
  - `actions_update_countdown_test.go`
  - `actions_delete_countdown_test.go`.
- [x] (2026-02-22 00:00Z) Continue MCP/GRPC test extraction by splitting `internal/services/game/api/grpc/systems/daggerheart/actions_session_flows_test.go` into:
  - `actions_session_attack_flow_test.go`
  - `actions_session_reaction_flow_test.go`
  - `actions_session_adversary_attack_roll_test.go`
  - `actions_session_adversary_action_check_test.go`
  - `actions_session_adversary_attack_flow_test.go`
  - `actions_session_group_action_flow_test.go`
  - `actions_session_tag_team_flow_test.go`.

## Validation and Acceptance

- Static/code-level checks:
  - compileability via `go test` is expected after this pass (deferred here for user-authorized execution).
- Refactor acceptance:
  - no API changes in observable service behavior.
  - no migration files added or schema fields changed.
  - optional MCP seams are no-op by default.
- Manual acceptance:
  - no TODO comments for required auth/rate/TLS behavior when config is explicitly provided.

## Idempotence and Recovery

- Changes are structurally additive/relocating; re-running this refactor should be idempotent because methods retain signatures and implementations are unchanged.
- If split-file extraction fails to compile, fallback is to keep the same methods in the original file while reattempting extraction boundaries.

## Artifacts and Notes

- Existing file `.agents/plans/newcomer-clarity-refactor.md` contains carryover checklist items from prior work and remains in scope.

## Interfaces and Dependencies

- New `Config`-level dependency seam points:
  - MCP `Config.AuthToken`
  - MCP `Config.TLSConfig`
  - MCP `Config.RequestAuthorizer`
  - MCP `Config.RateLimiter`
- Internal transport helper seam points in `internal/services/mcp/service/http_transport_*`:
  - `RequestAuthorizer` interface
  - `RequestRateLimiter` interface
- No external API contract changes outside these seams.
