# Pass 12: Authorization and Policy Matrix

## Summary

The authorization system is well-architected with a clean separation between
domain policy (`domain/authz/`) and transport enforcement
(`api/grpc/game/authz/`, `api/grpc/interceptors/`). The policy matrix is
declarative, exhaustiveness tests are thorough, and the session lock
interceptor has a startup validation check that catches drift between transport
and domain. Several issues remain: the telemetry interceptor's
`classifyMethodKind` is incomplete and will misclassify read methods as writes;
inline role checks in handler code bypass the domain policy matrix; and the
exhaustiveness tests do not verify that every (action x resource) combination
has a matrix entry.

---

## Findings

### 1. `classifyMethodKind` missing many read methods -- correctness risk

**File:** `internal/services/game/api/grpc/interceptors/telemetry.go:178-209`

The `classifyMethodKind` switch classifies methods as "read" or "write" for
audit telemetry. Its `default` case returns `"write"`, so any method not
explicitly listed as "read" is classified as a write. Multiple read-only
methods are missing from the "read" list:

- `CharacterService_ListCharacterProfiles_FullMethodName`
- `SessionService_ListActiveSessionsForUser_FullMethodName`
- `SceneService_GetScene_FullMethodName`
- `SceneService_ListScenes_FullMethodName`
- `InteractionService_GetInteractionState_FullMethodName`
- `CampaignAIService_GetCampaignAIBindingUsage_FullMethodName`
- `CampaignAIService_GetCampaignAIAuthState_FullMethodName`
- `IntegrationService_LeaseIntegrationOutboxEvents_FullMethodName` (debatable,
  but it is a read-like lease)

This causes audit events to report read-only RPCs as "write" operations,
polluting audit logs and making security auditing unreliable.

**Proposal:** Invert the approach: list write methods explicitly and default to
"read". Alternatively, add a test that compiles all `_FullMethodName` constants
from generated protos and asserts each one appears in the classification switch.
This would make proto additions fail the test until classification is updated.

---

### 2. Inline role checks bypass the domain policy matrix -- anti-pattern

Several handler files contain direct `CampaignAccess` comparisons instead of
delegating to `domain/authz` functions:

**(a)** `internal/services/game/api/grpc/game/campaigntransport/campaign_ai_binding_application.go:126`
```go
if actor.CampaignAccess != participant.CampaignAccessOwner {
```
The `requireCampaignOwner` helper calls `RequirePolicyActor` with
`CapabilityManageCampaign()` first (which allows both owner and manager), then
further restricts to owner-only with a hard-coded check. This is an
owner-only guard that has no corresponding capability in `domain/authz/`.

**(b)** `internal/services/game/api/grpc/game/interactiontransport/interaction_control.go:120`
```go
if actor.CampaignAccess == participant.CampaignAccessOwner || actor.CampaignAccess == participant.CampaignAccessManager {
```
This is a UI-control-state decision gating which transitions are shown, but it
duplicates the owner/manager role check outside the policy matrix.

**(c)** `internal/services/game/api/grpc/game/forktransport/fork_public_template.go:47`
```go
if record.CampaignAccess != participant.CampaignAccessOwner {
```
This validates that a public fork source has exactly one human owner seat.
Structural validation, not an authorization decision, but uses the same enum.

**Proposal:** For (a), consider adding `CapabilityManageAIBinding()` or similar
to the domain policy matrix with `roles(participant.CampaignAccessOwner)`, then
use `RequirePolicyActor` with that capability instead of a post-check.
For (b) and (c), these are acceptable since they are structural or UI-control
decisions rather than authorization gates -- but they should carry inline
comments documenting why they are not delegated to the policy matrix.

---

### 3. Exhaustiveness test does not verify full (action x resource) coverage -- missing best practice

**File:** `internal/services/game/domain/authz/policy_exhaustiveness_test.go`

The exhaustiveness tests verify:
- Every entry in `policyMatrix` has no duplicate (action,resource) pairs
- Every role in the matrix is recognized
- Every action and resource appears in at least one entry
- Every `Capability*()` accessor maps to an entry

However, they do **not** verify that every possible `(action, resource)`
combination has an entry. Currently 8 of 20 possible combinations are
implicitly denied by their absence (e.g., `mutate:campaign`,
`transfer_ownership:participant`). This is likely intentional, but there is no
test that explicitly asserts which combinations are intentionally absent.

If a new resource is added (e.g., `ResourceScene`), adding it to
`allResources` would pass the "covers all resources" test as long as one
action references it, but would silently leave many action combinations
undefined with no compile-time or test-time signal.

**Proposal:** Add a
`TestPolicyMatrixIntentionalGaps` that explicitly enumerates which
(action,resource) pairs are intentionally absent and fails if a new pair
appears without either a matrix entry or a gap declaration.

