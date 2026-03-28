// Package module defines the pluggable game-system layer for the write model.
//
// Each game system (e.g. Daggerheart) is registered as a Module that contributes:
//   - command/event registration,
//   - domain decision logic for system commands,
//   - projection logic for system events in the aggregate,
//   - and state factories for per-system snapshots.
//
// This boundary is what lets new game systems be added without changing the
// core campaign/session/participant command flow.
//
// Compare with the sibling package domain/systems, which provides the
// projection-side adapter registry and the API metadata bridge (proto enum
// mapping). The two packages collaborate but own different concerns:
//
//   - module.Registry  — write-path module routing (commands → decider, events → projector)
//   - systems.AdapterRegistry — projection-side adapters for system-specific read models
//
// # Typed Wrappers
//
// The module boundary uses any-typed state to stay system-agnostic. Three
// generic helpers eliminate the boilerplate type assertions that would
// otherwise appear in every system implementation:
//
//   - TypedDecider[S]  — wraps a func(S, Command, now) Decision as a Decider
//   - TypedFolder[S]   — wraps a func(S, Event) (S, error) as a Folder
//   - FoldRouter[S]    — dispatches fold calls by event type to typed handler
//     functions registered via HandleFold, replacing manual type switches
//
// # Optional Extension Interfaces
//
// A Module may implement additional interfaces to participate in lifecycle
// hooks beyond basic command/event handling:
//
//   - CharacterReadinessProvider — supplies character-level readiness checks
//     used during session-start validation (implement when the system has
//     per-character prerequisites).
//   - SessionStartBootstrapProvider — emits system-specific bootstrap events
//     when a session starts (implement when the system needs initial state
//     seeded at session open).
//   - CommandTyper — maps system commands to their command type identifiers
//     (required when the module registers system-scoped command types).
package module
