# P05: Provider Abstraction and OpenAI Adapter Boundary

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the generic provider contracts and the OpenAI implementation to determine whether the AI service is future-provider-ready, whether OpenAI-specific details are isolated cleanly, and whether external API behavior is easy to reason about and test.

Primary scope:

- `internal/services/ai/provider`
- `internal/services/ai/provider/openai`

## Progress

- [x] (2026-03-23 04:39Z) Reviewed provider contracts, app wiring, service call sites, and OpenAI adapter files across OAuth, model listing, invocation, Responses translation, and schema shaping.
- [x] (2026-03-23 04:41Z) Verified provider baseline with `go test ./internal/services/ai/provider ./internal/services/ai/provider/openai`.
- [x] (2026-03-23 04:48Z) Synthesized provider-boundary findings, future-provider cleanup needs, and the target cutover order.

## Surprises & Discoveries

- The provider surface is smaller than the service and transport surfaces, but more OpenAI-shaped than the package names suggest.
- The same OpenAI concrete type currently spans three seams at once: generic invoke, generic model listing, and orchestration turn execution.
- The strict-schema policy is correctly kept inside `provider/openai`, but the helper itself has no focused unit tests despite being a brittle compatibility rule.
- There are effectively two OpenAI OAuth implementations in the repo: the real provider-local adapter and a default app-local fallback adapter.

## Decision Log

- Decision: Keep `internal/services/ai/provider` as the shared vocabulary package, but shrink it to durable cross-provider concepts instead of carrying OpenAI-shaped auth and metadata details.
  Rationale: The package still has real value for provider identity and usage vocabulary, but several current contracts are broader than necessary and force future providers to fit OpenAI terminology.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P05 is complete for planning purposes. The generic provider seam is not yet clean enough to call future-provider-ready: auth material is named after credential secrets even when the runtime is holding provider-grant access tokens, the generic model shape exposes OpenAI listing metadata, and the OpenAI implementation is directly coupled to orchestration. The refactor target is a slimmer provider vocabulary, separate adapters per consumer seam, and one provider-local home for OpenAI OAuth/runtime behavior.

## Context and Orientation

Use:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `internal/services/ai/provider/openai/doc.go`

## Plan of Work

Inspect:

- provider interface clarity and minimality
- OpenAI-specific leakage into generic packages
- OAuth vs invocation vs model-listing cohesion
- request/response schema assembly
- usage reporting and error normalization
- adapter test seams and fakeability

## Current Findings

### F01: Generic provider auth inputs are named around credentials even when the runtime is passing access tokens

Category: missing best practice, contributor-clarity and maintainability risk

Evidence:

- `internal/services/ai/provider/contracts.go:57-77` names the auth field `CredentialSecret` in both `InvokeInput` and `ListModelsInput`.
- `internal/services/ai/service/invocation.go:109-115` passes the resolved invoke token into `CredentialSecret`.
- `internal/services/ai/service/campaign_orchestration.go:186-195` passes the same runtime token into orchestration as `CredentialSecret`.
- `internal/services/ai/service/agent.go:178-183` passes the resolved model-list token through `ListModelsInput{CredentialSecret: token}`.

Impact:

- The generic adapter seam leaks one auth-source implementation detail into every provider caller.
- Contributors have to remember that `CredentialSecret` sometimes means decrypted BYO key and sometimes means provider-grant access token.
- A second provider would either inherit misleading names or require more translation code around the generic seam.

Refactor direction:

- Rename the generic invoke/model auth field to `AccessToken` or `AuthToken`.
- Keep auth-source resolution in the service layer; adapters should only receive the runtime bearer material they need.
- Avoid reintroducing credential-vs-grant vocabulary at the provider seam.

### F02: The generic model contract is shaped by OpenAI listing metadata instead of the stable product contract

Category: anti-pattern

Evidence:

- `internal/services/ai/provider/contracts.go:79-84` defines `provider.Model` as `ID`, `OwnedBy`, and `Created`.
- `internal/services/ai/service/agent.go:186-191` sorts the generic model list by `Created`, then by `ID`.
- `api/proto/ai/v1/service.proto:244-247` only exposes `id` and `owned_by`; `created` never leaves the service boundary.
- `internal/services/ai/provider/openai/models.go:26-37` fills the generic model with OpenAI-native listing fields.

Impact:

