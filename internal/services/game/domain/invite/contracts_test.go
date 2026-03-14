package invite

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestRegisterRequiresRegistry(t *testing.T) {
	if err := RegisterCommands(nil); err == nil {
		t.Fatalf("expected error for nil command registry")
	}
	if err := RegisterEvents(nil); err == nil {
		t.Fatalf("expected error for nil event registry")
	}
}

func TestInviteContractTypeLists(t *testing.T) {
	emittable := EmittableEventTypes()
	if len(emittable) != 4 {
		t.Fatalf("EmittableEventTypes len = %d, want 4", len(emittable))
	}
	if emittable[0] != EventTypeCreated || emittable[1] != EventTypeClaimed || emittable[2] != EventTypeRevoked || emittable[3] != EventTypeUpdated {
		t.Fatalf("unexpected emittable events: %v", emittable)
	}

	commands := DeciderHandledCommands()
	if len(commands) != 4 {
		t.Fatalf("DeciderHandledCommands len = %d, want 4", len(commands))
	}
	if commands[0] != CommandTypeCreate || commands[1] != CommandTypeClaim || commands[2] != CommandTypeRevoke || commands[3] != CommandTypeUpdate {
		t.Fatalf("unexpected command list: %v", commands)
	}

	projectionTypes := ProjectionHandledTypes()
	foldTypes := FoldHandledTypes()
	if len(projectionTypes) != 4 || len(foldTypes) != 4 {
		t.Fatalf("projection/fold lengths = %d/%d, want 4/4", len(projectionTypes), len(foldTypes))
	}
	for i := range projectionTypes {
		if projectionTypes[i] != foldTypes[i] {
			t.Fatalf("projection and fold types differ at %d: %s vs %s", i, projectionTypes[i], foldTypes[i])
		}
	}
}

func TestInviteContractDeclarationsStayInParity(t *testing.T) {
	declaredCommandTypes := make([]command.Type, 0, len(inviteCommandContracts))
	for _, contract := range inviteCommandContracts {
		declaredCommandTypes = append(declaredCommandTypes, contract.definition.Type)
	}
	if testcontracts.HasDuplicates(declaredCommandTypes) {
		t.Fatalf("duplicate command declarations found: %v", declaredCommandTypes)
	}
	if !testcontracts.EqualSlices(DeciderHandledCommands(), declaredCommandTypes) {
		t.Fatalf("DeciderHandledCommands() = %v, want %v", DeciderHandledCommands(), declaredCommandTypes)
	}

	declaredEmittable := make([]event.Type, 0, len(inviteEventContracts))
	declaredProjection := make([]event.Type, 0, len(inviteEventContracts))
	for _, contract := range inviteEventContracts {
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
	if got, want := len(commandRegistry.ListDefinitions()), len(inviteCommandContracts); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if got, want := len(eventRegistry.ListDefinitions()), len(inviteEventContracts); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}
}

func TestFoldRecognizedEventsRejectCorruptPayload(t *testing.T) {
	tests := []event.Type{
		EventTypeCreated,
		EventTypeClaimed,
		EventTypeRevoked,
		EventTypeUpdated,
	}
	for _, typ := range tests {
		t.Run(string(typ), func(t *testing.T) {
			_, err := Fold(State{}, event.Event{
				Type:        typ,
				PayloadJSON: []byte(`{`),
			})
			if err == nil {
				t.Fatalf("expected fold error for corrupt payload")
			}
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Status
		wantOK  bool
		wantRaw string
	}{
		{name: "pending canonical", value: "pending", want: StatusPending, wantOK: true, wantRaw: statusPending},
		{name: "claimed enum", value: "INVITE_STATUS_CLAIMED", want: StatusClaimed, wantOK: true, wantRaw: statusClaimed},
		{name: "revoked uppercase", value: "REVOKED", want: StatusRevoked, wantOK: true, wantRaw: statusRevoked},
		{name: "invalid", value: "draft", want: StatusUnspecified, wantOK: false, wantRaw: ""},
		{name: "blank", value: " ", want: StatusUnspecified, wantOK: false, wantRaw: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeStatus(tc.value)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("NormalizeStatus(%q) = (%q, %v), want (%q, %v)", tc.value, got, ok, tc.want, tc.wantOK)
			}
			raw, rawOK := NormalizeStatusLabel(tc.value)
			if raw != tc.wantRaw || rawOK != tc.wantOK {
				t.Fatalf("NormalizeStatusLabel(%q) = (%q, %v), want (%q, %v)", tc.value, raw, rawOK, tc.wantRaw, tc.wantOK)
			}
		})
	}
}

func TestRegisterCommandsAndEvents_RejectUnknownTypes(t *testing.T) {
	commands := command.NewRegistry()
	if err := RegisterCommands(commands); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if _, err := commands.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.unknown"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}); !errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}

	events := event.NewRegistry()
	if err := RegisterEvents(events); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if _, err := events.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.unknown"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{}`),
	}); !errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}
}
