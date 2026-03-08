package aggregate

import (
	"testing"
)

func TestAssertState_Value(t *testing.T) {
	state := State{}
	_, err := AssertState[State](state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertState_Pointer(t *testing.T) {
	state := &State{}
	_, err := AssertState[State](state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertState_NilPointer(t *testing.T) {
	var state *State
	_, err := AssertState[State](state)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestAssertState_WrongType(t *testing.T) {
	_, err := AssertState[State]("not a state")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	want := `expected aggregate.State, got string`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestAssertState_Nil(t *testing.T) {
	_, err := AssertState[State](nil)
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestNewState_MapsInitialized(t *testing.T) {
	s := NewState()
	if s.Participants == nil {
		t.Fatal("Participants map is nil")
	}
	if s.Characters == nil {
		t.Fatal("Characters map is nil")
	}
	if s.Invites == nil {
		t.Fatal("Invites map is nil")
	}
	if s.Scenes == nil {
		t.Fatal("Scenes map is nil")
	}
	if s.Systems == nil {
		t.Fatal("Systems map is nil")
	}
}
