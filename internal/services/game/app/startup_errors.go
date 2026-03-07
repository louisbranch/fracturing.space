package server

import "fmt"

// startupPhase identifies a logical startup stage for typed bootstrap errors.
type startupPhase string

const (
	startupPhaseRegistries   startupPhase = "registries"
	startupPhaseNetwork      startupPhase = "network"
	startupPhaseStorage      startupPhase = "storage"
	startupPhaseDomain       startupPhase = "domain"
	startupPhaseSystems      startupPhase = "systems"
	startupPhaseDependencies startupPhase = "dependencies"
	startupPhaseTransport    startupPhase = "transport"
	startupPhaseRuntime      startupPhase = "runtime"
)

// StartupError wraps a startup failure with the phase and operation context.
type StartupError struct {
	Phase     startupPhase
	Operation string
	Err       error
}

// Error returns a startup-phase-prefixed error string.
func (e *StartupError) Error() string {
	if e == nil {
		return ""
	}
	if e.Operation == "" {
		return fmt.Sprintf("startup phase %s failed: %v", e.Phase, e.Err)
	}
	return fmt.Sprintf("startup phase %s failed during %s: %v", e.Phase, e.Operation, e.Err)
}

// Unwrap returns the underlying error.
func (e *StartupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// wrapStartupError annotates startup failures with phase and operation details.
func wrapStartupError(phase startupPhase, operation string, err error) error {
	if err == nil {
		return nil
	}
	return &StartupError{Phase: phase, Operation: operation, Err: err}
}
