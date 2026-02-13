// Package telemetry emits operational telemetry events.
//
// The package provides a minimal emitter and severity vocabulary for recording
// read-only gRPC telemetry into storage.TelemetryStore. It does not model
// gameplay events, which live in the campaign event journal.
//
// Subpackages reserve namespaces for future telemetry work:
//   - telemetry/events: structured telemetry event schemas (not yet implemented)
//   - telemetry/metrics: metrics exporters and collectors (not yet implemented)
package telemetry
