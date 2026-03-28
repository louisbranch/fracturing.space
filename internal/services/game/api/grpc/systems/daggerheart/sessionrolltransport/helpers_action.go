package sessionrolltransport

import (
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
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

func appendActionModifier(total int, entries []workflowtransport.RollModifierMetadata, source string, value int) (int, []workflowtransport.RollModifierMetadata) {
	if value == 0 {
		return total, entries
	}
	total += value
	entry := workflowtransport.RollModifierMetadata{Value: value}
	if trimmed := strings.TrimSpace(source); trimmed != "" {
		entry.Source = trimmed
	}
	return total, append(entries, entry)
}

func normalizeRollKind(kind pb.RollKind) pb.RollKind {
	if kind == pb.RollKind_ROLL_KIND_UNSPECIFIED {
		return pb.RollKind_ROLL_KIND_ACTION
	}
	return kind
}

func normalizeActionRollContext(context pb.ActionRollContext) pb.ActionRollContext {
	if context == pb.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED {
		return pb.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED
	}
	return context
}

func actionRollContextCode(context pb.ActionRollContext) string {
	switch normalizeActionRollContext(context) {
	case pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY:
		return "move_silently"
	default:
		return ""
	}
}

func resolveRoll(kind pb.RollKind, request daggerheartdomain.ActionRequest) (daggerheartdomain.ActionResult, bool, bool, bool, error) {
	switch normalizeRollKind(kind) {
	case pb.RollKind_ROLL_KIND_REACTION:
		result, err := daggerheartdomain.RollReaction(daggerheartdomain.ReactionRequest{Modifier: request.Modifier, Difficulty: request.Difficulty, Seed: request.Seed})
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

func normalizeHopeSpends(values []*pb.ActionRollHopeSpend) ([]hopeSpend, error) {
	if len(values) == 0 {
		return nil, nil
	}

	spends := make([]hopeSpend, 0, len(values))
	for index, value := range values {
		if value == nil {
			continue
		}
		sourceKey := normalizeHopeSpendSource(value.GetSource())
		expectedAmount, ok := hopeSpendAmountForSource(sourceKey)
		if !ok {
			return nil, fmt.Errorf("hope_spends[%d].source is unsupported", index)
		}
		amount := int(value.GetAmount())
		if amount != expectedAmount {
			return nil, fmt.Errorf("hope_spends[%d].amount must be %d for source %q", index, expectedAmount, sourceKey)
		}
		spends = append(spends, hopeSpend{Source: sourceKey, Amount: amount})
	}
	if len(spends) == 0 {
		return nil, nil
	}
	return spends, nil
}

func hopeSpendAmountForSource(source string) (int, bool) {
	switch source {
	case "experience", "help":
		return 1, true
	case "tag_team", "hope_feature":
		return 3, true
	default:
		return 0, false
	}
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
