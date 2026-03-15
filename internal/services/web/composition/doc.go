// Package composition wires root request-state and module registry inputs into
// the browser-facing app handler.
//
// Start here when changing how the root web service turns one
// principal.PrincipalResolver plus nested module dependencies into mounted
// public/protected module sets. This package should stay thin:
//   - it selects the root assembly flow,
//   - it passes shared runtime options into the module registry,
//   - it does not accumulate feature-local routing or backend wiring.
//
// If a change needs module-specific startup graphs, move that work into the
// owning area's composition entrypoint under modules/<area>/ instead.
package composition
