package aggregate

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestCoreFoldEntries_HasExpectedDomainEntries(t *testing.T) {
	entries := coreFoldEntries()
	if len(entries) != 6 {
		t.Fatalf("coreFoldEntries() len = %d, want 6", len(entries))
	}

	seen := make(map[event.Type]bool)
	for idx, entry := range entries {
		if entry.types == nil {
			t.Fatalf("entry %d has nil types", idx)
		}
		if entry.fold == nil {
			t.Fatalf("entry %d has nil fold", idx)
		}
		for _, evtType := range entry.types() {
			seen[evtType] = true
		}
	}

	expected := append(
		append(append(append(append(
			campaign.FoldHandledTypes(),
			session.FoldHandledTypes()...),
			action.FoldHandledTypes()...),
			participant.FoldHandledTypes()...),
			character.FoldHandledTypes()...),
		invite.FoldHandledTypes()...)
	for _, evtType := range expected {
		if !seen[evtType] {
			t.Fatalf("core fold dispatch missing event type %s", evtType)
		}
	}
}

func TestCoreFoldEntries_MatchFolderDispatchedTypes(t *testing.T) {
	expected := make(map[event.Type]struct{})
	for _, entry := range coreFoldEntries() {
		for _, evtType := range entry.types() {
			if _, dup := expected[evtType]; dup {
				t.Fatalf("coreFoldEntries() has duplicate event type %s", evtType)
			}
			expected[evtType] = struct{}{}
		}
	}

	folder := &Folder{}
	dispatchedTypes := folder.FoldDispatchedTypes()
	if len(dispatchedTypes) != len(expected) {
		t.Fatalf("FoldDispatchedTypes() len = %d, want %d", len(dispatchedTypes), len(expected))
	}
	seen := make(map[event.Type]struct{}, len(dispatchedTypes))
	for _, evtType := range dispatchedTypes {
		if _, dup := seen[evtType]; dup {
			t.Fatalf("FoldDispatchedTypes() contains duplicate event type %s", evtType)
		}
		seen[evtType] = struct{}{}
		if _, ok := expected[evtType]; !ok {
			t.Fatalf("FoldDispatchedTypes() contains unexpected event type %s", evtType)
		}
	}
	for evtType := range expected {
		if _, ok := seen[evtType]; !ok {
			t.Fatalf("FoldDispatchedTypes() missing expected event type %s", evtType)
		}
	}
}

func TestCoreFoldEntries_FoldFunctionsApplyRepresentativeEvents(t *testing.T) {
	state := &State{}
	for _, entry := range coreFoldEntries() {
		types := entry.types()
		if len(types) == 0 {
			t.Fatal("core fold entry has empty type list")
		}
		evt := representativeCoreFoldEvent(types[0])
		if err := entry.fold(state, evt); err != nil {
			t.Fatalf("entry fold for %s returned error: %v", types[0], err)
		}
	}
	if len(state.Participants) != 1 {
		t.Fatalf("participants map len = %d, want 1", len(state.Participants))
	}
	if len(state.Characters) != 1 {
		t.Fatalf("characters map len = %d, want 1", len(state.Characters))
	}
	if len(state.Invites) != 1 {
		t.Fatalf("invites map len = %d, want 1", len(state.Invites))
	}
}

func TestCoreFoldEntries_FoldFunctionsPropagateCorruptPayloadErrors(t *testing.T) {
	state := &State{}
	for _, entry := range coreFoldEntries() {
		types := entry.types()
		if len(types) == 0 {
			t.Fatal("core fold entry has empty type list")
		}
		evt := representativeCoreFoldEvent(types[0])
		evt.PayloadJSON = []byte(`{`)
		if err := entry.fold(state, evt); err == nil {
			t.Fatalf("entry fold for %s expected error for corrupt payload", types[0])
		}
	}
}

func representativeCoreFoldEvent(evtType event.Type) event.Event {
	switch evtType {
	case campaign.EventTypeCreated:
		return event.Event{Type: evtType, EntityID: "camp-1", PayloadJSON: []byte(`{}`)}
	case session.EventTypeStarted:
		return event.Event{Type: evtType, EntityID: "sess-1", PayloadJSON: []byte(`{}`)}
	case action.EventTypeRollResolved:
		return event.Event{Type: evtType, EntityID: "act-1", PayloadJSON: []byte(`{"roll_seq":1}`)}
	case participant.EventTypeJoined:
		return event.Event{Type: evtType, EntityID: "part-1", PayloadJSON: []byte(`{}`)}
	case character.EventTypeCreated:
		return event.Event{Type: evtType, EntityID: "char-1", PayloadJSON: []byte(`{}`)}
	case invite.EventTypeCreated:
		return event.Event{Type: evtType, EntityID: "inv-1", PayloadJSON: []byte(`{}`)}
	default:
		return event.Event{Type: evtType, EntityID: "entity-1", PayloadJSON: []byte(`{}`)}
	}
}
