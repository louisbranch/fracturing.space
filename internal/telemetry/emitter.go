package telemetry

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/storage"
)

// Severity describes the telemetry severity level.
type Severity string

const (
	SeverityInfo  Severity = "INFO"
	SeverityWarn  Severity = "WARN"
	SeverityError Severity = "ERROR"
)

// Emitter records operational telemetry events.
type Emitter struct {
	store storage.TelemetryStore
	clock func() time.Time
}

// NewEmitter creates a new telemetry emitter.
func NewEmitter(store storage.TelemetryStore) *Emitter {
	return &Emitter{store: store, clock: time.Now}
}

// Emit records a telemetry event. It is a no-op when the store is nil.
func (e *Emitter) Emit(ctx context.Context, evt storage.TelemetryEvent) error {
	if e == nil || e.store == nil {
		return nil
	}
	if evt.Timestamp.IsZero() {
		if e.clock == nil {
			evt.Timestamp = time.Now().UTC()
		} else {
			evt.Timestamp = e.clock().UTC()
		}
	}
	return e.store.AppendTelemetryEvent(ctx, evt)
}
