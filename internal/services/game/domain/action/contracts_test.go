package action

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestRegisterRequiresRegistry(t *testing.T) {
	if err := RegisterCommands(nil); err == nil {
		t.Fatalf("expected error for nil command registry")
	}
	if err := RegisterEvents(nil); err == nil {
		t.Fatalf("expected error for nil event registry")
	}
}

func TestActionContractTypeLists(t *testing.T) {
	emittable := EmittableEventTypes()
	if len(emittable) != 4 {
		t.Fatalf("EmittableEventTypes len = %d, want 4", len(emittable))
	}
	if emittable[0] != EventTypeRollResolved || emittable[1] != EventTypeOutcomeApplied || emittable[2] != EventTypeOutcomeRejected || emittable[3] != EventTypeNoteAdded {
		t.Fatalf("unexpected emittable events: %v", emittable)
	}

	commands := DeciderHandledCommands()
	if len(commands) != 4 {
		t.Fatalf("DeciderHandledCommands len = %d, want 4", len(commands))
	}
	if commands[0] != CommandTypeRollResolve || commands[1] != CommandTypeOutcomeApply || commands[2] != CommandTypeOutcomeReject || commands[3] != CommandTypeNoteAdd {
		t.Fatalf("unexpected command list: %v", commands)
	}

	foldTypes := FoldHandledTypes()
	if len(foldTypes) != 2 || foldTypes[0] != EventTypeRollResolved || foldTypes[1] != EventTypeOutcomeApplied {
		t.Fatalf("unexpected fold types: %v", foldTypes)
	}

	if got := ProjectionHandledTypes(); got != nil {
		t.Fatalf("ProjectionHandledTypes = %v, want nil for replay/audit-only action events", got)
	}
}

func TestBuildOutcomeEffectEvent_NormalizesAndDefaultsPayload(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID: "camp-1",
		Type:       CommandTypeOutcomeApply,
		ActorType:  command.ActorTypeSystem,
		ActorID:    "system-1",
	}
	evt := buildOutcomeEffectEvent(cmd, func() time.Time { return now }, OutcomeAppliedEffect{
		Type:       "  session.gate_opened  ",
		EntityType: "  session_gate  ",
		EntityID:   "  gate-1  ",
		SystemID:   "  daggerheart  ",
	})
	if evt.Type != event.Type("session.gate_opened") {
		t.Fatalf("Type = %s, want session.gate_opened", evt.Type)
	}
	if evt.EntityType != "session_gate" || evt.EntityID != "gate-1" {
		t.Fatalf("entity = %s/%s, want session_gate/gate-1", evt.EntityType, evt.EntityID)
	}
	if string(evt.PayloadJSON) != "{}" {
		t.Fatalf("PayloadJSON = %s, want {}", string(evt.PayloadJSON))
	}
	if evt.SystemID != "daggerheart" {
		t.Fatalf("SystemID = %q, want daggerheart", evt.SystemID)
	}
}

func TestFoldRecognizedEventsRejectCorruptPayload(t *testing.T) {
	tests := []event.Type{
		EventTypeRollResolved,
		EventTypeOutcomeApplied,
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

func TestValidateActionRequestAndRoll(t *testing.T) {
	if err := validateActionRequestAndRoll("", 1); !errors.Is(err, errActionRequestIDRequired) {
		t.Fatalf("expected request_id required error, got %v", err)
	}
	if err := validateActionRequestAndRoll("req-1", 0); !errors.Is(err, errActionRollSeqRequired) {
		t.Fatalf("expected roll_seq required error, got %v", err)
	}
	if err := validateActionRequestAndRoll("req-1", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOutcomeApplyEffects(t *testing.T) {
	validPayload, err := json.Marshal(map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	cases := []struct {
		name    string
		effects []OutcomeAppliedEffect
		wantErr bool
	}{
		{
			name: "valid effect",
			effects: []OutcomeAppliedEffect{
				{Type: "session.gate_opened", EntityType: "session_gate", EntityID: "gate-1", PayloadJSON: validPayload},
			},
		},
		{
			name: "missing type",
			effects: []OutcomeAppliedEffect{
				{EntityType: "session_gate", EntityID: "gate-1"},
			},
			wantErr: true,
		},
		{
			name: "missing entity type",
			effects: []OutcomeAppliedEffect{
				{Type: "session.gate_opened", EntityID: "gate-1"},
			},
			wantErr: true,
		},
		{
			name: "missing entity id",
			effects: []OutcomeAppliedEffect{
				{Type: "session.gate_opened", EntityType: "session_gate"},
			},
			wantErr: true,
		},
		{
			name: "invalid payload json",
			effects: []OutcomeAppliedEffect{
				{Type: "session.gate_opened", EntityType: "session_gate", EntityID: "gate-1", PayloadJSON: []byte(`{`)},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOutcomeApplyEffects(tc.effects)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
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
		Type:        command.Type("action.unknown"),
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
		Type:        event.Type("action.unknown"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "action",
		EntityID:    "req-1",
		PayloadJSON: []byte(`{}`),
	}); !errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}
}
