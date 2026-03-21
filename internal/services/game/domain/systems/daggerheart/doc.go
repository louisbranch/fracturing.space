// Package daggerheart defines the reference Daggerheart system module.
//
// Ownership map:
//   - module registration (`module.go`, `registry_system.go`)
//   - system command/event contracts (`event_types.go`, payload/profile files)
//   - deciders and replay fold (`decider*.go`, `folder.go`)
//   - adapter glue from system events into projection state (`adapter*.go`)
//   - state factory and readiness hooks (`state_factory.go`,
//     `creation_workflow.go`)
//
// Focused sibling packages own adjacent concerns:
//   - `domain/`: pure deterministic mechanics and probability logic
//   - `profile/`: profile normalization, defaults, and readiness helpers
//   - `projectionstore/`: Daggerheart-owned gameplay projection contracts
//   - `contentstore/`: Daggerheart-owned catalog/content store contracts
//   - `content/`: imported catalog/reference data
//
// Reading order for contributors:
//  1. `module.go` for the registered command/event boundary,
//  2. the relevant `decider_*.go` file for the mechanic being changed,
//  3. `folder.go` and `adapter*.go` when replay or projection behavior matters,
//  4. `profile/`, `projectionstore/`, or `contentstore/` when the change is
//     really about contracts rather than mutation rules.
//
// Non-goals:
//   - transport ownership; gRPC behavior belongs under
//     `api/grpc/systems/daggerheart/`,
//   - shared storage ownership; Daggerheart vocabulary stays in the
//     system-owned contract packages listed above.
package daggerheart
