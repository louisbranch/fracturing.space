// Package game exposes the stable gRPC surface over core domain aggregates.
//
// Each service maps to a domain aggregate and defines the intent boundary for one
// area of campaign state:
//   - CampaignService -> campaign lifecycle, status transitions, and fork metadata
//   - ParticipantService -> participant roster, roles, and controller access
//   - InviteService -> invite issuance, claim, and revocation workflows
//   - CharacterService -> character ownership and profile state
//   - SessionService -> active session lifecycle, gates, and spotlight
//   - SnapshotService -> replay checkpoints and materialized read state
//   - ForkService -> campaign branching for scenario exploration
//   - EventService -> raw event stream visibility for audit/debug
//   - StatisticsService -> aggregate-level telemetry and counters
//
// Store wiring contracts are split by startup concern:
//   - store declarations + runtime config (`stores.go`)
//   - projection-bundle constructor (`stores_construction.go`)
//   - applier construction/cache contract (`stores_applier.go`)
//   - startup validation requirements (`stores_validation.go`)
//
// Invite transport orchestration is partitioned by use-case:
//   - create flow (`invite_create_application.go`)
//   - claim flow (`invite_claim_application.go`)
//   - revoke flow (`invite_revoke_application.go`)
//
// Invite service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`invite_service.go`)
//   - mutation handlers (`invite_service_mutation.go`)
//   - read/list handlers (`invite_service_read.go`)
//   - pending-invite read handlers (`invite_service_pending.go`)
//
// Character transport orchestration is intentionally split by use-case to keep
// contributor onboarding shallow:
//   - create flow (`character_create_application.go`)
//   - update flow (`character_update_application.go`)
//   - delete flow (`character_delete_application.go`)
//   - control flow (`character_control_application.go`)
//   - workflow flow (`character_workflow.go`)
//   - workflow RPC handlers (`character_workflow_service.go`)
//   - profile patch flow (`character_profile_patch.go`)
//
// Character service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`character_service.go`)
//   - mutation handlers (`character_service_mutation.go`)
//   - read/sheet handlers (`character_service_read.go`)
//   - Daggerheart profile/state proto mappers (`character_service_helpers.go`)
//
// Campaign transport orchestration follows the same split-by-use-case pattern:
//   - create flow (`campaign_create_application.go`)
//   - mutation flow (`campaign_mutation_application.go`)
//   - status transitions (`campaign_status_application.go`)
//   - AI binding flow (`campaign_ai_binding_application.go`)
//
// Campaign service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`campaign_service.go`)
//   - create handler (`campaign_service_create.go`)
//   - list handler + pagination helpers (`campaign_service_list.go`)
//   - read handler (`campaign_service_read.go`)
//   - mutation handlers (`campaign_service_mutation.go`)
//   - shared campaign service helpers (`campaign_service_helpers.go`)
//   - readiness handler (`campaign_readiness_service.go`)
//   - readiness state builders (`campaign_readiness_state.go`)
//   - readiness locale/blocker mapping helpers (`campaign_readiness_localization.go`)
//
// Campaign AI internal service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`campaign_ai_service.go`)
//   - session-grant issuance handler (`campaign_ai_service_issue_grant.go`)
//   - AI binding usage handler (`campaign_ai_service_binding_usage.go`)
//   - auth-state read handler (`campaign_ai_service_auth_state.go`)
//   - auth-epoch rotation helper (`campaign_ai_auth_rotation.go`)
//
// Fork transport orchestration is split by lifecycle intent:
//   - application coordinator (`fork_application.go`)
//   - fork command flow + event replay (`fork_application_fork.go`)
//   - fork-point resolution (`fork_application_fork_point.go`)
//
// Fork service RPC handlers and shared helpers are split by intent:
//   - service constructors/coordinator (`fork_service.go`)
//   - fork mutation handler (`fork_service_fork.go`)
//   - lineage/list read handlers (`fork_service_read.go`)
//   - shared fork helper contracts (`fork_service_helpers.go`)
//
// Participant transport orchestration mirrors these boundaries:
//   - create flow (`participant_create_application.go`)
//   - update flow (`participant_update_application.go`)
//   - delete flow (`participant_delete_application.go`)
//   - mutation helpers (`participant_mutation_helpers.go`)
//   - policy helpers (`participant_policy_helpers.go`)
//
// Participant service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`participant_service.go`)
//   - mutation handlers (`participant_service_mutation.go`)
//   - read/list handlers (`participant_service_read.go`)
//
// Scene transport orchestration is partitioned by intent:
//   - lifecycle flow (`scene_lifecycle_application.go`)
//   - character membership flow (`scene_character_application.go`)
//   - gate flow (`scene_gate_application.go`)
//   - spotlight flow (`scene_spotlight_application.go`)
//
// Scene service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`scene_service.go`)
//   - lifecycle handlers (`scene_service_lifecycle.go`)
//   - character membership handlers (`scene_service_character.go`)
//   - gate handlers (`scene_service_gate.go`)
//   - spotlight handlers (`scene_service_spotlight.go`)
//   - read/list handlers + proto mapping (`scene_service_read.go`)
//
// Session transport orchestration follows the same separation:
//   - lifecycle flow (`session_lifecycle_application.go`)
//   - gate flow (`session_gate_application.go`)
//   - spotlight flow (`session_spotlight_application.go`)
//
// Session service RPC handlers are split by transport intent:
//   - service constructors/coordinator (`session_service.go`)
//   - lifecycle + read handlers (`session_service_lifecycle.go`)
//   - gate handlers (`session_service_gate.go`)
//   - spotlight handlers (`session_service_spotlight.go`)
//
// Snapshot transport orchestration isolates mutation paths:
//   - character state patch flow (`snapshot_state_patch_application.go`)
//   - character state patch validation/payload helpers (`snapshot_state_patch_helpers.go`)
//   - character state/condition command emission helpers (`snapshot_state_patch_commands.go`)
//   - snapshot update flow (`snapshot_update_application.go`)
//   - stress->condition helper flow (`snapshot_condition_helpers.go`)
//
// Event stream transport orchestration is split by read/stream concerns:
//   - append write flow (`event_append_service.go`)
//   - list pagination flow (`event_list_service.go`)
//   - subscribe realtime flow (`event_subscribe_service.go`)
//   - event/update mapping helpers (`event_mapping_helpers.go`)
//
// Authorization transport orchestration is partitioned by policy intent:
//   - single-check policy flow (`authorization_can_service.go`)
//   - batch-check fanout flow (`authorization_batch_service.go`)
//   - action/resource mapping helpers (`authorization_mapping_helpers.go`)
//   - target governance helpers (`authorization_target_evaluator.go`)
//   - shared policy guard flows (`authorization_policy.go`)
//   - actor resolution helpers (`authorization_actor_resolution.go`)
//   - override/attribute helpers (`authorization_helpers.go`)
//   - decision telemetry emission (`authorization_telemetry.go`)
//
// Event timeline transport separates handler/mapping/projection concerns:
//   - list handler flow (`timeline_service.go`)
//   - timeline entry mapping (`timeline_entry_mapping.go`)
//   - projection resolver/cache (`timeline_projection_resolver.go`)
//   - projection display builders (`timeline_projection_display.go`)
package game
