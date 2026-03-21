package fold

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type testState struct {
	Counter int
	Name    string
}

func TestCoreFoldRouter_DispatchesByEventType(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.increment"), func(s testState, evt event.Event) (testState, error) {
		s.Counter++
		return s, nil
	})
	r.Handle(event.Type("test.name"), func(s testState, evt event.Event) (testState, error) {
		s.Name = "updated"
		return s, nil
	})

	state, err := r.Fold(testState{}, event.Event{
		Type:        event.Type("test.increment"),
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if state.Counter != 1 {
		t.Fatalf("counter = %d, want 1", state.Counter)
	}

	state, err = r.Fold(state, event.Event{
		Type:        event.Type("test.name"),
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("fold name: %v", err)
	}
	if state.Name != "updated" {
		t.Fatalf("name = %s, want updated", state.Name)
	}
	if state.Counter != 1 {
		t.Fatalf("counter = %d after name fold, want 1", state.Counter)
	}
}

func TestCoreFoldRouter_ReturnsErrorOnUnknownEventType(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.known"), func(s testState, _ event.Event) (testState, error) {
		return s, nil
	})

	_, err := r.Fold(testState{}, event.Event{
		Type:        event.Type("test.unknown"),
		PayloadJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type, got nil")
	}
}

func TestCoreFoldRouter_FoldHandledTypes(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.a"), func(s testState, _ event.Event) (testState, error) { return s, nil })
	r.Handle(event.Type("test.b"), func(s testState, _ event.Event) (testState, error) { return s, nil })

	types := r.FoldHandledTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 handled types, got %d", len(types))
	}
	// Verify registration order is preserved.
	if types[0] != "test.a" {
		t.Fatalf("types[0] = %s, want test.a", types[0])
	}
	if types[1] != "test.b" {
		t.Fatalf("types[1] = %s, want test.b", types[1])
	}
}

func TestCoreFoldRouter_FoldHandledTypesReturnsCopy(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.a"), func(s testState, _ event.Event) (testState, error) { return s, nil })

	types := r.FoldHandledTypes()
	types[0] = "mutated"

	fresh := r.FoldHandledTypes()
	if fresh[0] != "test.a" {
		t.Fatal("FoldHandledTypes returned a reference instead of a copy")
	}
}

func TestCoreFoldRouter_PanicsOnDuplicateRegistration(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.a"), func(s testState, _ event.Event) (testState, error) { return s, nil })

	defer func() {
		if rv := recover(); rv == nil {
			t.Fatal("expected panic for duplicate fold handler registration")
		}
	}()
	r.Handle(event.Type("test.a"), func(s testState, _ event.Event) (testState, error) { return s, nil })
}

// TestCoreFoldRouter_SyncDriftDetection verifies that FoldHandledTypes is
// always derived from registered handlers. This is the key property that
// eliminates the sync-drift risk between a domain's Fold switch and its
// FoldHandledTypes list.
func TestCoreFoldRouter_SyncDriftDetection(t *testing.T) {
	r := NewCoreFoldRouter[testState]()
	r.Handle(event.Type("test.a"), func(s testState, _ event.Event) (testState, error) { return s, nil })

	// FoldHandledTypes should report exactly one type.
	types := r.FoldHandledTypes()
	if len(types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(types))
	}

	// Register another handler.
	r.Handle(event.Type("test.b"), func(s testState, _ event.Event) (testState, error) { return s, nil })

	// FoldHandledTypes must now include both — impossible with a manual list
	// that forgets to update.
	types = r.FoldHandledTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 types after adding handler, got %d", len(types))
	}
}
