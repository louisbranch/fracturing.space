package decider

import (
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// ── Snapshot lookup helpers ────────────────────────────────────────────

func snapshotCharacterState(snapshot daggerheartstate.SnapshotState, characterID ids.CharacterID) (daggerheartstate.CharacterState, bool) {
	trimmed := ids.CharacterID(strings.TrimSpace(characterID.String()))
	if trimmed == "" {
		return daggerheartstate.CharacterState{}, false
	}
	character, ok := snapshot.CharacterStates[trimmed]
	if !ok {
		return daggerheartstate.CharacterState{}, false
	}
	character.CharacterID = trimmed.String()
	character.CampaignID = snapshot.CampaignID.String()
	if character.LifeState == "" {
		character.LifeState = daggerheartstate.LifeStateAlive
	}
	return character, true
}

func snapshotAdversaryState(snapshot daggerheartstate.SnapshotState, adversaryID ids.AdversaryID) (daggerheartstate.AdversaryState, bool) {
	trimmed := ids.AdversaryID(strings.TrimSpace(adversaryID.String()))
	if trimmed == "" {
		return daggerheartstate.AdversaryState{}, false
	}
	adversary, ok := snapshot.AdversaryStates[trimmed]
	if !ok {
		return daggerheartstate.AdversaryState{}, false
	}
	adversary.AdversaryID = trimmed
	adversary.CampaignID = snapshot.CampaignID
	return adversary, true
}

// ── State mutation detection ───────────────────────────────────────────

func isCharacterStatePatchNoMutation(snapshot daggerheartstate.SnapshotState, p payload.CharacterStatePatchPayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, p.CharacterID)
	if !hasCharacter {
		return false
	}

	if p.HPAfter != nil {
		if character.HP != *p.HPAfter {
			return false
		}
	} else if p.HPBefore != nil && character.HP == 0 && character.HP != *p.HPBefore {
		return false
	}
	if p.HopeAfter != nil && character.Hope != *p.HopeAfter {
		return false
	}
	if p.HopeMaxAfter != nil && character.HopeMax != *p.HopeMaxAfter {
		return false
	}
	if p.StressAfter != nil && character.Stress != *p.StressAfter {
		return false
	}
	if p.ArmorAfter != nil && character.Armor != *p.ArmorAfter {
		return false
	}
	if p.LifeStateAfter != nil && character.LifeState != *p.LifeStateAfter {
		return false
	}
	if p.ClassStateAfter != nil {
		current := snapshot.CharacterClassStates[p.CharacterID].Normalized()
		if !reflect.DeepEqual(current, p.ClassStateAfter.Normalized()) {
			return false
		}
	}
	if p.SubclassStateAfter != nil {
		current := snapshot.CharacterSubclassStates[p.CharacterID].Normalized()
		if !reflect.DeepEqual(current, p.SubclassStateAfter.Normalized()) {
			return false
		}
	}

	return true
}

func normalizedClassStatePtr(value *daggerheartstate.CharacterClassState) *daggerheartstate.CharacterClassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}

// ── Condition helpers ──────────────────────────────────────────────────

func hasMissingConditionRemovals(current, removed []string) bool {
	normalizedCurrent, err := rules.NormalizeConditions(current)
	if err != nil {
		return false
	}
	normalizedRemoved, err := rules.NormalizeConditions(removed)
	if err != nil {
		return false
	}

	currentSet := make(map[string]struct{}, len(normalizedCurrent))
	for _, value := range normalizedCurrent {
		currentSet[value] = struct{}{}
	}
	for _, value := range normalizedRemoved {
		if _, ok := currentSet[value]; !ok {
			return true
		}
	}
	return false
}

// ── Small utility helpers ──────────────────────────────────────────────

func hasIntFieldChange(before, after *int) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func equalAdversaryFeatureStates(before, after []rules.AdversaryFeatureState) bool {
	if len(before) != len(after) {
		return false
	}
	for i := range before {
		if before[i] != after[i] {
			return false
		}
	}
	return true
}

func equalAdversaryPendingExperience(before, after *rules.AdversaryPendingExperience) bool {
	if before == nil || after == nil {
		return before == after
	}
	return *before == *after
}

func derefInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
