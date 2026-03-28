// Package module defines the narrow contract shared between root web
// assembly and area-owned modules.
//
// Start here when you need to understand what the root app is allowed to know
// about a feature area. This package should stay intentionally small:
//   - Module and Mount define the only transport contract that the root mux
//     relies on,
//   - Viewer describes the shared app-shell chrome data shape,
//   - request-state callback contracts belong in principal/, not here,
//   - feature startup wiring, route ownership, and backend clients do not
//     belong here.
//
// If a change needs more than these shared contract types, it usually belongs
// in principal/, app/, or the owning module package instead.
package module
