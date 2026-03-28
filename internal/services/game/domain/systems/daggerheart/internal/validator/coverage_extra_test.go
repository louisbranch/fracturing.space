package validator

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestValidateAdditionalPayloadFamilies(t *testing.T) {
	t.Parallel()

	hidden := mustStandardConditionState(t, rules.ConditionHidden)
	restrained := mustStandardConditionState(t, rules.ConditionRestrained)
	rollSeq := uint64(3)
	hpBefore := 6
	hpAfter := 4
	stressBefore := 1
	stressAfter := 2

	t.Run("beastform payloads", func(t *testing.T) {
		t.Parallel()

		active := &daggerheartstate.CharacterActiveBeastformState{BeastformID: "wolf-form"}
		classBefore := &daggerheartstate.CharacterClassState{}
		classAfter := &daggerheartstate.CharacterClassState{ActiveBeastform: active}

		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
			ClassStateBefore: classBefore,
			ClassStateAfter:  classAfter,
		})), "")
		assertErrorContains(t, ValidateBeastformTransformPayload(mustJSONRawMessage(t, payload.BeastformTransformPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
		})), "beastform transform must change at least one field")

		assertErrorContains(t, ValidateBeastformDropPayload(mustJSONRawMessage(t, payload.BeastformDropPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
			ClassStateBefore: classAfter,
			ClassStateAfter:  classBefore,
		})), "")
		assertErrorContains(t, ValidateBeastformDropPayload(mustJSONRawMessage(t, payload.BeastformDropPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			BeastformID:      "wolf-form",
			ClassStateBefore: classAfter,
			ClassStateAfter:  classAfter,
		})), "beastform drop must change class state")

		assertErrorContains(t, ValidateBeastformTransformedPayload(mustJSONRawMessage(t, payload.BeastformTransformedPayload{
			CharacterID:     ids.CharacterID("char-1"),
			BeastformID:     "wolf-form",
			ActiveBeastform: active,
		})), "")
		assertErrorContains(t, ValidateBeastformTransformedPayload(mustJSONRawMessage(t, payload.BeastformTransformedPayload{
			CharacterID: ids.CharacterID("char-1"),
			BeastformID: "wolf-form",
		})), "active_beastform is required")

		assertErrorContains(t, ValidateBeastformDroppedPayload(mustJSONRawMessage(t, payload.BeastformDroppedPayload{
			CharacterID: ids.CharacterID("char-1"),
			BeastformID: "wolf-form",
		})), "")
		assertErrorContains(t, ValidateBeastformDroppedPayload(mustJSONRawMessage(t, payload.BeastformDroppedPayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "beastform_id is required")
	})

	t.Run("companion payloads", func(t *testing.T) {
		t.Parallel()

		before := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
		after := &daggerheartstate.CharacterCompanionState{
			Status:             daggerheartstate.CompanionStatusAway,
			ActiveExperienceID: "scouting",
		}

		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			ActorCharacterID:     ids.CharacterID("actor-1"),
			CharacterID:          ids.CharacterID("char-1"),
			ExperienceID:         "scouting",
			CompanionStateBefore: before,
			CompanionStateAfter:  after,
		})), "")
		assertErrorContains(t, ValidateCompanionExperienceBeginPayload(mustJSONRawMessage(t, payload.CompanionExperienceBeginPayload{
			ActorCharacterID:     ids.CharacterID("actor-1"),
			CharacterID:          ids.CharacterID("char-1"),
			ExperienceID:         "scouting",
			CompanionStateBefore: before,
			CompanionStateAfter:  before,
		})), "companion begin must change companion state")

		assertErrorContains(t, ValidateCompanionReturnPayload(mustJSONRawMessage(t, payload.CompanionReturnPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			Resolution:       "complete",
			StressBefore:     &stressBefore,
			StressAfter:      &stressAfter,
		})), "")
		assertErrorContains(t, ValidateCompanionReturnPayload(mustJSONRawMessage(t, payload.CompanionReturnPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			CharacterID:      ids.CharacterID("char-1"),
			Resolution:       "complete",
		})), "companion return must change at least one field")

		assertErrorContains(t, ValidateCompanionExperienceBegunPayload(mustJSONRawMessage(t, payload.CompanionExperienceBegunPayload{
			CharacterID:    ids.CharacterID("char-1"),
			ExperienceID:   "scouting",
			CompanionState: after,
		})), "")
		assertErrorContains(t, ValidateCompanionExperienceBegunPayload(mustJSONRawMessage(t, payload.CompanionExperienceBegunPayload{
			CharacterID:  ids.CharacterID("char-1"),
			ExperienceID: "scouting",
		})), "companion_state is required")

		assertErrorContains(t, ValidateCompanionReturnedPayload(mustJSONRawMessage(t, payload.CompanionReturnedPayload{
			CharacterID:    ids.CharacterID("char-1"),
			Resolution:     "complete",
			CompanionState: before,
		})), "")
		assertErrorContains(t, ValidateCompanionReturnedPayload(mustJSONRawMessage(t, payload.CompanionReturnedPayload{
			CharacterID: ids.CharacterID("char-1"),
			Resolution:  "complete",
		})), "companion_state is required")
	})

	t.Run("environment payloads", func(t *testing.T) {
		t.Parallel()

		entity := payload.EnvironmentEntityUpdatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "fog-bank",
			Name:                "Fog Bank",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          14,
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
		}

		assertErrorContains(t, ValidateEnvironmentEntityUpdatePayload(mustJSONRawMessage(t, entity)), "")
		entity.Difficulty = 0
		assertErrorContains(t, ValidateEnvironmentEntityUpdatedPayload(mustJSONRawMessage(t, entity)), "difficulty must be positive")

		assertErrorContains(t, ValidateEnvironmentEntityDeletePayload(mustJSONRawMessage(t, payload.EnvironmentEntityDeletePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
		})), "")
		assertErrorContains(t, ValidateEnvironmentEntityDeletedPayload(mustJSONRawMessage(t, payload.EnvironmentEntityDeletePayload{})), "environment_entity_id is required")
	})

	t.Run("multi target damage apply", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateMultiTargetDamageApplyPayload(mustJSONRawMessage(t, payload.MultiTargetDamageApplyPayload{
			Targets: []payload.DamageApplyPayload{
				{
					CharacterID: ids.CharacterID("char-1"),
					HpBefore:    &hpBefore,
					HpAfter:     &hpAfter,
				},
			},
		})), "")
		assertErrorContains(t, ValidateMultiTargetDamageApplyPayload(mustJSONRawMessage(t, payload.MultiTargetDamageApplyPayload{
			Targets: []payload.DamageApplyPayload{
				{
					CharacterID: ids.CharacterID("char-1"),
				},
			},
		})), "targets[0]: damage apply must change hp, stress, or armor")
	})

	t.Run("character and subclass payloads", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateCharacterProfileDeletePayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileDeletePayload{
			CharacterID: ids.CharacterID("char-1"),
		})), "")
		assertErrorContains(t, ValidateCharacterProfileDeletedPayload(mustJSONRawMessage(t, daggerheartstate.CharacterProfileDeletedPayload{})), "character_id is required")

		subclassAfter := &daggerheartstate.CharacterSubclassState{ElementalistActionBonus: 1}
		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "elemental nova",
			Targets: []payload.SubclassFeatureTargetPatchPayload{
				{
					CharacterID:        ids.CharacterID("char-1"),
					SubclassStateAfter: subclassAfter,
				},
			},
		})), "")
		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "elemental nova",
			Targets: []payload.SubclassFeatureTargetPatchPayload{
				{CharacterID: ids.CharacterID("char-1")},
			},
		})), "subclass_feature apply must change at least one field")
	})

	t.Run("helper state mutation predicates", func(t *testing.T) {
		t.Parallel()

		if HasCompanionStateFieldChange(nil, nil) {
			t.Fatal("HasCompanionStateFieldChange(nil, nil) = true, want false")
		}
		if !HasCompanionStateFieldChange(nil, &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}) {
			t.Fatal("HasCompanionStateFieldChange(nil, value) = false, want true")
		}
		if HasConditionListMutation([]string{"hidden"}, []string{" hidden "}) {
			t.Fatal("HasConditionListMutation(normalized equal) = true, want false")
		}
		same := true
		if HasBoolFieldChange(&same, &same) {
			t.Fatal("HasBoolFieldChange(equal) = true, want false")
		}
		flipped := false
		if !HasBoolFieldChange(&same, &flipped) {
			t.Fatal("HasBoolFieldChange(flip) = false, want true")
		}
	})

	t.Run("condition-bearing subclass payload remains valid", func(t *testing.T) {
		t.Parallel()

		assertErrorContains(t, ValidateSubclassFeatureApplyPayload(mustJSONRawMessage(t, payload.SubclassFeatureApplyPayload{
			ActorCharacterID: ids.CharacterID("actor-1"),
			Feature:          "bind",
			CharacterConditionTargets: []payload.ConditionChangePayload{
				{
					CharacterID:     ids.CharacterID("char-1"),
					ConditionsAfter: []rules.ConditionState{hidden, restrained},
					Added:           []rules.ConditionState{restrained},
					RollSeq:         &rollSeq,
				},
			},
		})), "")
	})
}
