// Package telemetry provides observability for the Fracturing.Space.
//
// This package separates two distinct concerns:
//
// # Game Events
//
// Game events are the canonical event journal that captures gameplay actions
// for replay and state derivation. They are stored separately from telemetry.
//
// # Operational Metrics (telemetry/metrics)
//
// Operational metrics capture system health and performance:
//   - Request latency
//   - Error rates
//   - API usage patterns
//   - Resource utilization
//
// These metrics support monitoring, alerting, and capacity planning.
//
// # Design Philosophy
//
// Separating game events from operational metrics ensures:
//   - Game event storage can be optimized for replay/analysis
//   - Operational metrics can map to OpenTelemetry later
//   - Different retention policies for each concern
//   - Clear ownership boundaries
package telemetry
