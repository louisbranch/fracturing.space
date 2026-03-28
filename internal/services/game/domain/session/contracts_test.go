package session

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestFoldHandledTypes_ReturnsSessionEventContract(t *testing.T) {
	want := []event.Type{
		EventTypeStarted,
		EventTypeEnded,
		EventTypeGateOpened,
		EventTypeGateResponseRecorded,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
		EventTypeSceneActivated,
		EventTypeGMAuthoritySet,
		EventTypeCharacterControllerSet,
		EventTypeOOCOpened,
		EventTypeOOCPosted,
		EventTypeOOCReadyMarked,
		EventTypeOOCReadyCleared,
		EventTypeOOCClosed,
		EventTypeOOCResolved,
		EventTypeAITurnQueued,
		EventTypeAITurnRunning,
		EventTypeAITurnFailed,
		EventTypeAITurnCleared,
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

func TestDeciderHandledCommands_ReturnsSessionCommandContract(t *testing.T) {
	want := []command.Type{
		CommandTypeStart,
		CommandTypeEnd,
		CommandTypeGateOpen,
		CommandTypeGateRespond,
		CommandTypeGateResolve,
		CommandTypeGateAbandon,
		CommandTypeSpotlightSet,
		CommandTypeSpotlightClear,
		CommandTypeSceneActivate,
		CommandTypeGMAuthoritySet,
		CommandTypeCharacterControllerSet,
		CommandTypeOOCOpen,
		CommandTypeOOCPost,
		CommandTypeOOCReadyMark,
		CommandTypeOOCReadyClear,
		CommandTypeOOCClose,
		CommandTypeOOCResolve,
		CommandTypeAITurnQueue,
		CommandTypeAITurnStart,
		CommandTypeAITurnFail,
		CommandTypeAITurnClear,
	}

	if got := DeciderHandledCommands(); !testcontracts.EqualSlices(got, want) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", got, want)
	}
}

func TestSessionContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(sessionCommandContracts))
	for _, contract := range sessionCommandContracts {
		declaredCommandTypes = append(declaredCommandTypes, contract.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(sessionEventContracts))
	declaredProjection := make([]event.Type, 0, len(sessionEventContracts))
	for _, contract := range sessionEventContracts {
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
	if got, want := len(commandRegistry.ListDefinitions()), len(sessionCommandContracts); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(eventRegistry.ListDefinitions()), len(sessionEventContracts); got != want {
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

func TestFold_ReturnsErrorForUnknownEventType(t *testing.T) {
	_, err := Fold(State{}, event.Event{
		Type:        event.Type("session.unknown"),
		PayloadJSON: []byte(`{"ignored":true}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type, got nil")
	}
}

func TestValidateGateResponseRecordedPayload_MatchesShape(t *testing.T) {
	raw, err := json.Marshal(GateResponseRecordedPayload{
		GateID:        "gate-1",
		ParticipantID: "part-1",
		Decision:      "ready",
		Response:      map[string]any{"note": "locked in"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := validateGateResponseRecordedPayload(raw); err != nil {
		t.Fatalf("validateGateResponseRecordedPayload() error = %v", err)
	}
	if err := validateGateResponseRecordedPayload(json.RawMessage("{")); err == nil {
		t.Fatal("expected invalid payload error")
	}
}
