// Package modules owns registry composition for area modules.
//
// Start here when changing which feature areas mount, how shared runtime
// helpers are constructed once, or how startup dependency bundles are nested by
// owning area before being handed to composition.
//
// This package should not become a second feature layer:
//   - registry files choose module order and pass shared options,
//   - Dependencies stays nested by area ownership,
//   - production gateway/service graphs are built in the owning area packages,
//     not inline in the registry.
//
// For behavior inside one feature, continue into the owning
// internal/services/web/modules/<area> package instead of growing this layer.
package modules
