package domainwrite

import (
	"sync"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestShouldApplyProjectionInline_FallsBackToKnownAuditOnlyWhenIntentIndexMissing(t *testing.T) {
	inlineApplyIntentIndex = nil
	inlineApplyIntentIndexOnce = sync.Once{}
	inlineApplyIntentIndexOnce.Do(func() {
		inlineApplyIntentIndex = map[event.Type]event.Intent{}
	})
	t.Cleanup(func() {
		inlineApplyIntentIndex = nil
		inlineApplyIntentIndexOnce = sync.Once{}
	})

	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{name: "story note remains skipped", eventType: event.Type("story.note_added"), want: false},
		{name: "outcome rejection remains skipped", eventType: event.Type("action.outcome_rejected"), want: false},
		{name: "unknown event still applies", eventType: event.Type("custom.unknown"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldApplyProjectionInline(event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("ShouldApplyProjectionInline(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestShouldApplyProjectionInline_UsesEventIntent(t *testing.T) {
	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{name: "audit note event", eventType: event.Type("story.note_added"), want: false},
		{name: "audit outcome rejected event", eventType: event.Type("action.outcome_rejected"), want: false},
		{name: "projection roll event", eventType: event.Type("action.roll_resolved"), want: true},
		{name: "projection outcome applied event", eventType: event.Type("action.outcome_applied"), want: true},
		{name: "projection system event", eventType: event.Type("sys.daggerheart.gm_fear_changed"), want: true},
		{name: "unknown event defaults to apply", eventType: event.Type("custom.unknown"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldApplyProjectionInline(event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("ShouldApplyProjectionInline(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}
