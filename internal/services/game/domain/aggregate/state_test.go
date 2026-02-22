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
	if err != nil {
		t.Fatalf("unexpected error for nil pointer: %v", err)
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
	if err != nil {
		t.Fatalf("unexpected error for nil: %v", err)
	}
}
