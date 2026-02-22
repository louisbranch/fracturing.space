package domainwrite

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestNewIntentFilter_SkipsAuditOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("action.roll_resolved"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	if err := registry.Register(event.Definition{
		Type:   event.Type("story.note_added"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	filter := NewIntentFilter(registry)

	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{"projection event applies", event.Type("action.roll_resolved"), true},
		{"audit-only event skipped", event.Type("story.note_added"), false},
		{"unknown event skipped (fail closed)", event.Type("custom.unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter(event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("filter(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestNewIntentFilter_SkipsReplayOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("action.roll_resolved"),
		Owner:  event.OwnerCore,
		Intent: event.IntentReplayOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	filter := NewIntentFilter(registry)
	if filter(event.Event{Type: event.Type("action.roll_resolved")}) {
		t.Fatal("expected replay-only event to be filtered out")
	}
}

func TestNewIntentFilter_NilRegistryFailsClosed(t *testing.T) {
	filter := NewIntentFilter(nil)

	if filter(event.Event{Type: event.Type("action.roll_resolved")}) {
		t.Fatal("expected nil registry filter to fail closed")
	}
}