---

### 4. `ValidateSessionLockPolicyCoverage` is namespace-level, not per-method -- contributor friction

**File:** `internal/services/game/api/grpc/interceptors/session_lock.go:124-155`

The validation checks that:
1. Every RPC in `blockedMethodCommandTypes` maps to a command that is actually
   classified as "blocked" in the registry.
2. Every blocked core command *namespace* has at least one transport entry.

The namespace-level check is coarse: if a new blocked command
`campaign.rebrand` is added, it would be covered by the existing `campaign`
namespace and would not trigger a validation failure, even though no RPC maps
to it. This means a contributor could add a new blocked command, register it,
and never add the corresponding RPC to `blockedMethodCommandTypes` -- the
startup validation would still pass.

This is partially mitigated by the fact that the mapping also goes the other
direction (RPC -> command type), so a new RPC added without a command
registration would fail. But a new command without a corresponding RPC silently
passes.

**Proposal:** Tighten the validation to check that every blocked core command
type (not just namespace) has at least one RPC method mapped to it, or add a
test-only exhaustive check.

---

### 5. Session lock interceptor lacks streaming support -- correctness risk

**File:** `internal/services/game/api/grpc/interceptors/session_lock.go:63`

`SessionLockInterceptor` is only a `grpc.UnaryServerInterceptor`. The stream
interceptor chain in `bootstrap_transport.go:158-163` does not include a
session lock check. If a mutating streaming RPC is added in the future, it
would bypass session lock enforcement entirely.

Currently all mutating RPCs are unary, so this is not an active bug. But it is
a latent gap that a contributor adding a streaming mutator would not
automatically discover.

**Proposal:** Either add a `SessionLockStreamInterceptor` placeholder that
rejects blocked methods (symmetric with the unary chain), or add a startup
assertion that no registered streaming service has methods in the
`blockedMethodCommandTypes` map.

---

### 6. `constants.go` duplicates doc comment -- contributor friction

**File:** `internal/services/game/api/grpc/game/authz/constants.go:1-3`

The package-level doc comment reads:

```go
// Package authz provides authorization policy enforcement, evaluator, and
// telemetry shared across entity-scoped transport subpackages.
```

But `doc.go` in the same package already has a package doc comment:

```go
// Package authz provides authorization policy enforcement, actor resolution,
// and telemetry for the game gRPC transport layer.
```

Having two conflicting package doc comments causes `godoc` to pick one
arbitrarily and confuses contributors about which is canonical.

**Proposal:** Remove the doc comment from `constants.go` (leave only the one
in `doc.go`).

---

### 7. `CanCharacterMutation` uses `strings.TrimSpace` on typed IDs -- anti-pattern

**File:** `internal/services/game/domain/authz/policy.go:294-296`

```go
actor := ids.ParticipantID(strings.TrimSpace(actorParticipantID.String()))
owner := ids.ParticipantID(strings.TrimSpace(ownerParticipantID.String()))
```

The function accepts `ids.ParticipantID` typed values but immediately converts
them to strings, trims, and reconverts. This defensive trim is redundant if
callers already normalize IDs (which the command registry does). More
importantly, it converts typed IDs to untyped strings for the comparison,
losing the type safety that `ids.ParticipantID` provides.

**Proposal:** Either trust the typed ID and compare directly
(`actorParticipantID == ownerParticipantID`), or do the trimming once at the
transport boundary and pass guaranteed-clean values.

---

### 8. `PolicyTable()` allocates on every call -- missing best practice

**File:** `internal/services/game/domain/authz/policy.go:244-256`

`PolicyTable()` iterates the full `policyMatrix`, expanding each entry into
individual `RolePolicyRow` values. This allocates a new slice on every call.
The function is only used in tests today, but its name and export status
suggest it is meant as a public API.

**Proposal:** Minor: either mark it as test-only (move to a `_test.go` helper
or unexport) or cache the result like `matrixIndex`.

---

### 9. Admin override bypasses all domain policy checks -- correctness risk (by design, but under-tested)

**File:** `internal/services/game/api/grpc/game/authz/actor_resolution.go:20-32`

When `AdminOverrideFromContext` returns true, the actor resolution short-
circuits and returns a synthetic participant record with no `CampaignAccess`
set. This means the admin override bypasses:
- The role-based policy matrix
- Character ownership checks
- Participant governance invariants (last-owner guard, etc.)

The override requires both `ADMIN` platform role and a non-empty override
reason, and the audit telemetry records it as a separate "override" decision.
This is clearly intentional. However, the admin override path is only tested
for the `RequirePolicy` and `RequireCharacterMutationPolicy` flows -- it is
not tested for participant governance operations
(`CanParticipantAccessChange`, `CanParticipantRemoval`).