- Future providers are forced to synthesize or reinterpret provider-specific metadata just to satisfy the generic contract.
- The service layer is applying ordering policy based on metadata that is not part of the public AI contract.
- The current seam is broader than the user-facing feature actually needs.

Refactor direction:

- Shrink the generic model descriptor to the fields the product truly needs.
- If provider-specific metadata matters later, carry it as explicitly optional provider-local data instead of making it part of the shared contract.
- Move any provider-specific ordering policy closer to the adapter or make ordering explicit in one service-owned comparator that does not depend on hidden OpenAI semantics.

### F03: `InvokeAdapter` is carrying three responsibilities and directly depends on orchestration

Category: anti-pattern, maintainability and testability risk

Evidence:

- `internal/services/ai/provider/openai/invoke.go:20-24` states that `InvokeAdapter` implements `provider.InvocationAdapter`, `orchestration.Provider`, and `provider.ModelAdapter`.
- `internal/services/ai/provider/openai/invoke.go:9` imports `internal/services/ai/orchestration`.
- `internal/services/ai/app/server.go:120-130` wires one `openAIAdapter` instance into invocation, orchestration tool runtime, and model listing maps.

Impact:

- The OpenAI boundary is not actually isolated by capability; one concrete type spans unrelated consumer seams.
- The provider package now depends outward into orchestration, which makes the dependency direction harder for new contributors to reason about.
- Tests for one capability must live next to a struct that also owns unrelated capabilities.

Refactor direction:

- Split OpenAI capability adapters by consumer seam:
  - OAuth adapter
  - generic invoke adapter
  - generic model-list adapter
  - orchestration-specific Responses runtime adapter
- Keep shared HTTP/request helpers internal to `provider/openai`, but stop exporting or wiring one omnibus adapter for everything.
- Prefer the orchestration-specific adapter to depend on a small provider-local client/helper rather than making the base invoke adapter import orchestration directly.

### F04: The OAuth contract advertises universal revoke support, but OpenAI revocation is effectively optional/no-op and duplicated outside the provider package

Category: anti-pattern

Evidence:

- `internal/services/ai/provider/contracts.go:8-14` requires every `OAuthAdapter` to implement `RevokeToken`.
- `internal/services/ai/provider/openai/oauth.go:100-106` returns `nil` with a comment that OpenAI revocation endpoint support is optional at this phase boundary.
- `internal/services/ai/app/openai_oauth_default.go:11-48` contains a second OpenAI OAuth implementation outside `provider/openai`.

Impact:

- The generic contract claims a capability that the main concrete provider does not truly implement.
- Contributors have to reason about OpenAI OAuth behavior across two packages instead of one provider-local boundary.
- Service logic cannot distinguish “provider supports revoke and it succeeded” from “provider has no revoke endpoint so we silently skipped it.”

Refactor direction:

- Decide explicitly whether revoke is:
  - a required provider capability, or
  - an optional capability with explicit service fallback behavior.
- Keep OpenAI OAuth behavior in `provider/openai`; if the default app-local flow is really a brokered facade, give it its own package and name instead of presenting it as another OpenAI adapter.

### F05: OpenAI Responses translation still relies on ad-hoc maps and anonymous payload shapes, and the strict-schema helper lacks focused tests

Category: missing best practice, testability risk

Evidence:

- `internal/services/ai/provider/openai/invoke.go:67-127` builds tool/runtime request bodies with nested `map[string]any`.
- `internal/services/ai/provider/openai/responses.go:15-36` and `:78-107` decode the Responses API into an internal anonymous payload struct and generic request helper.
- `internal/services/ai/provider/openai/schema.go:8-57` enforces strict schema policy recursively.
- `internal/services/ai/provider/openai/invoke_adapter_test.go` is large (418 lines), but there are no direct tests for `schema.go` despite it holding provider-compatibility policy.

Impact:

- Request/response translation is harder to scan and modify than it needs to be.
- Strict-schema regressions would be caught only indirectly through broad adapter tests.
- Contributors have to reconstruct the OpenAI request contract from loose maps rather than typed internal request/response helpers.

Refactor direction:

- Introduce small internal typed request/response helper structs for the two Responses API paths:
  - plain text invocation
  - orchestration tool-step execution
