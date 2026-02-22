package audit

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Severity describes the audit severity level.
type Severity string

const (
	SeverityInfo  Severity = "INFO"
	SeverityWarn  Severity = "WARN"
	SeverityError Severity = "ERROR"
)

// Emitter records operational audit events.
type Emitter struct {
	store storage.AuditEventStore
	clock func() time.Time
}

// NewEmitter creates a new audit event emitter.
func NewEmitter(store storage.AuditEventStore) *Emitter {
	return &Emitter{store: store, clock: time.Now}
}

// Emit records an audit event. It is a no-op when the store is nil.
func (e *Emitter) Emit(ctx context.Context, evt storage.AuditEvent) error {
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
	return e.store.AppendAuditEvent(ctx, evt)
}
