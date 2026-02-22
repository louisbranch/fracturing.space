package projection

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// testPayload is a sample payload for core router tests.
type testPayload struct {
	Name string `json:"name"`
}

func TestCoreRouter_HandleProjection_AutoUnmarshals(t *testing.T) {
	router := NewCoreRouter()

	var got testPayload
	HandleProjection(router, "test.created", 0, 0,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error {
			got = p
			return nil
		})

	evt := event.Event{
		Type:        "test.created",
		PayloadJSON: []byte(`{"name":"alice"}`),
	}
	if err := router.Route(Applier{}, context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("expected payload name %q, got %q", "alice", got.Name)
	}
}

func TestCoreRouter_Route_RejectsUnknownType(t *testing.T) {
	router := NewCoreRouter()

	evt := event.Event{Type: "unknown.type"}
	err := router.Route(Applier{}, context.Background(), evt)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestCoreRouter_Route_ChecksStorePreconditions(t *testing.T) {
	router := NewCoreRouter()

	HandleProjection(router, "test.created", needCampaign, 0,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error {
			return nil
		})

	// Applier with nil Campaign store should fail precondition check.
	evt := event.Event{
		Type:        "test.created",
		PayloadJSON: []byte(`{}`),
	}
	err := router.Route(Applier{}, context.Background(), evt)
	if err == nil {
		t.Fatal("expected precondition error for missing campaign store")
	}
}

func TestCoreRouter_Route_ChecksIDPreconditions(t *testing.T) {
	router := NewCoreRouter()

	HandleProjection(router, "test.created", 0, requireCampaignID,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error {
			return nil
		})

	// Event without CampaignID should fail.
	evt := event.Event{
		Type:        "test.created",
		PayloadJSON: []byte(`{}`),
	}
	err := router.Route(Applier{}, context.Background(), evt)
	if err == nil {
		t.Fatal("expected precondition error for missing campaign ID")
	}
}

func TestCoreRouter_HandledTypes_ReturnsRegistrationOrder(t *testing.T) {
	router := NewCoreRouter()

	HandleProjection(router, "b.event", 0, 0,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error { return nil })
	HandleProjection(router, "a.event", 0, 0,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error { return nil })

	types := router.HandledTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
	if types[0] != "b.event" || types[1] != "a.event" {
		t.Fatalf("expected [b.event, a.event], got %v", types)
	}
}

func TestCoreRouter_HandleProjectionRaw_NoPayloadUnmarshal(t *testing.T) {
	router := NewCoreRouter()

	called := false
	HandleProjectionRaw(router, "test.cleared", 0, 0,
		func(a Applier, ctx context.Context, evt event.Event) error {
			called = true
			return nil
		})

	evt := event.Event{Type: "test.cleared"}
	if err := router.Route(Applier{}, context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestCoreRouter_Route_RejectsInvalidPayload(t *testing.T) {
	router := NewCoreRouter()

	HandleProjection(router, "test.created", 0, 0,
		func(a Applier, ctx context.Context, evt event.Event, p testPayload) error {
			return nil
		})

	evt := event.Event{
		Type:        "test.created",
		PayloadJSON: []byte(`not-json`),
	}
	err := router.Route(Applier{}, context.Background(), evt)
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}
