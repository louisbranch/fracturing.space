package scene

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestFoldHandledTypes_ReturnsSceneEventContract(t *testing.T) {
	want := []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeEnded,
		EventTypeCharacterAdded,
		EventTypeCharacterRemoved,
		EventTypeGateOpened,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
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

func TestDeciderHandledCommands_ReturnsSceneCommandContract(t *testing.T) {
	want := []command.Type{
		CommandTypeCreate,
		CommandTypeUpdate,
		CommandTypeEnd,
		CommandTypeCharacterAdd,
		CommandTypeCharacterRemove,
		CommandTypeCharacterTransfer,
		CommandTypeTransition,
		CommandTypeGateOpen,
		CommandTypeGateResolve,
		CommandTypeGateAbandon,
		CommandTypeSpotlightSet,
		CommandTypeSpotlightClear,
	}

	if got := DeciderHandledCommands(); !testcontracts.EqualSlices(got, want) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", got, want)
	}
}

func TestSceneContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(sceneCommandContracts))
	for _, contract := range sceneCommandContracts {
		declaredCommandTypes = append(declaredCommandTypes, contract.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(sceneEventContracts))
	declaredProjection := make([]event.Type, 0, len(sceneEventContracts))
	for _, contract := range sceneEventContracts {
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
	if got, want := len(commandRegistry.ListDefinitions()), len(sceneCommandContracts); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(eventRegistry.ListDefinitions()), len(sceneEventContracts); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}
}

func TestValidatePayloadFunctions_AcceptValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		validate func(json.RawMessage) error
		payload  string
	}{
		{"create", validateCreatePayload, `{"scene_id":"s1","name":"x","character_ids":["c1"]}`},
		{"update", validateUpdatePayload, `{"scene_id":"s1","name":"x"}`},
		{"end", validateEndPayload, `{"scene_id":"s1"}`},
		{"characterAdded", validateCharacterAddedPayload, `{"scene_id":"s1","character_id":"c1"}`},
		{"characterRemoved", validateCharacterRemovedPayload, `{"scene_id":"s1","character_id":"c1"}`},
		{"characterTransfer", validateCharacterTransferPayload, `{"source_scene_id":"s1","target_scene_id":"s2","character_id":"c1"}`},
		{"transition", validateTransitionPayload, `{"source_scene_id":"s1","new_scene_id":"s2","name":"x"}`},
		{"gateOpened", validateGateOpenedPayload, `{"scene_id":"s1","gate_id":"g1","gate_type":"decision"}`},
		{"gateResolved", validateGateResolvedPayload, `{"scene_id":"s1","gate_id":"g1"}`},
		{"gateAbandoned", validateGateAbandonedPayload, `{"scene_id":"s1","gate_id":"g1"}`},
		{"spotlightSet", validateSpotlightSetPayload, `{"scene_id":"s1","spotlight_type":"gm"}`},
		{"spotlightCleared", validateSpotlightClearedPayload, `{"scene_id":"s1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.validate(json.RawMessage(tt.payload)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePayloadFunctions_RejectInvalidJSON(t *testing.T) {
	invalid := json.RawMessage(`{bad json`)
	validators := []func(json.RawMessage) error{
		validateCreatePayload,
		validateUpdatePayload,
		validateEndPayload,
		validateCharacterAddedPayload,
		validateCharacterRemovedPayload,
		validateCharacterTransferPayload,
		validateTransitionPayload,
		validateGateOpenedPayload,
		validateGateResolvedPayload,
		validateGateAbandonedPayload,
		validateSpotlightSetPayload,
		validateSpotlightClearedPayload,
	}
	for i, v := range validators {
		if err := v(invalid); err == nil {
			t.Fatalf("validator[%d] expected error for invalid JSON", i)
		}
	}
}

func TestRegisterCommands_RejectsNilRegistry(t *testing.T) {
	err := RegisterCommands(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterEvents_RejectsNilRegistry(t *testing.T) {
	err := RegisterEvents(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