**Proposal:** Add tests verifying that admin override correctly bypasses
participant governance guards when invoked through the evaluator path.

---

### 10. `Evaluator.Evaluate` does not return error for campaign store failures -- correctness risk

**File:** `internal/services/game/api/grpc/game/authz/evaluator.go:58-61`

```go
campaignRecord, err := e.stores.Campaign.Get(ctx, campaignID)
if err != nil {
    return nil, err
}
```

The campaign store error is returned raw without going through
`grpcerror.Internal` or any conversion. This means if the store returns a
non-gRPC error (e.g., a database timeout), it will bubble up without proper
gRPC status code translation. The `ErrorConversionUnaryInterceptor` would
catch it at the outermost layer, but the evaluator's telemetry would not
record the failure as an authz decision event.

**Proposal:** Wrap the campaign store error similarly to how participant store
errors are wrapped, and emit a decision telemetry event for the failure path.

---

### 11. `classifyMethodKind` and `blockedMethodCommandTypes` are maintained independently with no cross-validation -- contributor friction

**File:** `internal/services/game/api/grpc/interceptors/telemetry.go:178-209`
**File:** `internal/services/game/api/grpc/interceptors/session_lock.go:23-48`

Both `classifyMethodKind` and `blockedMethodCommandTypes` maintain independent
lists of RPC method names. When a new RPC is added:
- A contributor must remember to add it to `blockedMethodCommandTypes` if it
  is a mutator that should be blocked during sessions.
- A contributor must remember to add it to `classifyMethodKind` if it is a
  read method.
- There is no test ensuring these lists are consistent or complete relative to
  the proto definitions.

The `blockedMethodCommandTypes` map at least has `ValidateSessionLockPolicyCoverage`
as a startup guard. The `classifyMethodKind` switch has no such guard.

**Proposal:** Add a test that extracts all `_FullMethodName` constants from
the generated proto packages and verifies each appears in exactly one of:
`classifyMethodKind` read list, `blockedMethodCommandTypes`, or an explicit
"unclassified-ok" set. This creates a single point of failure when protos
change.

---

### 12. `allActions` and `allResources` require manual sync -- contributor friction

**File:** `internal/services/game/domain/authz/policy.go:171-185`

The `allActions` and `allResources` slices enumerate all recognized values.
These must be manually updated when new actions or resources are added. Go
string-typed enums do not support `iota` or compile-time exhaustiveness.

The exhaustiveness tests catch staleness in the matrix rows, but only if the
new value is added to the `all*` slices first. If a contributor adds a new
`const ResourceScene Resource = "scene"` but forgets to add it to
`allResources`, the exhaustiveness tests pass silently.

**Proposal:** Add a compile-time or init-time registration pattern where
each const is added to a registry, or add a test that uses reflection or
code-generation to extract all exported `Action` and `Resource` constants
and verify they appear in the `all*` slices.

---

### 13. Interceptor ordering relies on doc comment convention, not enforcement -- missing best practice

**File:** `internal/services/game/api/grpc/interceptors/doc.go:1-22`
**File:** `internal/services/game/app/bootstrap_transport.go:149-164`

The interceptor ordering is documented in `doc.go` with a numbered list and a
"verify ordering in app/bootstrap_transport.go" note. The actual ordering is
in `newDefaultGRPCServer`. There is no test that the interceptor order matches
the documented order.

The documented order is:
1. metadata
2. internal_identity
3. telemetry
4. session_lock
5. error_conversion

The actual order in `ChainUnaryInterceptor`:
1. `grpcmeta.UnaryServerInterceptor` (metadata)
2. `InternalServiceIdentityUnaryInterceptor` (internal_identity)
3. `AuditInterceptor` (telemetry)
4. `SessionLockInterceptor` (session_lock)
5. `ErrorConversionUnaryInterceptor` (error_conversion)

These match. But the stream chain omits session_lock (finding #5). The
doc.go does not distinguish unary from stream ordering.

**Proposal:** Either add a test that verifies interceptor types/order via
reflection on the gRPC server options, or update the doc comment to note
that session lock is unary-only.

---

### 14. `RequireCharacterMutationPolicy` early-returns for non-member without ownership check -- correctness risk (low)

**File:** `internal/services/game/api/grpc/game/authz/policy.go:105`

```go
if reasonCode == ReasonAllowAccessLevel && actor.CampaignAccess != participant.CampaignAccessMember {
```

This short-circuits ownership resolution for owner/manager actors. While
correct (owners and managers can mutate any character), this means the
telemetry event for owner/manager character mutations never includes the
`owner_participant_id` attribute. This is a minor observability gap -- audit
logs for admin character edits do not record which character owner was
affected.

**Proposal:** Minor: consider resolving ownership even for privileged actors
and including it in telemetry, or document this as intentional.
