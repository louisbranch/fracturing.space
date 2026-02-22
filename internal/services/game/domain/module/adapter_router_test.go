package module

import (
	"context"
	"fmt"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type testAdapterPayload struct {
	Value string `json:"value"`
}

func TestAdapterRouter_Dispatch(t *testing.T) {
	router := NewAdapterRouter()
	var captured testAdapterPayload
	HandleAdapter(router, event.Type("test.event"), func(ctx context.Context, evt event.Event, p testAdapterPayload) error {
		captured = p
		return nil
	})

	err := router.Apply(context.Background(), event.Event{
		Type:        event.Type("test.event"),
		PayloadJSON: []byte(`{"value":"hello"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Value != "hello" {
		t.Fatalf("value = %q, want %q", captured.Value, "hello")
	}
}

func TestAdapterRouter_UnknownEventType(t *testing.T) {
	router := NewAdapterRouter()
	err := router.Apply(context.Background(), event.Event{
		Type:        event.Type("unknown.event"),
		PayloadJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestAdapterRouter_UnmarshalError(t *testing.T) {
	router := NewAdapterRouter()
	HandleAdapter(router, event.Type("test.event"), func(_ context.Context, _ event.Event, _ testAdapterPayload) error {
		t.Fatal("handler should not be called on unmarshal error")
		return nil
	})

	err := router.Apply(context.Background(), event.Event{
		Type:        event.Type("test.event"),
		PayloadJSON: []byte(`{bad json`),
	})
	if err == nil {
		t.Fatal("expected error for bad payload JSON")
	}
}

func TestAdapterRouter_HandlerErrorPropagates(t *testing.T) {
	router := NewAdapterRouter()
	HandleAdapter(router, event.Type("test.event"), func(_ context.Context, _ event.Event, _ testAdapterPayload) error {
		return fmt.Errorf("handler failed")
	})

	err := router.Apply(context.Background(), event.Event{
		Type:        event.Type("test.event"),
		PayloadJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if err.Error() != "handler failed" {
		t.Fatalf("error = %q, want %q", err.Error(), "handler failed")
	}
}

func TestAdapterRouter_HandledTypes(t *testing.T) {
	router := NewAdapterRouter()
	HandleAdapter(router, event.Type("b.event"), func(_ context.Context, _ event.Event, _ testAdapterPayload) error {
		return nil
	})
	HandleAdapter(router, event.Type("a.event"), func(_ context.Context, _ event.Event, _ testAdapterPayload) error {
		return nil
	})

	types := router.HandledTypes()
	if len(types) != 2 {
		t.Fatalf("len = %d, want 2", len(types))
	}
	// Registration order preserved.
	if types[0] != event.Type("b.event") {
		t.Fatalf("types[0] = %s, want b.event", types[0])
	}
	if types[1] != event.Type("a.event") {
		t.Fatalf("types[1] = %s, want a.event", types[1])
	}
}
