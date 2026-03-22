package decider

import (
	"reflect"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// ── Snapshot lookup helpers ────────────────────────────────────────────

func snapshotCharacterState(snapshot daggerheartstate.SnapshotState, characterID ids.CharacterID) (daggerheartstate.CharacterState, bool) {
	trimmed, ok := normalize.RequireID(characterID)
	if !ok {
		return daggerheartstate.CharacterState{}, false
	}
	character, found := snapshot.CharacterStates[trimmed]
	if !found {
		return daggerheartstate.CharacterState{}, false
	}
	character.CharacterID = trimmed.String()
	character.CampaignID = snapshot.CampaignID.String()
	if character.LifeState == "" {
		character.LifeState = daggerheartstate.LifeStateAlive
	}
	return character, true
}

func snapshotAdversaryState(snapshot daggerheartstate.SnapshotState, adversaryID dhids.AdversaryID) (daggerheartstate.AdversaryState, bool) {
	trimmed, ok := normalize.RequireID(adversaryID)
	if !ok {
		return daggerheartstate.AdversaryState{}, false
	}
	adversary, found := snapshot.AdversaryStates[trimmed]
	if !found {
		return daggerheartstate.AdversaryState{}, false
	}
	adversary.AdversaryID = trimmed
	adversary.CampaignID = snapshot.CampaignID
	return adversary, true
}

// ── Shared damage validation ──────────────────────────────────────────

// rejectArmorSpendLimit rejects commands that try to spend more than one armor
// slot in a single damage application.
func rejectArmorSpendLimit(armorSpent int) *command.Rejection {
	if armorSpent > 1 {
		return &command.Rejection{
			Code:    rejectionCodeDamageArmorSpendLimit,
			Message: "damage apply can spend at most one armor slot",
		}
	}
	return nil
}

// rejectDamageBeforeMismatch checks HP and Armor "before" fields against current
// state. This is the shared validation used by character damage, adversary
// damage, and multi-target damage handlers.
func rejectDamageBeforeMismatch(hpBefore *int, currentHP int, armorBefore *int, currentArmor int, code, message string) *command.Rejection {
	if hpBefore != nil && currentHP != *hpBefore {
		return &command.Rejection{Code: code, Message: message}
	}
	if armorBefore != nil && currentArmor != *armorBefore {
		return &command.Rejection{Code: code, Message: message}
	}
	return nil
}

// characterDamageAppliedPayload builds a DamageAppliedPayload from a
// DamageApplyPayload, eliminating the duplicated field-by-field construction
// across decideDamageApply and decideMultiTargetDamageApply.
func characterDamageAppliedPayload(p payload.DamageApplyPayload) payload.DamageAppliedPayload {
	return payload.DamageAppliedPayload{
		CharacterID:        p.CharacterID,
		Hp:                 p.HpAfter,
		Stress:             p.StressAfter,
		Armor:              p.ArmorAfter,
		ArmorSpent:         p.ArmorSpent,
		Severity:           p.Severity,
		Marks:              p.Marks,
		DamageType:         p.DamageType,
		RollSeq:            p.RollSeq,
		ResistPhysical:     p.ResistPhysical,
		ResistMagic:        p.ResistMagic,
		ImmunePhysical:     p.ImmunePhysical,
		ImmuneMagic:        p.ImmuneMagic,
		Direct:             p.Direct,
		MassiveDamage:      p.MassiveDamage,
		Mitigated:          p.Mitigated,
		Source:             p.Source,
		SourceCharacterIDs: p.SourceCharacterIDs,
	}
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
