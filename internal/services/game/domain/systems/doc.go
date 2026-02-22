// Package systems provides the projection-side adapter registry and the API
// metadata bridge for pluggable game systems.
//
// Compare with the sibling package domain/module, which handles write-path
// module routing (commands to decider, events to projector). The two packages
// collaborate but own different concerns:
//
//   - module.Registry  — write-path module routing
//   - systems.AdapterRegistry — projection-side adapters for system-specific read models
//   - systems.Registry — API metadata bridge (proto enum mapping, version catalog)
package systems
