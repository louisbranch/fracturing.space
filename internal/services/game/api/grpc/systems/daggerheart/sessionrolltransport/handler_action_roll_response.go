package sessionrolltransport

import (
	"encoding/json"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

func (h *Handler) buildSessionActionRollPayload(
	actionRoll actionRollContext,
	result daggerheartdomain.ActionResult,
	requestID string,
	seed uint64,
	seedSource string,
	rollMode commonv1.RollMode,
	rollSeq uint64,
	generateHopeFear bool,
	triggerGMMove bool,
	critNegatesEffects bool,
) ([]byte, string, error) {
	outcomeCode := outcomeToProto(result.Outcome).String()
	flavor := workflowtransport.OutcomeFlavorFromCode(outcomeCode)
	if !generateHopeFear {
		flavor = ""
	}

	results := map[string]any{
		"rng": map[string]any{
			"seed_used":   uint64(seed),
			"rng_algo":    random.RngAlgoMathRandV1,
			"seed_source": seedSource,
			"roll_mode":   rollMode.String(),
		},
		"dice": map[string]any{
			"hope_die":      result.Hope,
			"fear_die":      result.Fear,
			"advantage_die": result.AdvantageDie,
		},
		workflowtransport.KeyModifier: result.Modifier,
		"advantage_modifier":          result.AdvantageModifier,
		workflowtransport.KeyTotal:    result.Total,
		"difficulty":                  actionRoll.Difficulty,
		"success":                     result.MeetsDifficulty,
		workflowtransport.KeyCrit:     result.IsCrit,
	}
	if len(actionRoll.ModifierMetadata) > 0 {
		results["modifiers"] = actionRoll.ModifierMetadata
	}

	systemMetadata := workflowtransport.RollSystemMetadata{
		CharacterID:  actionRoll.CharacterID,
		Trait:        actionRoll.Trait,
		RollKind:     actionRoll.RollKind.String(),
		Outcome:      outcomeCode,
		Flavor:       flavor,
		HopeFear:     workflowtransport.BoolPtr(generateHopeFear),
		Crit:         workflowtransport.BoolPtr(result.IsCrit),
		CritNegates:  workflowtransport.BoolPtr(critNegatesEffects),
		GMMove:       workflowtransport.BoolPtr(triggerGMMove),
		Advantage:    workflowtransport.IntPtr(actionRoll.Advantage),
		Disadvantage: workflowtransport.IntPtr(actionRoll.Disadvantage),
		Underwater:   workflowtransport.BoolPtr(actionRoll.Underwater),
		Modifiers:    actionRoll.ModifierMetadata,
	}
	if actionRoll.BreathCountdownID != "" {
		systemMetadata.BreathCountdownID = actionRoll.BreathCountdownID
	}

	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID:  requestID,
		RollSeq:    rollSeq,
		Results:    results,
		Outcome:    outcomeCode,
		SystemData: systemMetadata.MapValue(),
	})
	if err != nil {
		return nil, "", grpcerror.Internal("encode payload", err)
	}

	return payloadJSON, flavor, nil
}

func buildSessionActionRollResponse(
	rollSeq uint64,
	result daggerheartdomain.ActionResult,
	difficulty int,
	flavor string,
	seed uint64,
	seedSource string,
	rollMode commonv1.RollMode,
) *pb.SessionActionRollResponse {
	return &pb.SessionActionRollResponse{
		RollSeq:    rollSeq,
		HopeDie:    int32(result.Hope),
		FearDie:    int32(result.Fear),
		Crit:       result.IsCrit,
		Total:      int32(result.Total),
		Difficulty: int32(difficulty),
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Rng: &commonv1.RngResponse{
			SeedUsed:   seed,
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}
}
