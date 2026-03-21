// Package manifest declares the built-in game-system descriptors used by
// startup.
//
// A SystemDescriptor is the one place a built-in system should describe how it
// participates in runtime:
//
//   - BuildModule feeds the write-path registry builder.
//   - BuildMetadataSystem feeds transport-facing system metadata.
//   - BuildAdapter feeds projection apply and replay repair from the concrete
//     store source available at startup or replay time.
//
// Keep add/remove operations centered here so contributors do not need to wire
// the same system through separate startup lists by hand. Startup parity
// validation in internal/services/game/app refuses to boot if the module,
// metadata, and adapter surfaces drift apart.
//
// Built-in systems may still need dedicated projection backends, but that
// extraction logic should stay inside the owning descriptor rather than
// requiring callers to build manifest-owned store bundles.
package manifest
