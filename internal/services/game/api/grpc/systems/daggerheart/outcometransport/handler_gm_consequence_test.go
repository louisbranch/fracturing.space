package outcometransport

import (
	"context"
	"testing"
)

func TestHandlerBuildGMConsequenceOutcomeEffectsAddsGateAndSpotlight(t *testing.T) {
	handler, _, _ := newTestHandler()

	effects, err := handler.buildGMConsequenceOutcomeEffects(context.Background(), "camp-1", "sess-1", 9, "req-1")
	if err != nil {
		t.Fatalf("buildGMConsequenceOutcomeEffects returned error: %v", err)
	}
	if got := len(effects); got != 2 {
		t.Fatalf("effects len = %d, want 2", got)
	}
	if got := effects[0].Type; got != "session.gate_opened" {
		t.Fatalf("first effect type = %q", got)
	}
	if got := effects[1].Type; got != "session.spotlight_set" {
		t.Fatalf("second effect type = %q", got)
	}
}

func TestHandlerOpenGMConsequenceGateExecutesRepairs(t *testing.T) {
	handler, _, recorder := newTestHandler()

	if err := handler.openGMConsequenceGate(testSessionContext("camp-1", "sess-1"), "camp-1", "sess-1", "scene-1", 9, "req-1"); err != nil {
		t.Fatalf("openGMConsequenceGate returned error: %v", err)
	}
	if got := len(recorder.coreCommands); got != 2 {
		t.Fatalf("core command count = %d, want 2", got)
	}
	if recorder.coreCommands[0].CommandType != commandTypeSessionGateOpen {
		t.Fatalf("first core command type = %q", recorder.coreCommands[0].CommandType)
	}
	if recorder.coreCommands[1].CommandType != commandTypeSessionSpotlightSet {
		t.Fatalf("second core command type = %q", recorder.coreCommands[1].CommandType)
	}
}
