// Package outcometransport owns the Daggerheart transport handlers that derive
// durable roll outcomes from already-resolved roll events.
//
// It keeps the root Daggerheart service thin while preserving the existing
// public gRPC surface. Session workflow orchestration still lives in the root
// package for now and composes this handler through a narrow dependency seam.
package outcometransport
