package aggregate

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldEntityKeyed_ValidEntityID(t *testing.T) {
	m := map[string]int{"e1": 10}
	evt := event.Event{EntityID: "e1", Type: "test.event"}
	err := foldEntityKeyed(&m, evt, "test", func(s int, _ event.Event) (int, error) {
		return s + 1, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["e1"] != 11 {
		t.Fatalf("expected 11, got %d", m["e1"])
	}
}

func TestFoldEntityKeyed_EmptyEntityID(t *testing.T) {
	m := map[string]int{}
	evt := event.Event{EntityID: "", Type: "test.event"}
	err := foldEntityKeyed(&m, evt, "test", func(s int, _ event.Event) (int, error) {
		return s, nil
	})
	if err == nil {
		t.Fatal("expected error for empty EntityID")
	}
}

func TestFoldEntityKeyed_NilMapLazyInit(t *testing.T) {
	var m map[string]int
	evt := event.Event{EntityID: "e1", Type: "test.event"}
	err := foldEntityKeyed(&m, evt, "test", func(s int, _ event.Event) (int, error) {
		return s + 5, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected map to be initialized")
	}
	if m["e1"] != 5 {
		t.Fatalf("expected 5, got %d", m["e1"])
	}
}
