// Package fork defines campaign fork domain contracts and command/event helpers.
//
// A fork creates a new campaign from an existing one at a specific event
// sequence. The new campaign replays events up to the fork point and then
// evolves independently, sharing no further state with the parent. Fork
// metadata tracks parent-child lineage so operators and UI can display
// campaign heritage and navigate between related campaigns.
package fork
