// Package integrity provides event hash and signing helpers used to protect the
// event journal's tamper-evident chain.
//
// Why this package exists:
// - It ensures each stored event carries a deterministic hash input.
// - It links events into a chain so replay order and authenticity can be verified.
// - It isolates cryptographic details from higher-level storage and replay code.
package integrity
