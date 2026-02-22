package module

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type testState struct {
	Count int
}

// TypedFolder tests

func TestTypedFolder_SatisfiesFolderInterface(t *testing.T) {
	var _ Folder = TypedFolder[testState]{}
}

func TestTypedFolder_Apply_DelegatesToFold(t *testing.T) {
	p := TypedFolder[testState]{
		Assert: func(state any) (testState, error) {
			if state == nil {
				return testState{}, nil
			}
			s, ok := state.(testState)
			if !ok {
				return testState{}, errors.New("bad state type")
			}
			return s, nil
		},
		Fold: func(s testState, evt event.Event) (testState, error) {
			s.Count++
			return s, nil
		},
		Types: func() []event.Type {
			return []event.Type{"test.event"}
		},
	}

	result, err := p.Apply(nil, event.Event{Type: "test.event"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := result.(testState)
	if !ok {
		t.Fatalf("result is %T, want testState", result)
	}
	if s.Count != 1 {
		t.Fatalf("count = %d, want 1", s.Count)
	}
}

func TestTypedFolder_Apply_PropagatesAssertError(t *testing.T) {
	p := TypedFolder[testState]{
		Assert: func(state any) (testState, error) {
			return testState{}, errors.New("bad state")
		},
		Fold: func(s testState, evt event.Event) (testState, error) {
			t.Fatal("fold should not be called on assert error")
			return s, nil
		},
		Types: func() []event.Type { return nil },
	}

	_, err := p.Apply("wrong type", event.Event{})
	if err == nil {
		t.Fatal("expected error from assert")
	}
}

func TestTypedFolder_Apply_PropagatesFoldError(t *testing.T) {
	p := TypedFolder[testState]{
		Assert: func(state any) (testState, error) {
			return testState{}, nil
		},
		Fold: func(s testState, evt event.Event) (testState, error) {
			return s, errors.New("fold failed")
		},
		Types: func() []event.Type { return nil },
	}

	_, err := p.Apply(nil, event.Event{})
	if err == nil {
		t.Fatal("expected error from fold")
	}
}

func TestTypedFolder_FoldHandledTypes(t *testing.T) {
	expected := []event.Type{"a.done", "b.done"}
	p := TypedFolder[testState]{
		Types: func() []event.Type { return expected },
	}
	got := p.FoldHandledTypes()
	if len(got) != len(expected) {
		t.Fatalf("FoldHandledTypes len = %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("FoldHandledTypes[%d] = %s, want %s", i, got[i], expected[i])
		}
	}
}

func TestTypedFolder_Apply_NilAssertReturnsError(t *testing.T) {
	p := TypedFolder[testState]{
		Fold:  func(s testState, _ event.Event) (testState, error) { return s, nil },
		Types: func() []event.Type { return nil },
	}
	_, err := p.Apply(nil, event.Event{})
	if err == nil {
		t.Fatal("expected error for nil Assert")
	}
}

func TestTypedFolder_Apply_NilFoldReturnsError(t *testing.T) {
	p := TypedFolder[testState]{
		Assert: func(any) (testState, error) { return testState{}, nil },
		Types:  func() []event.Type { return nil },
	}
	_, err := p.Apply(nil, event.Event{})
	if err == nil {
		t.Fatal("expected error for nil Fold")
	}
}

func TestTypedFolder_FoldHandledTypes_NilTypesReturnsNil(t *testing.T) {
	p := TypedFolder[testState]{}
	got := p.FoldHandledTypes()
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// TypedDecider tests

func TestTypedDecider_SatisfiesDeciderInterface(t *testing.T) {
	var _ Decider = TypedDecider[testState]{}
}

func TestTypedDecider_Decide_DelegatesToFn(t *testing.T) {
	d := TypedDecider[testState]{
		Assert: func(state any) (testState, error) {
			if state == nil {
				return testState{}, nil
			}
			return state.(testState), nil
		},
		Fn: func(s testState, cmd command.Command, now func() time.Time) command.Decision {
			return command.Reject(command.Rejection{Code: "TEST", Message: "test"})
		},
	}

	decision := d.Decide(nil, command.Command{}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "TEST" {
		t.Fatalf("rejection code = %s, want TEST", decision.Rejections[0].Code)
	}
}

func TestTypedDecider_Decide_PropagatesAssertError(t *testing.T) {
	d := TypedDecider[testState]{
		Assert: func(state any) (testState, error) {
			return testState{}, errors.New("bad state")
		},
		Fn: func(s testState, cmd command.Command, now func() time.Time) command.Decision {
			t.Fatal("fn should not be called on assert error")
			return command.Decision{}
		},
	}

	decision := d.Decide("wrong", command.Command{}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "STATE_ASSERT_FAILED" {
		t.Fatalf("rejection code = %s, want STATE_ASSERT_FAILED", decision.Rejections[0].Code)
	}
}

func TestTypedDecider_Decide_NilAssertRejects(t *testing.T) {
	d := TypedDecider[testState]{
		Fn: func(_ testState, _ command.Command, _ func() time.Time) command.Decision {
			return command.Decision{}
		},
	}
	decision := d.Decide(nil, command.Command{}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
}

func TestTypedDecider_Decide_NilFnRejects(t *testing.T) {
	d := TypedDecider[testState]{
		Assert: func(any) (testState, error) { return testState{}, nil },
	}
	decision := d.Decide(nil, command.Command{}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
}
