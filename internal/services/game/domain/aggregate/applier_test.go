package aggregate

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func TestFolderApply_UpdatesSessionGateState(t *testing.T) {
	applier := Folder{}
	state := State{}

	opened, err := applier.Fold(state, event.Event{
		Type:        event.Type("session.gate_opened"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
	})
	if err != nil {
		t.Fatalf("apply gate opened: %v", err)
	}
	updated, ok := opened.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if !updated.Session.GateOpen {
		t.Fatal("expected gate to be open")
	}
	if updated.Session.GateID != "gate-1" {
		t.Fatalf("gate id = %s, want %s", updated.Session.GateID, "gate-1")
	}

	closed, err := applier.Fold(updated, event.Event{
		Type:        event.Type("session.gate_resolved"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"approve"}`),
	})
	if err != nil {
		t.Fatalf("apply gate resolved: %v", err)
	}
	closedState, ok := closed.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if closedState.Session.GateOpen {
		t.Fatal("expected gate to be closed")
	}
}

type fakeSystemFolder struct{}

func (fakeSystemFolder) Apply(state any, _ event.Event) (any, error) {
	count := 0
	if existing, ok := state.(int); ok {
		count = existing
	}
	count++
	return count, nil
}

func (fakeSystemFolder) FoldHandledTypes() []event.Type { return nil }

type fakeSystemModule struct{}

func (fakeSystemModule) ID() string                                 { return "system-1" }
func (fakeSystemModule) Version() string                            { return "v1" }
func (fakeSystemModule) RegisterCommands(_ *command.Registry) error { return nil }
func (fakeSystemModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (fakeSystemModule) EmittableEventTypes() []event.Type          { return nil }
func (fakeSystemModule) Decider() module.Decider                    { return nil }
func (fakeSystemModule) Folder() module.Folder                      { return fakeSystemFolder{} }
func (fakeSystemModule) StateFactory() module.StateFactory          { return nil }

func TestFolderApply_RoutesSystemEvents(t *testing.T) {
	registry := module.NewRegistry()
	if err := registry.Register(fakeSystemModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	applier := Folder{SystemRegistry: registry}
	state := State{}
	key := module.Key{ID: "system-1", Version: "v1"}

	updated, err := applier.Fold(state, event.Event{
		Type:          event.Type("action.tested"),
		SystemID:      "system-1",
		SystemVersion: "v1",
	})
	if err != nil {
		t.Fatalf("apply system event: %v", err)
	}
	result, ok := updated.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	systemState, ok := result.Systems[key]
	if !ok {
		t.Fatal("expected system state entry")
	}
	if systemState.(int) != 1 {
		t.Fatalf("system state = %v, want 1", systemState)
	}
}

func TestFolderApply_ReturnsErrorForUnregisteredSystemEvents(t *testing.T) {
	applier := Folder{SystemRegistry: module.NewRegistry()}
	state := State{}

	_, err := applier.Fold(state, event.Event{
		Type:          event.Type("action.unregistered_system"),
		SystemID:      "unregistered-system",
		SystemVersion: "v1",
	})
	if err == nil {
		t.Fatal("expected error for unregistered system event")
	}
}

func TestFolderApply_PropagatesFoldError(t *testing.T) {
	applier := Folder{}
	state := State{}

	_, err := applier.Fold(state, event.Event{
		Type:        event.Type("campaign.created"),
		PayloadJSON: []byte(`{corrupt`),
	})
	if err == nil {
		t.Fatal("expected error for corrupt payload")
	}
}

func TestFolderApply_SkipsAuditOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("test.audit_event"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	applier := Folder{Events: registry}
	state := State{Campaign: campaign.State{Name: "unchanged"}}

	result, err := applier.Fold(state, event.Event{
		Type:        event.Type("test.audit_event"),
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("apply audit-only event: %v", err)
	}
	updated, ok := result.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	// Audit-only event should not modify state.
	if updated.Campaign.Name != "unchanged" {
		t.Fatalf("campaign name = %s, want unchanged", updated.Campaign.Name)
	}
}

func TestFolderFoldDispatchedTypes_ReturnsNonEmpty(t *testing.T) {
	applier := &Folder{}
	types := applier.FoldDispatchedTypes()
	if len(types) == 0 {
		t.Fatal("expected FoldDispatchedTypes to return non-empty slice")
	}
	// Verify no duplicate types in the dispatched set.
	seen := make(map[event.Type]bool)
	for _, et := range types {
		if seen[et] {
			t.Fatalf("duplicate dispatched type: %s", et)
		}
		seen[et] = true
	}
}

func TestFolderApply_UpdatesInviteState(t *testing.T) {
	applier := Folder{}
	state := State{}

	updated, err := applier.Fold(state, event.Event{
		Type:        event.Type("invite.created"),
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","status":"pending"}`),
	})
	if err != nil {
		t.Fatalf("apply invite created: %v", err)
	}
	result, ok := updated.(State)
	if !ok {
		t.Fatal("expected State result")
	}
	if result.Invites == nil {
		t.Fatal("expected invites map to be initialized")
	}
	inv, ok := result.Invites["inv-1"]
	if !ok {
		t.Fatal("expected invite state entry")
	}
	if !inv.Created {
		t.Fatal("expected invite to be marked created")
	}
	if inv.Status != "pending" {
		t.Fatalf("invite status = %s, want %s", inv.Status, "pending")
	}
}
