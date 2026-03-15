# Session 22: API Transport — Core Entity Transports (Part 2)

## Status: `complete`

## Package Summaries

### `api/grpc/game/interactiontransport/` (interaction_application.go at 1,354 lines)
Interaction transport handling gate workflows, spotlight management, and AI turn orchestration.

### `api/grpc/game/charactertransport/` (20 files)
Character transport: CRUD, avatar, aliases, ownership transfer.

### `api/grpc/game/participanttransport/` (17 files)
Participant transport: join, leave, seat reassignment, role management.

### `api/grpc/game/invitetransport/` (invite_claim_application.go at 669 lines)
Invite transport with complex claim workflow.

### Other packages:
- `scenetransport/` — Scene lifecycle and character management
- `forktransport/` — Campaign fork operations
- `snapshottransport/` — Snapshot management
- `authorizationtransport/` — Authorization query endpoints
- `characterworkflow/` — Character creation workflow
- `authz/` (13 files) — Transport-level authorization enforcement

## Findings

### Finding 1: interaction_application.go at 1,354 Lines — Strongest Decomposition Candidate
- **Severity**: critical
- **Location**: `api/grpc/game/interactiontransport/interaction_application.go`
- **Issue**: The single largest production file in the entire service at 1,354 lines. Handles gate workflows, spotlight management, OOC operations, and AI turn orchestration in one file. This is a maintainability hazard — any change requires understanding all interaction types. The file likely has high merge conflict frequency.
- **Recommendation**: Split into:
  - `interaction_gate.go` — Gate open/respond/resolve/abandon
  - `interaction_spotlight.go` — Spotlight set/clear
  - `interaction_ooc.go` — OOC pause/post/ready/resume
  - `interaction_ai_turn.go` — AI turn queue/start/fail/clear
  Each file should be 200-400 lines. This is the highest-priority decomposition in the service.

### Finding 2: invite_claim_application.go at 669 Lines — Complex Workflow
- **Severity**: medium
- **Location**: `api/grpc/game/invitetransport/invite_claim_application.go`
- **Issue**: Invite claim is a complex cross-entity workflow: validate invite, resolve user, check existing participation, create participant, bind character. At 669 lines, this is a workflow handler, not a simple CRUD operation.
- **Recommendation**: The complexity is inherent in the workflow. Consider extracting the multi-step orchestration into a domain-level workflow service rather than embedding it in the transport layer.

### Finding 3: authz/ (13 Files) Boundary with domain/authz/
- **Severity**: info
- **Location**: `api/grpc/game/authz/`
- **Issue**: Transport authz enforces authorization at the gRPC boundary: resolving caller identity to participant, checking capabilities via `domain/authz`, and returning appropriate gRPC errors. Domain authz provides the policy matrix. The transport layer applies it.
- **Recommendation**: Clean boundary. Transport authz is the enforcement point; domain authz is the policy definition. The 13 files handle different entity-specific authorization flows.

### Finding 4: Authorization Enforcement Consistency
- **Severity**: medium
- **Location**: Across all transport packages
- **Issue**: Every transport handler must enforce authorization. The pattern should be: resolve identity → check capability → proceed or reject. Inconsistencies (e.g., missing authz checks on new endpoints) would create security gaps. The architecture test (Session 20) likely validates this.
- **Recommendation**: The write_path_architecture_test.go should verify authorization is enforced on all write endpoints. For read endpoints, verify that authorization checks exist in the transport layer.

### Finding 5: characterworkflow/ vs charactertransport/ Boundary
- **Severity**: low
- **Location**: `api/grpc/game/characterworkflow/`, `api/grpc/game/charactertransport/`
- **Issue**: `characterworkflow/` handles multi-step character creation (which involves both core character creation and system-specific profile setup). `charactertransport/` handles standard character CRUD. The workflow package exists because creation requires orchestrating multiple domain commands.
- **Recommendation**: The boundary is clear but the naming could be improved. Consider `charactercreation/` instead of `characterworkflow/` for clarity.

## Summary Statistics
- Files reviewed: ~100+ (interaction + character + participant + invite + scene + fork + snapshot + authz + workflow)
- Findings: 5 (1 critical, 0 high, 2 medium, 1 low, 1 info)
