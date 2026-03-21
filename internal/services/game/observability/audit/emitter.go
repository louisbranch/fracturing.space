package audit

import (
	"context"
	"errors"
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

var errEnabledAuditStoreRequired = errors.New("audit store is required when audit is enabled")

// Policy declares whether durable audit writes are enabled for a runtime seam.
type Policy struct {
	enabled bool
	store   storage.AuditEventStore
}

// EnabledPolicy turns durable audit writes on for the provided store.
func EnabledPolicy(store storage.AuditEventStore) Policy {
	return Policy{enabled: true, store: store}
}

// DisabledPolicy turns durable audit writes off explicitly.
func DisabledPolicy() Policy {
	return Policy{}
}

// Enabled reports whether the runtime seam is configured to emit audit events.
func (p Policy) Enabled() bool {
	return p.enabled
}

// Store exposes the backing audit store when audit is enabled.
func (p Policy) Store() storage.AuditEventStore {
	if !p.enabled {
		return nil
	}
	return p.store
}

// Emitter records operational audit events.
type Emitter struct {
	policy Policy
	clock  func() time.Time
}

// NewEmitter creates a new audit event emitter.
func NewEmitter(policy Policy) *Emitter {
	return &Emitter{policy: policy, clock: time.Now}
}

// Emit records an audit event. It is a no-op only when audit is explicitly
// disabled; enabled-without-store is a wiring error.
func (e *Emitter) Emit(ctx context.Context, evt storage.AuditEvent) error {
	if e == nil || !e.policy.enabled {
		return nil
	}
	if e.policy.store == nil {
		return errEnabledAuditStoreRequired
	}
	if evt.Timestamp.IsZero() {
		if e.clock == nil {
			evt.Timestamp = time.Now().UTC()
		} else {
			evt.Timestamp = e.clock().UTC()
		}
	}
	return e.policy.store.AppendAuditEvent(ctx, evt)
}
