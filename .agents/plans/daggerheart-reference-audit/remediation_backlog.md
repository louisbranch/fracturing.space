# Remediation Backlog

Generated backlog synthesized from `audit_matrix.json` gap rows.

- Gap rows: 438
- Epics: 3

## Adversary Feature Parity

- Epic ID: `adversary-feature-parity`
- Priority: `p2`
- Gap rows: 129
- Boundary: Adversary runtime semantics beyond base stats, conditions, and damage application.
- Summary: Close the gap between adversary catalog entries and runtime adversary behavior, especially fear features, move semantics, and entry-specific rules.
- Kinds: adversary=129
- Audit areas: adversaries=129
- Gap classes: behavior=129
- Sample references: `adversary-acid-burrower`, `adversary-adult-flickerfly`, `adversary-apprentice-assassin`, `adversary-arch-necromancer`, `adversary-archer-guard`, `adversary-archer-squadron`, `adversary-assassin-poisoner`, `adversary-battle-box`

Contracts to touch:
- `internal/services/game/domain/systems/daggerheart/state.go`
- `internal/services/game/api/grpc/systems/daggerheart/damagetransport/`
- `internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/`
- `api/proto/systems/daggerheart/v1/content.proto`

Tests required:
- Domain tests for adversary feature activation and state transitions.
- gRPC integration tests for adversary feature execution paths.
- Scenario coverage for representative adversary fear-feature interactions.

Removal criteria:
- Remove entry-specific special cases once adversary feature execution is data-driven or otherwise uniformly modeled.

Representative code evidence:
- `api/proto/systems/daggerheart/v1/content.proto`
- `internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler.go`
- `internal/services/game/domain/systems/daggerheart/mechanics_manifest.go`
- `internal/services/game/domain/systems/daggerheart/state.go`

Representative test evidence:
- `internal/services/game/api/grpc/systems/daggerheart/adversaries_test.go`
- `internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler_test.go`

Representative docs:
- `docs/product/daggerheart-PRD.md`
- `docs/reference/daggerheart-event-timeline-contract.md`

## Ability Mapping And Semantic Audit

- Epic ID: `ability-mapping-and-semantic-audit`
- Priority: `p3`
- Gap rows: 189
- Boundary: Content-to-runtime semantic mapping for ability-shaped reference entries.
- Summary: Introduce an authoritative mapping layer from extracted ability rows to domain cards, class features, subclass features, or explicit runtime mechanics so ability coverage is proven instead of inferred.
- Kinds: ability=189
- Audit areas: domain_cards=189
- Gap classes: ambiguous_mapping=189
- Sample references: `ability-a-soldier-s-bond`, `ability-adjust-reality`, `ability-arcana-touched`, `ability-arcane-reflection`, `ability-armorer`, `ability-astral-projection`, `ability-banish`, `ability-bare-bones`

Contracts to touch:
- `internal/services/game/domain/systems/daggerheart/contentstore/contracts.go`
- `internal/services/game/domain/systems/daggerheart/mechanics_manifest.go`
- `api/proto/systems/daggerheart/v1/content.proto`
- `internal/tools/importer/content/daggerheart/v1/`

Tests required:
- Tooling tests that every ability row resolves to a stable runtime mapping or an explicit unsupported rationale.
- Content transport tests for any new ability-derived catalog or metadata fields.
- Targeted domain or transport tests for newly mapped ability mechanics.

Removal criteria:
- Remove temporary alias tables once every ability row points at an authoritative runtime or catalog contract.
- Drop provisional ambiguous-mapping status once the audit can prove semantic equivalence automatically.

Representative code evidence:
- `api/proto/systems/daggerheart/v1/content.proto`
- `internal/services/game/domain/systems/daggerheart/contentstore/contracts.go`
- `internal/services/game/domain/systems/daggerheart/mechanics_manifest.go`

Representative test evidence:
- `internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_test.go`

Representative docs:
- `docs/product/daggerheart-PRD.md`

## Item Use Modeling

- Epic ID: `item-use-modeling`
- Priority: `p3`
- Gap rows: 120
- Boundary: Inventory- and item-use semantics for Daggerheart content rows.
- Summary: Promote mechanically meaningful item and consumable entries into explicit runtime item-use behavior rather than leaving them as descriptive catalog text.
- Kinds: consumable=60, item=60
- Audit areas: equipment_items=120
- Gap classes: missing_model=120
- Sample references: `consumable-acidpaste`, `consumable-armor-stitcher`, `consumable-attune-potion`, `consumable-blinding-orb`, `consumable-blood-of-the-yorgi`, `consumable-bolster-potion`, `consumable-bonding-honey`, `consumable-bridge-seed`

Contracts to touch:
- `internal/services/game/domain/systems/daggerheart/`
- `internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/`
- `api/proto/systems/daggerheart/v1/content.proto`
- `api/proto/systems/daggerheart/v1/state.proto`

Tests required:
- Domain tests for representative item/consumable effect execution.
- Transport tests for inventory mutation and item-use commands.
- Importer/content tests for any new structured item effect fields.

Removal criteria:
- Remove content-only handling for mechanical items once runtime item-use commands or explicit non-goals replace it.

Representative code evidence:
- `api/proto/systems/daggerheart/v1/content.proto`
- `internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/`
- `internal/services/game/domain/systems/daggerheart/contentstore/contracts.go`
- `internal/services/game/storage/sqlite/daggerheartcontent/`
- `internal/tools/importer/content/daggerheart/v1/`

Representative test evidence:
- `internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_test.go`
- `internal/services/game/storage/sqlite/daggerheartcontent/store_content_test.go`

Representative docs:
- `docs/product/daggerheart-PRD.md`

