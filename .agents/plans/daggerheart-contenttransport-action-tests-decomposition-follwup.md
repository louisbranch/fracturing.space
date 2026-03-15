# ExecPlan: Daggerheart Contenttransport Action Test Follow-Up

## Status

`completed`

## Goal

Complete the next cleanup after the sessionrolltransport split by decomposing
remaining compendium asset descriptors and the large root action test suites that
still aggregate multiple responsibilities in a single file.

## Target Boundaries

- `descriptors_compendium_assets.go`
  - remove monolithic grouping of loot/weapon/armor/item/environment descriptors.
  - move each descriptor family into a dedicated `descriptors_compendium_assets_*.go` file.
- `actions_swap_loadout_flow_test.go`
  - isolate domain-command assertion tests into a domain-focused companion file.
- `actions_session_action_roll_test.go`
  - split validation-only coverage into `actions_session_action_roll_validation_test.go`.
- `actions_adversary_damage_test.go`
  - split validation coverage into `actions_adversary_damage_validation_test.go`.
- `actions_apply_attack_outcome_test.go`
  - split validation and success coverage into endpoint- and responsibility-owned files.

## Stable Contracts

- Descriptor names and behavior remain unchanged.
- `DaggerheartService` request/response contracts remain unchanged.
- No production behavior changes; tests only adjust ownership and file boundaries.

## Non-Goals

- Change transport/projection/domain logic.
- Change public APIs or protobuf contracts.

## Tasks

- [x] Decompose compendium asset descriptors into family-owned files.
- [x] Split each listed root action suite into smaller, responsibility-owned test files.
- [x] Run `make check` and targeted verification for impacted packages.

## Removal Criteria

- No action/test or descriptor family is held in a single oversized monolithic file.
- `make check` passes.
