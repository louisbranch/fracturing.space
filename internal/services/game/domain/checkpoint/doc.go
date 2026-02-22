// Package checkpoint provides replay checkpoint storage implementations.
//
// It exposes pluggable stores that let replay pipelines resume from prior
// checkpoints or intentionally replay from zero when checkpoints are disabled.
package checkpoint
