// Package events defines canonical game audit event names.
//
// The names intentionally remain stable (`telemetry.*`) because operational
// consumers already rely on these values.
package events

const (
	// GRPCRead captures durable audit events for read-only gRPC handlers.
	GRPCRead = "telemetry.grpc.read"
	// GRPCWrite captures durable audit events for write-path gRPC handlers.
	GRPCWrite = "telemetry.grpc.write"
	// AuthzDecision captures authorization allow/deny/override decisions.
	AuthzDecision = "telemetry.authz.decision"
)
