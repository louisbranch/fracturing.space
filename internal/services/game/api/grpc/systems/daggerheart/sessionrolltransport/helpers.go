package sessionrolltransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

type hopeSpend struct {
	Source string
	Amount int
}

func normalizeActionModifiers(modifiers []*pb.ActionRollModifier) (int, []workflowtransport.RollModifierMetadata) {
	if len(modifiers) == 0 {
		return 0, nil
	}

	entries := make([]workflowtransport.RollModifierMetadata, 0, len(modifiers))
	total := 0
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		value := int(modifier.GetValue())
		total += value
		entry := workflowtransport.RollModifierMetadata{Value: value}
		if source := strings.TrimSpace(modifier.GetSource()); source != "" {
			entry.Source = source
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return total, nil
	}
	return total, entries
}

func normalizeRollKind(kind pb.RollKind) pb.RollKind {
	if kind == pb.RollKind_ROLL_KIND_UNSPECIFIED {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	return kind
}

func damageDiceFromProto(specs []*pb.DiceSpec) ([]bridge.DamageDieSpec, error) {
	if len(specs) == 0 {
		return nil, dice.ErrMissingDice
	}
	converted := make([]bridge.DamageDieSpec, 0, len(specs))
	for _, spec := range specs {
		if spec == nil {
			return nil, dice.ErrInvalidDiceSpec
		}
		sides := int(spec.GetSides())
		count := int(spec.GetCount())
		if sides <= 0 || count <= 0 {
			return nil, dice.ErrInvalidDiceSpec
		}
		converted = append(converted, bridge.DamageDieSpec{Sides: sides, Count: count})
	}
	return converted, nil
}

func diceRollsToProto(rolls []dice.Roll) []*pb.DiceRoll {
	if len(rolls) == 0 {
		return nil
	}
	converted := make([]*pb.DiceRoll, 0, len(rolls))
	for _, roll := range rolls {
		results := make([]int32, 0, len(roll.Results))
		for _, value := range roll.Results {
			results = append(results, int32(value))
		}
		converted = append(converted, &pb.DiceRoll{
			Sides:   int32(roll.Sides),
			Results: results,
			Total:   int32(roll.Total),
		})
	}
	return converted
}

func resolveRoll(kind pb.RollKind, request daggerheartdomain.ActionRequest) (daggerheartdomain.ActionResult, bool, bool, bool, error) {
	switch normalizeRollKind(kind) {
	case pb.RollKind_ROLL_KIND_REACTION:
		result, err := daggerheartdomain.RollReaction(daggerheartdomain.ReactionRequest{
			Modifier:   request.Modifier,
			Difficulty: request.Difficulty,
			Seed:       request.Seed,
		})
		if err != nil {
			return daggerheartdomain.ActionResult{}, false, false, false, err
		}
		return result.ActionResult, result.GeneratesHopeFear, result.TriggersGMMove, result.CritNegatesEffects, nil
	default:
		result, err := daggerheartdomain.RollAction(request)
		if err != nil {
			return daggerheartdomain.ActionResult{}, true, true, false, err
		}
		return result, true, true, false, nil
	}
}

func hopeSpendsFromModifiers(modifiers []*pb.ActionRollModifier) []hopeSpend {
	if len(modifiers) == 0 {
		return nil
	}

	spends := make([]hopeSpend, 0)
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		sourceKey := normalizeHopeSpendSource(modifier.GetSource())
		amount := 0
		switch sourceKey {
		case "experience", "help":
			amount = 1
		case "tag_team", "hope_feature":
			amount = 3
		default:
			continue
		}
		spends = append(spends, hopeSpend{Source: sourceKey, Amount: amount})
	}

	if len(spends) == 0 {
		return nil
	}
	return spends
}

func normalizeHopeSpendSource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(normalized)
}

func outcomeToProto(outcome daggerheartdomain.Outcome) pb.Outcome {
	switch outcome {
	case daggerheartdomain.OutcomeRollWithHope:
		return pb.Outcome_ROLL_WITH_HOPE
	case daggerheartdomain.OutcomeRollWithFear:
		return pb.Outcome_ROLL_WITH_FEAR
	case daggerheartdomain.OutcomeSuccessWithHope:
		return pb.Outcome_SUCCESS_WITH_HOPE
	case daggerheartdomain.OutcomeSuccessWithFear:
		return pb.Outcome_SUCCESS_WITH_FEAR
	case daggerheartdomain.OutcomeFailureWithHope:
		return pb.Outcome_FAILURE_WITH_HOPE
	case daggerheartdomain.OutcomeFailureWithFear:
		return pb.Outcome_FAILURE_WITH_FEAR
	case daggerheartdomain.OutcomeCriticalSuccess:
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}
