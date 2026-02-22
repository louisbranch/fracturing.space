// Package bridge provides the projection-side adapter registry and the API
// metadata bridge for pluggable game systems.
//
// Compare with the sibling package domain/module, which handles write-path
// module routing (commands to decider, events to projector). The two packages
// collaborate but own different concerns:
//
//   - module.Registry  — write-path module routing
//   - bridge.AdapterRegistry — projection-side adapters for system-specific read models
//   - bridge.Registry — API metadata bridge (proto enum mapping, version catalog)
package bridge
