package domainwrite

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestNormalizeOptions_CreatesDefaultShouldApplyResolver(t *testing.T) {
	called := 0
	resolver := func(_ event.Type) (event.Intent, bool) {
		called++
		return event.IntentProjectionAndReplay, true
	}

	options := normalizeOptions(Options{
		DefaultEventResolver: resolver,
	})
	if options.ShouldApply == nil {
		t.Fatal("expected default ShouldApply to be injected")
	}
	if !options.ShouldApply(event.Event{Type: event.Type("action.roll_resolved")}) {
		t.Fatal("expected resolver-driven ShouldApply decision to be true")
	}
	if called != 1 {
		t.Fatalf("expected resolver called once, got %d", called)
	}
}

func TestShouldApplyProjectionInline_FailsClosedWhenIntentIndexMissing(t *testing.T) {
	resolver := func(_ event.Type) (event.Intent, bool) {
		return "", false
	}

	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{name: "story note remains skipped", eventType: event.Type("story.note_added"), want: false},
		{name: "outcome rejection remains skipped", eventType: event.Type("action.outcome_rejected"), want: false},
		{name: "unknown event now skips", eventType: event.Type("custom.unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldApplyProjectionInlineWithResolver(resolver, event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("ShouldApplyProjectionInline(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestShouldApplyProjectionInline_UsesEventIntent(t *testing.T) {
	resolver := buildTestEventIntentResolver()

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
		{name: "unknown event defaults to skip", eventType: event.Type("custom.unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldApplyProjectionInlineWithResolver(resolver, event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("ShouldApplyProjectionInline(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestShouldApplyProjectionInline_FailsClosedWhenIntentUnknown(t *testing.T) {
	resolver := func(_ event.Type) (event.Intent, bool) {
		return "", false
	}

	if ShouldApplyProjectionInlineWithResolver(resolver, event.Event{Type: event.Type("custom.unknown")}) {
		t.Fatal("expected unknown event intent to skip inline apply")
	}
}

func TestShouldApplyProjectionInline_RetriesBootstrapAfterCachedFailure(t *testing.T) {
	resolver := func(eventType event.Type) (event.Intent, bool) {
		if eventType == event.Type("action.roll_resolved") {
			return event.IntentProjectionAndReplay, true
		}
		return "", false
	}

	evt := event.Event{Type: event.Type("action.roll_resolved")}
	if !ShouldApplyProjectionInlineWithResolver(resolver, evt) {
		t.Fatal("expected resolver decision to be deterministic")
	}
	if !ShouldApplyProjectionInlineWithResolver(resolver, evt) {
		t.Fatal("expected resolver decision to remain deterministic across calls")
	}
}

func buildTestEventIntentResolver() EventIntentResolver {
	intentIndex := map[event.Type]event.Intent{
		"story.note_added":                event.IntentAuditOnly,
		"action.outcome_rejected":         event.IntentAuditOnly,
		"action.roll_resolved":            event.IntentProjectionAndReplay,
		"action.outcome_applied":          event.IntentProjectionAndReplay,
		"sys.daggerheart.gm_fear_changed": event.IntentProjectionAndReplay,
	}

	return func(eventType event.Type) (event.Intent, bool) {
		intent, ok := intentIndex[eventType]
		return intent, ok
	}
}
