package gametools

import (
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

func TestGMInteractionInputFromToolIgnoresIllustrationWithoutImageURL(t *testing.T) {
	input, err := gmInteractionInputFromTool(&interactionGMInteractionInput{
		Title: "Dawn at the Docks",
		Illustration: &interactionGMInteractionIllustrationInput{
			Alt:     "Unused alt text",
			Caption: "Unused caption",
		},
		Beats: []interactionGMInteractionBeatInput{{
			Type: "fiction",
			Text: "The tide creeps toward the pilings.",
		}},
	}, nil)
	if err != nil {
		t.Fatalf("gmInteractionInputFromTool() error = %v", err)
	}
	if input.Illustration != nil {
		t.Fatalf("illustration = %#v, want nil", input.Illustration)
	}
}

func TestGMInteractionInputFromToolRequiresAltWhenImageURLPresent(t *testing.T) {
	_, err := gmInteractionInputFromTool(&interactionGMInteractionInput{
		Title: "Dawn at the Docks",
		Illustration: &interactionGMInteractionIllustrationInput{
			ImageURL: "https://cdn.example.com/harbor.png",
		},
		Beats: []interactionGMInteractionBeatInput{{
			Type: "fiction",
			Text: "The tide creeps toward the pilings.",
		}},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), "interaction illustration alt is required when illustration is provided"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestGMInteractionInputFromToolMapsIllustrationWhenComplete(t *testing.T) {
	input, err := gmInteractionInputFromTool(&interactionGMInteractionInput{
		Title: "Dawn at the Docks",
		Illustration: &interactionGMInteractionIllustrationInput{
			ImageURL: "https://cdn.example.com/harbor.png",
			Alt:      "Lantern light on the harbor",
			Caption:  "Dawn at the docks.",
		},
		Beats: []interactionGMInteractionBeatInput{{
			Type: "fiction",
			Text: "The tide creeps toward the pilings.",
		}},
	}, nil)
	if err != nil {
		t.Fatalf("gmInteractionInputFromTool() error = %v", err)
	}
	if input.Illustration == nil {
		t.Fatal("illustration = nil, want populated illustration")
	}
	if got, want := input.Illustration.GetImageUrl(), "https://cdn.example.com/harbor.png"; got != want {
		t.Fatalf("image_url = %q, want %q", got, want)
	}
	if got, want := input.Illustration.GetAlt(), "Lantern light on the harbor"; got != want {
		t.Fatalf("alt = %q, want %q", got, want)
	}
	if got, want := input.Illustration.GetCaption(), "Dawn at the docks."; got != want {
		t.Fatalf("caption = %q, want %q", got, want)
	}
}

func TestInteractionStateFromProtoMarksPlayerPhaseAsReadyForCompletion(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
		PlayerPhase: &statev1.ScenePlayerPhase{
			PhaseId:              "phase-1",
			Status:               statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS,
			ActingParticipantIds: []string{"player-1"},
		},
		Ooc: &statev1.OOCState{},
	})

	if !result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want true", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "Players may act next") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}

func TestInteractionStateFromProtoMarksGMReviewAsBlocked(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
		PlayerPhase: &statev1.ScenePlayerPhase{
			PhaseId: "phase-1",
			Status:  statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW,
		},
		Ooc: &statev1.OOCState{},
	})

	if result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want false", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "interaction_resolve_scene_player_review") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}

func TestInteractionStateFromProtoMarksGMControlAsBlocked(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
		PlayerPhase: &statev1.ScenePlayerPhase{
			Status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM,
		},
		Ooc: &statev1.OOCState{},
	})

	if result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want false", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "open the next player phase") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}

func TestInteractionStateFromProtoMarksOOCResolutionPendingAsBlocked(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
		PlayerPhase: &statev1.ScenePlayerPhase{
			PhaseId:              "phase-1",
			Status:               statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS,
			ActingParticipantIds: []string{"player-1"},
		},
		Ooc: &statev1.OOCState{ResolutionPending: true},
	})

	if result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want false", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "interaction_session_ooc_resolve") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}

func TestInteractionStateFromProtoUsesExplicitOOCToolNamesWhenPaused(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
		Ooc:         &statev1.OOCState{Open: true},
	})

	if !result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want true", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "interaction_open_session_ooc") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
	if !strings.Contains(result.NextStepHint, "interaction_session_ooc_resolve") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}

func TestInteractionStateFromProtoUsesDefaultActiveSceneCreateGuidanceWhenMissingScene(t *testing.T) {
	result := interactionStateFromProto(&statev1.InteractionState{
		Ooc: &statev1.OOCState{},
	})

	if result.AITurnReadyForCompletion {
		t.Fatalf("ai_turn_ready_for_completion = %v, want false", result.AITurnReadyForCompletion)
	}
	if !strings.Contains(result.NextStepHint, "scene_create (which activates by default)") {
		t.Fatalf("next_step_hint = %q", result.NextStepHint)
	}
}
