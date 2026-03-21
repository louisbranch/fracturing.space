package statmodifiertransport

import (
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

// StatModifierView is the transport-layer representation of a stat modifier.
type StatModifierView struct {
	ID            string
	Target        string
	Delta         int
	Label         string
	Source        string
	SourceID      string
	ClearTriggers []pb.DaggerheartConditionClearTrigger
}

// StatModifierFromProto converts a proto stat modifier to a transport view.
func StatModifierFromProto(m *pb.DaggerheartStatModifier) (*StatModifierView, error) {
	if m == nil {
		return nil, fmt.Errorf("stat modifier is required")
	}
	id := strings.TrimSpace(m.GetId())
	if id == "" {
		return nil, fmt.Errorf("stat modifier id is required")
	}
	target := strings.TrimSpace(m.GetTarget())
	if !rules.ValidStatModifierTarget(target) {
		return nil, fmt.Errorf("stat modifier target %q is invalid", target)
	}
	for _, trigger := range m.GetClearTriggers() {
		if trigger == pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED {
			return nil, fmt.Errorf("stat modifier clear trigger is required")
		}
	}
	return &StatModifierView{
		ID:            id,
		Target:        target,
		Delta:         int(m.GetDelta()),
		Label:         strings.TrimSpace(m.GetLabel()),
		Source:        strings.TrimSpace(m.GetSource()),
		SourceID:      strings.TrimSpace(m.GetSourceId()),
		ClearTriggers: append([]pb.DaggerheartConditionClearTrigger(nil), m.GetClearTriggers()...),
	}, nil
}

// StatModifiersFromProto converts proto stat modifiers to transport views.
func StatModifiersFromProto(values []*pb.DaggerheartStatModifier) ([]*StatModifierView, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]*StatModifierView, 0, len(values))
	for _, v := range values {
		view, err := StatModifierFromProto(v)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

// StatModifierViewsToDomain converts transport views to domain stat modifiers.
func StatModifierViewsToDomain(views []*StatModifierView) []rules.StatModifierState {
	if len(views) == 0 {
		return nil
	}
	result := make([]rules.StatModifierState, 0, len(views))
	for _, v := range views {
		if v == nil {
			continue
		}
		entry := rules.StatModifierState{
			ID:       v.ID,
			Target:   rules.StatModifierTarget(v.Target),
			Delta:    v.Delta,
			Label:    v.Label,
			Source:   v.Source,
			SourceID: v.SourceID,
		}
		for _, trigger := range v.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, clearTriggerFromProto(trigger))
		}
		result = append(result, entry)
	}
	return result
}

// DomainStatModifiersToViews converts domain stat modifiers to transport views.
func DomainStatModifiersToViews(values []rules.StatModifierState) []*StatModifierView {
	if len(values) == 0 {
		return nil
	}
	result := make([]*StatModifierView, 0, len(values))
	for _, v := range values {
		result = append(result, &StatModifierView{
			ID:            v.ID,
			Target:        string(v.Target),
			Delta:         v.Delta,
			Label:         v.Label,
			Source:        v.Source,
			SourceID:      v.SourceID,
			ClearTriggers: domainClearTriggersToProto(v.ClearTriggers),
		})
	}
	return result
}

// StatModifierViewsToProto converts transport views to proto stat modifiers.
func StatModifierViewsToProto(views []*StatModifierView) []*pb.DaggerheartStatModifier {
	if len(views) == 0 {
		return nil
	}
	result := make([]*pb.DaggerheartStatModifier, 0, len(views))
	for _, v := range views {
		if v == nil {
			continue
		}
		result = append(result, &pb.DaggerheartStatModifier{
			Id:            v.ID,
			Target:        v.Target,
			Delta:         int32(v.Delta),
			Label:         v.Label,
			Source:        v.Source,
			SourceId:      v.SourceID,
			ClearTriggers: append([]pb.DaggerheartConditionClearTrigger(nil), v.ClearTriggers...),
		})
	}
	return result
}

// ProjectionStatModifiersToViews converts projection stat modifiers to views.
func ProjectionStatModifiersToViews(values []projectionstore.DaggerheartStatModifier) []*StatModifierView {
	if len(values) == 0 {
		return nil
	}
	result := make([]*StatModifierView, 0, len(values))
	for _, v := range values {
		result = append(result, &StatModifierView{
			ID:            v.ID,
			Target:        v.Target,
			Delta:         v.Delta,
			Label:         v.Label,
			Source:        v.Source,
			SourceID:      v.SourceID,
			ClearTriggers: projectionClearTriggersToProto(v.ClearTriggers),
		})
	}
	return result
}

// ProjectionStatModifiersToDomain converts projection stat modifiers to domain form.
func ProjectionStatModifiersToDomain(values []projectionstore.DaggerheartStatModifier) []rules.StatModifierState {
	if len(values) == 0 {
		return nil
	}
	result := make([]rules.StatModifierState, 0, len(values))
	for _, v := range values {
		entry := rules.StatModifierState{
			ID:       v.ID,
			Target:   rules.StatModifierTarget(v.Target),
			Delta:    v.Delta,
			Label:    v.Label,
			Source:   v.Source,
			SourceID: v.SourceID,
		}
		for _, trigger := range v.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, rules.ConditionClearTrigger(strings.TrimSpace(trigger)))
		}
		result = append(result, entry)
	}
	return result
}

// ProjectionStatModifiersToProto converts projection stat modifiers to proto.
func ProjectionStatModifiersToProto(values []projectionstore.DaggerheartStatModifier) []*pb.DaggerheartStatModifier {
	return StatModifierViewsToProto(ProjectionStatModifiersToViews(values))
}

// clearTriggerFromProto reuses the condition clear trigger enum for stat
// modifier durations (same lifecycle semantics).
func clearTriggerFromProto(trigger pb.DaggerheartConditionClearTrigger) rules.ConditionClearTrigger {
	switch trigger {
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST:
		return rules.ConditionClearTriggerShortRest
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST:
		return rules.ConditionClearTriggerLongRest
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END:
		return rules.ConditionClearTriggerSessionEnd
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN:
		return rules.ConditionClearTriggerDamageTaken
	default:
		return ""
	}
}

// domainClearTriggersToProto converts domain clear triggers to proto enum.
func domainClearTriggersToProto(values []rules.ConditionClearTrigger) []pb.DaggerheartConditionClearTrigger {
	if len(values) == 0 {
		return nil
	}
	result := make([]pb.DaggerheartConditionClearTrigger, 0, len(values))
	for _, v := range values {
		result = append(result, domainClearTriggerToProto(v))
	}
	return result
}

// projectionClearTriggersToProto converts stored string triggers to proto enum.
func projectionClearTriggersToProto(values []string) []pb.DaggerheartConditionClearTrigger {
	if len(values) == 0 {
		return nil
	}
	result := make([]pb.DaggerheartConditionClearTrigger, 0, len(values))
	for _, v := range values {
		result = append(result, domainClearTriggerToProto(rules.ConditionClearTrigger(strings.TrimSpace(v))))
	}
	return result
}

func domainClearTriggerToProto(value rules.ConditionClearTrigger) pb.DaggerheartConditionClearTrigger {
	switch value {
	case rules.ConditionClearTriggerShortRest:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST
	case rules.ConditionClearTriggerLongRest:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST
	case rules.ConditionClearTriggerSessionEnd:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END
	case rules.ConditionClearTriggerDamageTaken:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN
	default:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED
	}
}
