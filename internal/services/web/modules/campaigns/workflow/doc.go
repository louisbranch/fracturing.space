// Package workflow owns transport-facing character-creation workflow seams for
// campaigns.
//
// It sits between campaigns app services and system-specific workflow
// implementations:
//   - app services provide progress/catalog/profile reads plus step/reset mutations,
//   - workflow services resolve one installed system workflow and coordinate GET
//     vs POST transport behavior,
//   - system subpackages such as daggerheart own form parsing and view assembly
//     on workflow-local types.
//
// This package should stay focused on transport-owned workflow contracts and
// orchestration. It must not reintroduce app-owned page aggregates or module
// template ownership.
package workflow
