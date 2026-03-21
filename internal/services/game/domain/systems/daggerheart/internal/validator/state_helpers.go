package validator

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func HasCharacterStateChange(p payload.CharacterStatePatchPayload) bool {
	return HasIntFieldChange(p.HPBefore, p.HPAfter) ||
		HasIntFieldChange(p.HopeBefore, p.HopeAfter) ||
		HasIntFieldChange(p.HopeMaxBefore, p.HopeMaxAfter) ||
		HasIntFieldChange(p.StressBefore, p.StressAfter) ||
		HasIntFieldChange(p.ArmorBefore, p.ArmorAfter) ||
		HasStringFieldChange(p.LifeStateBefore, p.LifeStateAfter) ||
		HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) ||
		HasSubclassStateFieldChange(p.SubclassStateBefore, p.SubclassStateAfter) ||
		HasBoolFieldChange(p.ImpenetrableUsedThisShortRestBefore, p.ImpenetrableUsedThisShortRestAfter)
}

func HasClassStateFieldChange(before, after *daggerheartstate.CharacterClassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !before.Equal(*after)
}

func HasCompanionStateFieldChange(before, after *daggerheartstate.CharacterCompanionState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !before.Equal(*after)
}

func HasSubclassStateFieldChange(before, after *daggerheartstate.CharacterSubclassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !before.Equal(*after)
}

func HasConditionListMutation(before, after []string) bool {
	beforeNormalized, err := rules.NormalizeConditions(before)
	if err != nil {
		return true
	}
	afterNormalized, err := rules.NormalizeConditions(after)
	if err != nil {
		return true
	}
	return !rules.ConditionsEqual(beforeNormalized, afterNormalized)
}

func HasIntFieldChange(before, after *int) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func HasStringFieldChange(before, after *string) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func HasBoolFieldChange(before, after *bool) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func Abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
