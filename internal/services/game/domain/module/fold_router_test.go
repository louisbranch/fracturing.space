package module

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type foldRouterState struct {
	Counter int
	Name    string
}

type foldCounterPayload struct {
	Delta int `json:"delta"`
}

type foldNamePayload struct {
	Name string `json:"name"`
}

func TestFoldRouter_DispatchesByEventType(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			s := &foldRouterState{}
			return s, nil
		}
		s, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return s, nil
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		s.Counter += p.Delta
		return nil
	})
	HandleFold(router, event.Type("test.name"), func(s *foldRouterState, p foldNamePayload) error {
		s.Name = p.Name
		return nil
	})

	state, err := router.Fold(nil, event.Event{
		Type:        event.Type("test.counter"),
		PayloadJSON: []byte(`{"delta":5}`),
	})
	if err != nil {
		t.Fatalf("fold counter: %v", err)
	}
	typed := state.(*foldRouterState)
	if typed.Counter != 5 {
		t.Fatalf("counter = %d, want 5", typed.Counter)
	}

	state, err = router.Fold(state, event.Event{
		Type:        event.Type("test.name"),
		PayloadJSON: []byte(`{"name":"alice"}`),
	})
	if err != nil {
		t.Fatalf("fold name: %v", err)
	}
	typed = state.(*foldRouterState)
	if typed.Name != "alice" {
		t.Fatalf("name = %s, want alice", typed.Name)
	}
	if typed.Counter != 5 {
		t.Fatalf("counter = %d after name fold, want 5", typed.Counter)
	}
}

func TestFoldRouter_ReturnsErrorOnUnknownEventType(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			return &foldRouterState{}, nil
		}
		s, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return s, nil
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		s.Counter += p.Delta
		return nil
	})

	_, err := router.Fold(nil, event.Event{
		Type:        event.Type("test.unknown"),
		PayloadJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type, got nil")
	}
}

func TestFoldRouter_ReturnsErrorOnUnmarshalFailure(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			return &foldRouterState{}, nil
		}
		s, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return s, nil
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		s.Counter += p.Delta
		return nil
	})

	_, err := router.Fold(nil, event.Event{
		Type:        event.Type("test.counter"),
		PayloadJSON: []byte(`{bad json`),
	})
	if err == nil {
		t.Fatal("expected error for bad payload JSON, got nil")
	}
}

func TestFoldRouter_ReturnsErrorOnAssertFailure(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			return &foldRouterState{}, nil
		}
		_, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return nil, errors.New("type mismatch")
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		s.Counter += p.Delta
		return nil
	})

	_, err := router.Fold("not-a-state", event.Event{
		Type:        event.Type("test.counter"),
		PayloadJSON: []byte(`{"delta":1}`),
	})
	if err == nil {
		t.Fatal("expected error for state assertion failure, got nil")
	}
}

func TestFoldRouter_HandlerErrorPropagates(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			return &foldRouterState{}, nil
		}
		s, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return s, nil
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		return errors.New("handler error")
	})

	_, err := router.Fold(nil, event.Event{
		Type:        event.Type("test.counter"),
		PayloadJSON: []byte(`{"delta":1}`),
	})
	if err == nil {
		t.Fatal("expected handler error, got nil")
	}
	if err.Error() != "handler error" {
		t.Fatalf("error = %v, want handler error", err)
	}
}

func TestFoldRouter_FoldHandledTypes(t *testing.T) {
	router := NewFoldRouter(func(v any) (*foldRouterState, error) {
		if v == nil {
			return &foldRouterState{}, nil
		}
		s, ok := v.(*foldRouterState)
		if !ok {
			return nil, errors.New("type mismatch")
		}
		return s, nil
	})
	HandleFold(router, event.Type("test.counter"), func(s *foldRouterState, p foldCounterPayload) error {
		s.Counter += p.Delta
		return nil
	})
	HandleFold(router, event.Type("test.name"), func(s *foldRouterState, p foldNamePayload) error {
		s.Name = p.Name
		return nil
	})

	types := router.FoldHandledTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 handled types, got %d", len(types))
	}
	typeSet := make(map[event.Type]bool)
	for _, et := range types {
		typeSet[et] = true
	}
	if !typeSet[event.Type("test.counter")] {
		t.Fatal("missing test.counter in handled types")
	}
	if !typeSet[event.Type("test.name")] {
		t.Fatal("missing test.name in handled types")
	}
}
