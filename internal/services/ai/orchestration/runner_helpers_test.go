package orchestration

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildPlayerPhaseStartReminderIncludesDraftNarration(t *testing.T) {
	got := NewInteractionTurnPolicy().Controller(DefaultCommitToolName).BuildPlayerPhaseStartReminder("  Hold the pier.  ")
	if !strings.Contains(got, "interaction_open_scene_player_phase") {
		t.Fatalf("reminder missing player phase tool guidance: %q", got)
	}
	if !strings.Contains(got, "Hold the pier.") {
		t.Fatalf("reminder missing trimmed draft narration: %q", got)
	}
}

func TestToolHandsControlBackToPlayers(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want bool
	}{
		{name: "player phase start", tool: playerPhaseStartToolName, want: true},
		{name: "review resolve", tool: reviewResolveToolName, want: true},
		{name: "interrupt resolve", tool: interruptResolutionToolName, want: true},
		{name: "commit only", tool: "interaction_record_scene_gm_interaction", want: false},
		{name: "blank", tool: "  ", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := toolHandsControlBackToPlayers(tc.tool); got != tc.want {
				t.Fatalf("toolHandsControlBackToPlayers(%q) = %v, want %v", tc.tool, got, tc.want)
			}
		})
	}
}

func TestDecodeArgs(t *testing.T) {
	t.Run("empty returns empty object", func(t *testing.T) {
		got, err := decodeArgs("")
		if err != nil {
			t.Fatalf("decodeArgs() error = %v", err)
		}
		if !reflect.DeepEqual(got, map[string]any{}) {
			t.Fatalf("decodeArgs() = %#v, want empty object", got)
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		if _, err := decodeArgs("{"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("json object returns decoded value", func(t *testing.T) {
		got, err := decodeArgs(`{"scene_id":"scene-1","count":2}`)
		if err != nil {
			t.Fatalf("decodeArgs() error = %v", err)
		}
		want := map[string]any{
			"scene_id": "scene-1",
			"count":    float64(2),
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeArgs() = %#v, want %#v", got, want)
		}
	})
}

func TestToolResultControlState(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		ready, handoff, ok := toolResultControlState("{")
		if ready || handoff || ok {
			t.Fatalf("toolResultControlState() = (%v, %v, %v), want all false", ready, handoff, ok)
		}
	})

	t.Run("explicit ready flag wins", func(t *testing.T) {
		ready, handoff, ok := toolResultControlState(`{"ai_turn_ready_for_completion":false,"player_phase":{"phase_id":"phase-1","status":"players","acting_participant_ids":["p-1"]}}`)
		if !ok {
			t.Fatal("expected parse success")
		}
		if ready {
			t.Fatal("explicit ready flag should keep ready false")
		}
		if !handoff {
			t.Fatal("expected player handoff to remain true")
		}
	})

	t.Run("ooc ready without player handoff", func(t *testing.T) {
		ready, handoff, ok := toolResultControlState(`{"ooc":{"open":true,"resolution_pending":false}}`)
		if !ok {
			t.Fatal("expected parse success")
		}
		if !ready || handoff {
			t.Fatalf("toolResultControlState() = (%v, %v, %v), want ready true handoff false ok true", ready, handoff, ok)
		}
	})

	t.Run("players infer ready and handoff", func(t *testing.T) {
		ready, handoff, ok := toolResultControlState(`{"player_phase":{"phase_id":"phase-1","status":"players","acting_participant_ids":["p-1"]}}`)
		if !ok {
			t.Fatal("expected parse success")
		}
		if !ready || !handoff {
			t.Fatalf("toolResultControlState() = (%v, %v, %v), want ready true handoff true ok true", ready, handoff, ok)
		}
	})
}

func TestStaticToolPolicyAndFilterTools(t *testing.T) {
	policy := NewStaticToolPolicy([]string{" scene_create ", "", playerPhaseStartToolName})
	if !policy.Allows("scene_create") {
		t.Fatal("expected trimmed allowlist name to be allowed")
	}
	if policy.Allows(" ") {
		t.Fatal("blank tool name should not be allowed")
	}
	if policy.Allows("campaign_create") {
		t.Fatal("unexpected tool allowed")
	}

	filtered := filterTools([]Tool{
		{Name: " scene_create "},
		{Name: ""},
		{Name: "campaign_create"},
		{Name: playerPhaseStartToolName},
	}, policy)
	want := []Tool{
		{Name: "scene_create"},
		{Name: playerPhaseStartToolName},
	}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("filterTools() = %#v, want %#v", filtered, want)
	}
}