- Keep `openAIToolSchema` provider-local, but add direct table-driven tests for nested object/array cases and required/additionalProperties behavior.
- Centralize output-text/tool-call extraction so both invoke paths use one translator instead of partially duplicated logic.

### F06: `TokenExchangeResult.LastRefreshError` is dead generic surface area

Category: missing best practice

Evidence:

- `internal/services/ai/provider/contracts.go:39-45` exposes `LastRefreshError`.
- The field has no meaningful runtime consumers in the AI service; refresh failure state is recorded through the `providergrant` domain instead.

Impact:

- The generic contract carries lifecycle language that belongs to the domain package, not to the provider adapter result.
- Contributors reading the interface have to ask whether refresh errors should be returned via the field, via `error`, or via both.

Refactor direction:

- Remove `LastRefreshError` from `TokenExchangeResult`.
- Keep refresh-failure state modeling in `providergrant.RecordRefreshFailure(...)` and the service layer that applies it.

### F07: One provider-local boundary is already good and should be preserved

Category: best practice already employed, preserve as-is

Evidence:

- `internal/services/ai/provider/openai/schema.go:8-57` keeps strict-schema shaping inside the OpenAI package instead of widening the shared `provider` package.
- `internal/services/ai/provider/usage.go:3-27` keeps shared usage accounting small and durable.

Impact:

- Provider-specific quirks are not automatically leaking into the generic provider vocabulary.
- The existing package split can support a cleaner refactor if the shared contracts are narrowed rather than widened.

Refactor direction:

- Preserve provider-local ownership for OpenAI-specific request shaping and Responses quirks.
- Keep `provider.Usage` as a small shared value object unless a concrete multi-provider need appears.

## Concrete Steps

1. Map every exported provider contract to its callers.
2. Identify where OpenAI-specific assumptions leak outward.
3. Review request construction and response translation for readability and testability.
4. Record any breaking contract cleanup needed for future providers.

## Target Provider Seam Shape

Keep these concepts in `internal/services/ai/provider`:

- provider identity (`provider.Provider`)
- small shared usage accounting (`provider.Usage`)
- only the minimal cross-provider inputs/outputs that real consumers need

Refactor the rest toward consumer-specific, capability-specific seams:

1. Narrow auth vocabulary at the provider boundary.
   - adapters receive runtime bearer material, not credential-source terminology
2. Shrink the generic model descriptor.
   - keep only stable product-facing fields
   - move provider-specific listing metadata out of the shared contract
3. Split the current OpenAI omnibus adapter by responsibility.
   - OAuth lifecycle
   - generic invoke
   - generic model discovery
   - orchestration Responses runtime
4. Keep OpenAI-specific request shaping in `provider/openai`, but make it easier to read and test.
   - typed internal request/response helpers
   - one shared output translator
   - direct strict-schema tests
5. Make optional provider capabilities explicit.
   - revoke support should be modeled intentionally instead of hidden behind no-op implementations

## Cutover Order

1. Remove dead generic contract surface such as `TokenExchangeResult.LastRefreshError`.
2. Rename `CredentialSecret` at the generic provider seams to token-oriented vocabulary and rewire service/orchestration callers.
3. Shrink the generic model descriptor and remove service-side dependence on OpenAI `Created` metadata.
4. Split the current OpenAI `InvokeAdapter` into smaller capability adapters while reusing shared provider-local HTTP helpers.
5. Decide the revoke-capability model and move any non-provider-local OpenAI OAuth fallback out of the app package into an explicitly named boundary.
6. Add focused `schema.go` and response-translation tests after the helper boundaries are smaller.

## Validation and Acceptance

- `go test ./internal/services/ai/provider ./internal/services/ai/provider/openai`
- `go test ./internal/services/ai/...`

Acceptance:

- target generic provider contract is explicit
- OpenAI-only behavior is isolated or flagged
- test seam improvements are named concretely

## Idempotence and Recovery

- Prefer shrinking generic interfaces over widening them unless a second-provider use case is already concrete.

## Artifacts and Notes

- Record any adapter logic that should move into shared helpers or be kept provider-local on purpose.
- The contributor map should eventually reflect any split between generic invoke/model adapters and orchestration-specific OpenAI runtime adapters.

## Interfaces and Dependencies

Track any proposed changes to:

- provider adapter interfaces
- usage/model types
- OpenAI request/response helper contracts
- `app` wiring for default OpenAI OAuth behavior
