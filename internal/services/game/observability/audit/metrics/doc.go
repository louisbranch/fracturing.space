// Package metrics reserves game-specific metrics integration points for audit
// instrumentation.
//
// Durable metrics are not emitted yet. This package is intentionally present to
// keep the metrics surface near game audit concerns.
package metrics

const (
	// AuditWritesEmittedTotal is reserved for future exported metric wiring.
	AuditWritesEmittedTotal = "game_audit_writes_emitted_total"
	// AuditWriteErrorsTotal is reserved for future exported metric wiring.
	AuditWriteErrorsTotal = "game_audit_write_errors_total"
)
