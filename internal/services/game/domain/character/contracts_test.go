package character

import (
	"reflect"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestFoldHandledTypes_ReturnsCharacterEventContract(t *testing.T) {
	want := []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeDeleted,
	}

	if got := FoldHandledTypes(); !testcontracts.EqualSlices(got, want) {
		t.Fatalf("FoldHandledTypes() = %v, want %v", got, want)
	}
}

func TestRegistryEventTypeHelpers_StayAlignedWithFoldContract(t *testing.T) {
	foldHandled := FoldHandledTypes()
	if got := EmittableEventTypes(); !testcontracts.EqualSlices(got, foldHandled) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", got, foldHandled)
	}
	if got := ProjectionHandledTypes(); !testcontracts.EqualSlices(got, foldHandled) {
		t.Fatalf("ProjectionHandledTypes() = %v, want %v", got, foldHandled)
	}
}

func TestDeciderHandledCommands_ReturnsCharacterCommandContract(t *testing.T) {
	want := []command.Type{
		CommandTypeCreate,
		CommandTypeUpdate,
		CommandTypeDelete,
	}

	if got := DeciderHandledCommands(); !testcontracts.EqualSlices(got, want) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", got, want)
	}
}

func TestCharacterContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(characterCommandContracts))
	for _, contract := range characterCommandContracts {
		declaredCommandTypes = append(declaredCommandTypes, contract.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(characterEventContracts))
	declaredProjection := make([]event.Type, 0, len(characterEventContracts))
	for _, contract := range characterEventContracts {
		if contract.emittable {
			declaredEmittable = append(declaredEmittable, contract.definition.Type)
		}
		if contract.projection {
			declaredProjection = append(declaredProjection, contract.definition.Type)
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
	if got, want := len(commandRegistry.ListDefinitions()), len(characterCommandContracts); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(eventRegistry.ListDefinitions()), len(characterEventContracts); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}
}

func TestRegisterCommands_RejectsNilRegistry(t *testing.T) {
	err := RegisterCommands(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "command registry is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "command registry is required")
	}
}

func TestRegisterEvents_RejectsNilRegistry(t *testing.T) {
	err := RegisterEvents(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "event registry is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "event registry is required")
	}
}

func TestFold_IgnoresUnknownEventType(t *testing.T) {
	original := State{
		Created:            true,
		CharacterID:        "char-1",
		Name:               "Aria",
		Kind:               "pc",
		OwnerParticipantID: "p-owner",
		ParticipantID:      "p-controller",
		Aliases:            []string{"alias-1"},
	}

	updated, err := Fold(original, event.Event{
		Type:        event.Type("character.unknown"),
		PayloadJSON: []byte(`{"ignored":true}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(updated, original) {
		t.Fatalf("fold updated state for unknown event: got %+v, want %+v", updated, original)
	}
}
