package journalimport

import (
	"context"
	"fmt"

	domainwrite "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const defaultBatchSize = 200

// Importer appends already-authoritative historical events in order and applies
// them through the normal runtime intent filter.
type Importer interface {
	Import(ctx context.Context, events []event.Event) error
}

// Service imports historical journal events behind one explicit write seam.
type Service struct {
	events    storage.EventAppender
	applier   projection.Applier
	runtime   *domainwrite.Runtime
	registry  *event.Registry
	batchSize int
}

// NewService constructs the centralized historical event importer.
func NewService(events storage.EventAppender, applier projection.Applier, runtime *domainwrite.Runtime, registry *event.Registry) Service {
	return Service{
		events:    events,
		applier:   applier,
		runtime:   runtime,
		registry:  registry,
		batchSize: defaultBatchSize,
	}
}

// Import appends the provided historical events in deterministic order using
// bounded batches and applies stored events through the runtime intent filter.
func (s Service) Import(ctx context.Context, events []event.Event) error {
	if len(events) == 0 {
		return nil
	}
	if s.events == nil {
		return fmt.Errorf("event store is not configured")
	}
	for start := 0; start < len(events); start += s.batchLimit() {
		end := start + s.batchLimit()
		if end > len(events) {
			end = len(events)
		}
		stored, err := s.appendBatch(ctx, events[start:end])
		if err != nil {
			return err
		}
		if err := s.applyStored(ctx, stored); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) batchLimit() int {
	if s.batchSize <= 0 {
		return defaultBatchSize
	}
	return s.batchSize
}

func (s Service) appendBatch(ctx context.Context, batch []event.Event) ([]event.Event, error) {
	validated := make([]event.Event, 0, len(batch))
	for _, evt := range batch {
		validatedEvent, err := s.validate(evt)
		if err != nil {
			return nil, err
		}
		validated = append(validated, validatedEvent)
	}

	if ba, ok := s.events.(storage.BatchEventAppender); ok {
		stored, err := ba.BatchAppendEvents(ctx, validated)
		if err != nil {
			return nil, fmt.Errorf("append imported events: %w", err)
		}
		return stored, nil
	}

	stored := make([]event.Event, 0, len(validated))
	for _, evt := range validated {
		storedEvent, err := s.events.AppendEvent(ctx, evt)
		if err != nil {
			return nil, fmt.Errorf("append imported event: %w", err)
		}
		stored = append(stored, storedEvent)
	}
	return stored, nil
}

func (s Service) validate(evt event.Event) (event.Event, error) {
	if s.registry == nil {
		return evt, nil
	}
	validated, err := s.registry.ValidateForAppend(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("validate imported event %s: %w", evt.Type, err)
	}
	return validated, nil
}

func (s Service) applyStored(ctx context.Context, stored []event.Event) error {
	if !s.runtime.InlineApplyEnabled() {
		return nil
	}
	shouldApply := s.runtime.ShouldApply()
	for _, evt := range stored {
		if !shouldApply(evt) {
			continue
		}
		if err := s.applier.Apply(ctx, evt); err != nil {
			return fmt.Errorf("apply imported event %s: %w", evt.Type, err)
		}
	}
	return nil
}
