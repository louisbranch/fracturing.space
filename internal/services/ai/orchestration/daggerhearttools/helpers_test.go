package daggerhearttools

import (
	"encoding/json"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func decodeDaggerheartToolOutput[T any](t *testing.T, result string) T {
	t.Helper()

	var value T
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	return value
}

func TestToolResultJSONMarshalsPayload(t *testing.T) {
	result, err := toolResultJSON(struct {
		Kind string `json:"kind"`
	}{Kind: "combat_board"})
	if err != nil {
		t.Fatalf("toolResultJSON() error = %v", err)
	}

	payload := decodeDaggerheartToolOutput[struct {
		Kind string `json:"kind"`
	}](t, result.Output)
	if payload.Kind != "combat_board" {
		t.Fatalf("kind = %q, want combat_board", payload.Kind)
	}
}

func TestRollModeHelpersRoundTripRecognizedModes(t *testing.T) {
	if got := rollModeToProto(" replay "); got != commonv1.RollMode_REPLAY {
		t.Fatalf("rollModeToProto(replay) = %v", got)
	}
	if got := rollModeToProto("LIVE"); got != commonv1.RollMode_LIVE {
		t.Fatalf("rollModeToProto(LIVE) = %v", got)
	}
	if got := rollModeToProto("unknown"); got != commonv1.RollMode_ROLL_MODE_UNSPECIFIED {
		t.Fatalf("rollModeToProto(unknown) = %v", got)
	}
	if got := rollModeLabel(commonv1.RollMode_REPLAY); got != "REPLAY" {
		t.Fatalf("rollModeLabel(REPLAY) = %q", got)
	}
	if got := rollModeLabel(commonv1.RollMode_ROLL_MODE_UNSPECIFIED); got != "" {
		t.Fatalf("rollModeLabel(UNSPECIFIED) = %q, want empty", got)
	}
}

func TestRNGHelpersConvertSeedsAndResponses(t *testing.T) {
	seed := uint64(42)
	req := rngRequestToProto(&rngRequest{Seed: &seed, RollMode: "live"})
	if req.GetSeed() != 42 || req.GetRollMode() != commonv1.RollMode_LIVE {
		t.Fatalf("rngRequestToProto() = %#v", req)
	}
	if rngRequestToProto(nil) != nil {
		t.Fatal("rngRequestToProto(nil) = non-nil, want nil")
	}

	result := rngResultFromProto(&commonv1.RngResponse{
		SeedUsed:   42,
		RngAlgo:    "pcg",
		SeedSource: "explicit",
		RollMode:   commonv1.RollMode_REPLAY,
	})
	if result == nil || result.RollMode != "REPLAY" || result.RngAlgo != "pcg" {
		t.Fatalf("rngResultFromProto() = %#v", result)
	}
	if rngResultFromProto(nil) != nil {
		t.Fatal("rngResultFromProto(nil) = non-nil, want nil")
	}
}

func TestEnumConversionHelpersCoverRecognizedAndFallbackValues(t *testing.T) {
	if got := intSlice([]int32{1, -2, 3}); len(got) != 3 || got[1] != -2 {
		t.Fatalf("intSlice() = %#v", got)
	}
	if got := countdownToneToProto("progress"); got != pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS {
		t.Fatalf("countdownToneToProto(progress) = %v", got)
	}
	if got := countdownToneToString(pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE); got != "CONSEQUENCE" {
		t.Fatalf("countdownToneToString() = %q", got)
	}
	if got := countdownPolicyToProto("long_rest"); got != pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST {
		t.Fatalf("countdownPolicyToProto(long_rest) = %v", got)
	}
	if got := countdownPolicyToString(pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC); got != "ACTION_DYNAMIC" {
		t.Fatalf("countdownPolicyToString() = %q", got)
	}
	if got := countdownLoopBehaviorToProto("reset_decrease_start"); got != pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START {
		t.Fatalf("countdownLoopBehaviorToProto() = %v", got)
	}
	if got := countdownLoopBehaviorToString(pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE); got != "NONE" {
		t.Fatalf("countdownLoopBehaviorToString() = %q", got)
	}
	if got := countdownStatusToString(pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING); got != "TRIGGER_PENDING" {
		t.Fatalf("countdownStatusToString() = %q", got)
	}
	if got := actionRollContextToProto("move_silently"); got != pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY {
		t.Fatalf("actionRollContextToProto() = %v", got)
	}
	if got := daggerheartLifeStateToString(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY); got != "BLAZE_OF_GLORY" {
		t.Fatalf("daggerheartLifeStateToString() = %q", got)
	}
	if got := daggerheartAttackRangeToProto("ranged"); got != pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED {
		t.Fatalf("daggerheartAttackRangeToProto() = %v", got)
	}
	if got := daggerheartDamageTypeToProto("mixed"); got != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED {
		t.Fatalf("daggerheartDamageTypeToProto() = %v", got)
	}
	if got := gmMoveKindToProto("additional_move"); got != pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE {
		t.Fatalf("gmMoveKindToProto() = %v", got)
	}
	if got := gmMoveShapeToProto("custom"); got != pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM {
		t.Fatalf("gmMoveShapeToProto() = %v", got)
	}
}

func TestSessionActionOutcomeLabelMapsFlavorAndCrit(t *testing.T) {
	tests := []struct {
		name string
		resp *pb.SessionActionRollResponse
		want string
	}{
		{
			name: "critical success",
			resp: &pb.SessionActionRollResponse{Crit: true},
			want: pb.Outcome_CRITICAL_SUCCESS.String(),
		},
		{
			name: "success with hope",
			resp: &pb.SessionActionRollResponse{Success: true, Flavor: "hope"},
			want: pb.Outcome_SUCCESS_WITH_HOPE.String(),
		},
		{
			name: "success with fear",
			resp: &pb.SessionActionRollResponse{Success: true, Flavor: "fear"},
			want: pb.Outcome_SUCCESS_WITH_FEAR.String(),
		},
		{
			name: "failure with hope",
			resp: &pb.SessionActionRollResponse{Success: false, Flavor: "hope"},
			want: pb.Outcome_FAILURE_WITH_HOPE.String(),
		},
		{
			name: "failure with fear",
			resp: &pb.SessionActionRollResponse{Success: false, Flavor: "fear"},
			want: pb.Outcome_FAILURE_WITH_FEAR.String(),
		},
		{
			name: "unknown flavor",
			resp: &pb.SessionActionRollResponse{Success: true, Flavor: "mixed"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionActionOutcomeLabel(tt.resp); got != tt.want {
				t.Fatalf("sessionActionOutcomeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvedOutcomeUpdateFromProtoReturnsNilWhenEmpty(t *testing.T) {
	if resolvedOutcomeUpdateFromProto(nil) != nil {
		t.Fatal("resolvedOutcomeUpdateFromProto(nil) = non-nil, want nil")
	}
	if resolvedOutcomeUpdateFromProto(&pb.OutcomeUpdated{}) != nil {
		t.Fatal("resolvedOutcomeUpdateFromProto(empty) = non-nil, want nil")
	}

	update := resolvedOutcomeUpdateFromProto(&pb.OutcomeUpdated{
		CharacterStates: []*pb.OutcomeCharacterState{{
			CharacterId: "char-1",
			Hope:        2,
			Stress:      1,
			Hp:          5,
		}},
		GmFear: int32Ptr(3),
	})
	if update == nil || len(update.CharacterStates) != 1 {
		t.Fatalf("resolvedOutcomeUpdateFromProto() = %#v", update)
	}
	if update.GMFear == nil || *update.GMFear != 3 {
		t.Fatalf("gm_fear = %#v, want 3", update.GMFear)
	}
}

func TestCountdownSummariesAndConditionsShapeProtoState(t *testing.T) {
	sceneSummary := countdownSummaryFromSceneProto(&pb.DaggerheartSceneCountdown{
		CountdownId:       "clock-1",
		Name:              "Ritual",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD,
		StartingValue:     6,
		RemainingValue:    2,
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET,
		Status:            pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
		LinkedCountdownId: "clock-2",
	})
	if sceneSummary.Name != "Ritual" || sceneSummary.Tone != "CONSEQUENCE" || sceneSummary.Status != "ACTIVE" {
		t.Fatalf("scene summary = %#v", sceneSummary)
	}

	campaignSummary := countdownSummaryFromCampaignProto(&pb.DaggerheartCampaignCountdown{
		CountdownId:       "camp-clock",
		Name:              "War",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
	})
	if campaignSummary.Name != "War" || campaignSummary.AdvancementPolicy != "MANUAL" {
		t.Fatalf("campaign summary = %#v", campaignSummary)
	}

	conditions := conditionsFromProto([]*pb.DaggerheartConditionState{
		{Code: "hidden", ClearTriggers: []pb.DaggerheartConditionClearTrigger{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST}},
		{Label: "Marked", ClearTriggers: []pb.DaggerheartConditionClearTrigger{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN}},
		{},
	})
	if len(conditions) != 2 {
		t.Fatalf("conditions length = %d, want 2", len(conditions))
	}
	if conditions[0].Label != "hidden" || conditions[1].ClearTriggers[0] != "DAMAGE_TAKEN" {
		t.Fatalf("conditions = %#v", conditions)
	}
}

func TestCompactStringsAndRangeInputHelpersNormalizeValues(t *testing.T) {
	values := compactStrings([]string{" alpha ", "", "beta", "alpha", " beta "})
	if len(values) != 2 || values[0] != "alpha" || values[1] != "beta" {
		t.Fatalf("compactStrings() = %#v", values)
	}

	if rangeInputToRNG(&rangeInput{Min: 1, Max: 6}) != nil {
		t.Fatal("rangeInputToRNG(without seed) = non-nil, want nil")
	}
	seed := uint64(99)
	req := rangeInputToRNG(&rangeInput{Seed: &seed})
	if req == nil || req.Seed == nil || *req.Seed != 99 {
		t.Fatalf("rangeInputToRNG() = %#v", req)
	}
}

func TestGMMoveApplyRequestFromInputRequiresExactlyOneTarget(t *testing.T) {
	_, err := gmMoveApplyRequestFromInput("camp-1", "sess-1", "scene-1", gmMoveApplyInput{FearSpent: 1})
	if err == nil || err.Error() != "one gm move spend target is required" {
		t.Fatalf("gmMoveApplyRequestFromInput() error = %v", err)
	}

	_, err = gmMoveApplyRequestFromInput("camp-1", "sess-1", "scene-1", gmMoveApplyInput{
		FearSpent: 1,
		DirectMove: &gmMoveDirectMoveInput{
			Kind:  "interrupt_and_move",
			Shape: "custom",
		},
		AdversaryFeature: &gmMoveAdversaryFeatureInput{
			AdversaryID: "adv-1",
			FeatureID:   "fear-1",
		},
	})
	if err == nil || err.Error() != "only one gm move spend target may be provided" {
		t.Fatalf("gmMoveApplyRequestFromInput() multi-target error = %v", err)
	}
}

func TestGMMoveApplyRequestFromInputBuildsSpendTarget(t *testing.T) {
	req, err := gmMoveApplyRequestFromInput("camp-1", "sess-1", "scene-1", gmMoveApplyInput{
		FearSpent: 2,
		DirectMove: &gmMoveDirectMoveInput{
			Kind:        " interrupt_and_move ",
			Shape:       "custom",
			Description: " Close the exit ",
			AdversaryID: " adv-7 ",
		},
	})
	if err != nil {
		t.Fatalf("gmMoveApplyRequestFromInput() error = %v", err)
	}
	if req.GetCampaignId() != "camp-1" || req.GetSessionId() != "sess-1" || req.GetSceneId() != "scene-1" {
		t.Fatalf("request IDs = %#v", req)
	}
	directMove := req.GetDirectMove()
	if directMove == nil {
		t.Fatal("direct move target = nil, want populated target")
	}
	if directMove.GetKind() != pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE {
		t.Fatalf("kind = %v", directMove.GetKind())
	}
	if directMove.GetDescription() != "Close the exit" || directMove.GetAdversaryId() != "adv-7" {
		t.Fatalf("direct move = %#v", directMove)
	}
}

func int32Ptr(value int32) *int32 {
	return &value
}
