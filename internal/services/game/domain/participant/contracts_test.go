package participant

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestParticipantContractTypeLists(t *testing.T) {
	emittable := EmittableEventTypes()
	wantEmittable := []event.Type{
		EventTypeJoined,
		EventTypeUpdated,
		EventTypeLeft,
		EventTypeBound,
		EventTypeUnbound,
		EventTypeSeatReassigned,
	}
	if !testcontracts.EqualSlices(emittable, wantEmittable) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", emittable, wantEmittable)
	}

	commands := DeciderHandledCommands()
	wantCommands := []command.Type{
		CommandTypeJoin,
		CommandTypeUpdate,
		CommandTypeLeave,
		CommandTypeBind,
		CommandTypeUnbind,
		CommandTypeSeatReassign,
	}
	if !testcontracts.EqualSlices(commands, wantCommands) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", commands, wantCommands)
	}

	projection := ProjectionHandledTypes()
	if !testcontracts.EqualSlices(projection, wantEmittable) {
		t.Fatalf("ProjectionHandledTypes() = %v, want %v", projection, wantEmittable)
	}

	foldTypes := FoldHandledTypes()
	wantFold := []event.Type{
		EventTypeJoined,
		EventTypeUpdated,
		EventTypeLeft,
		EventTypeBound,
		EventTypeUnbound,
		EventTypeSeatReassigned,
	}
	if !testcontracts.EqualSlices(foldTypes, wantFold) {
		t.Fatalf("FoldHandledTypes() = %v, want %v", foldTypes, wantFold)
	}
}

func TestParticipantContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(participantCommandRegistrations))
	for _, registration := range participantCommandRegistrations {
		declaredCommandTypes = append(declaredCommandTypes, registration.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(participantEventRegistrations))
	declaredProjection := make([]event.Type, 0, len(participantEventRegistrations))
	for _, registration := range participantEventRegistrations {
		if registration.emittable {
			declaredEmittable = append(declaredEmittable, registration.definition.Type)
		}
		if registration.projection {
			declaredProjection = append(declaredProjection, registration.definition.Type)
		}
	}
	if testcontracts.HasDuplicates(declaredEmittable) {
		t.Fatalf("duplicate emittable event declarations found: %v", declaredEmittable)
	}
	if testcontracts.HasDuplicates(declaredProjection) {
		t.Fatalf("duplicate projection event declarations found: %v", declaredProjection)
	}
	if !testcontracts.EqualSlices(EmittableEventTypes(), declaredEmittable) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", EmittableEventTypes(), declaredEmittable)
	}
	if !testcontracts.EqualSlices(ProjectionHandledTypes(), declaredProjection) {
		t.Fatalf("ProjectionHandledTypes() = %v, want %v", ProjectionHandledTypes(), declaredProjection)
	}

	commandRegistry := command.NewRegistry()
	if err := RegisterCommands(commandRegistry); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if got, want := len(commandRegistry.ListDefinitions()), len(participantCommandRegistrations); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(eventRegistry.ListDefinitions()), len(participantEventRegistrations); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}
}

func TestRegisterRequiresRegistry(t *testing.T) {
	if err := RegisterCommands(nil); err == nil {
		t.Fatalf("expected error for nil command registry")
	}
	if err := RegisterEvents(nil); err == nil {
		t.Fatalf("expected error for nil event registry")
	}
}

func TestFoldUnknownTypeReturnsError(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        event.Type("participant.unknown"),
		PayloadJSON: []byte(`{"ignored":true}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type, got nil")
	}
}
