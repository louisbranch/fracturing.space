# Seed Findings

These are confirmed starting findings from the initial audit reconnaissance.
They are not the full audit result.

## 1. Character creation workflow ordering drift

- The extracted reference uses the sequence:
  `class/subclass -> heritage -> traits -> details -> equipment -> background -> experiences -> domain cards -> connections`.
- The current implementation in
  `internal/services/game/domain/bridge/daggerheart/creation_workflow.go` and
  `internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go`
  still orders equipment before details/background.
- `docs/reference/daggerheart-creation-workflow.md` already documents the
  reference ordering, so this is both a behavior gap and repo-doc drift target.

Suggested follow-up epic: `creation-workflow-alignment`

## 2. Companion modeling is missing

- The Beastbound subclass in the extracted reference requires a companion.
- The repo has content support for beastforms and companion experiences, but no
  clear companion sheet/profile model in the Daggerheart profile or creation
  workflow surfaces.

Suggested follow-up epic: `companion-modeling`

## 3. Mechanics manifest is useful but not sufficient

- `internal/services/game/domain/bridge/daggerheart/mechanics_manifest.go`
  provides a starting mechanic inventory and currently derives a COMPLETE
  implementation stage.
- That manifest does not guarantee that every corpus entry or every normative
  clause was checked, and it already leaves areas like companions and beastforms
  pending.

Suggested follow-up epic: `audit-manifest-reconciliation`
