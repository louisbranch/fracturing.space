package validator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestValidateLevelUpApplyPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload payload.LevelUpApplyPayload
		wantErr string
	}{
		{
			name: "accepts valid rewards and advancements",
			payload: payload.LevelUpApplyPayload{
				CharacterID: ids.CharacterID("char-1"),
				LevelBefore: 1,
				LevelAfter:  2,
				Advancements: []payload.LevelUpAdvancementPayload{
					{Type: string(mechanics.AdvTraitIncrease), Trait: "agility"},
					{Type: string(mechanics.AdvAddHPSlots)},
				},
				Rewards: []payload.LevelUpRewardPayload{
					{Type: "domain_card", DomainCardID: "card-1", DomainCardLevel: 1},
					{Type: "companion_bonus_choices", CompanionBonusChoices: 1},
				},
			},
		},
		{
			name: "rejects unsupported reward type",
			payload: payload.LevelUpApplyPayload{
				CharacterID: ids.CharacterID("char-1"),
				LevelBefore: 1,
				LevelAfter:  2,
				Advancements: []payload.LevelUpAdvancementPayload{
					{Type: string(mechanics.AdvTraitIncrease), Trait: "agility"},
					{Type: string(mechanics.AdvAddHPSlots)},
				},
				Rewards: []payload.LevelUpRewardPayload{
					{Type: "mystery_box"},
				},
			},
			wantErr: `reward type "mystery_box" is unsupported`,
		},
		{
			name: "rejects missing advancements",
			payload: payload.LevelUpApplyPayload{
				CharacterID: ids.CharacterID("char-1"),
				LevelBefore: 1,
				LevelAfter:  2,
			},
			wantErr: "advancements is required",
		},
		{
			name: "rejects non sequential level change",
			payload: payload.LevelUpApplyPayload{
				CharacterID: ids.CharacterID("char-1"),
				LevelBefore: 1,
				LevelAfter:  3,
				Advancements: []payload.LevelUpAdvancementPayload{
					{Type: string(mechanics.AdvTraitIncrease), Trait: "agility"},
				},
			},
			wantErr: "level_after must be level_before + 1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLevelUpApplyPayload(mustJSONRawMessage(t, tt.payload))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateDamageAppliedInvariants(t *testing.T) {
	t.Parallel()

	rollSeqZero := uint64(0)
	hp := 5

	tests := []struct {
		name    string
		payload payload.DamageAppliedPayload
		wantErr string
	}{
		{
			name: "accepts valid payload",
			payload: payload.DamageAppliedPayload{
				CharacterID: ids.CharacterID("char-1"),
				Hp:          &hp,
				ArmorSpent:  1,
				Marks:       1,
				Severity:    "major",
				SourceCharacterIDs: []ids.CharacterID{
					ids.CharacterID("char-2"),
				},
			},
		},
		{
			name: "rejects armor spent above cap",
			payload: payload.DamageAppliedPayload{
				CharacterID: ids.CharacterID("char-1"),
				Hp:          &hp,
				ArmorSpent:  mechanics.ArmorMaxCap + 1,
			},
			wantErr: "armor_spent must be in range",
		},
		{
			name: "rejects zero roll sequence",
			payload: payload.DamageAppliedPayload{
				CharacterID: ids.CharacterID("char-1"),
				Hp:          &hp,
				RollSeq:     &rollSeqZero,
			},
			wantErr: "roll_seq must be positive",
		},
		{
			name: "rejects unsupported severity",
			payload: payload.DamageAppliedPayload{
				CharacterID: ids.CharacterID("char-1"),
				Hp:          &hp,
				Severity:    "catastrophic",
			},
			wantErr: "severity must be one of",
		},
		{
			name: "rejects empty source character ids",
			payload: payload.DamageAppliedPayload{
				CharacterID:        ids.CharacterID("char-1"),
				Hp:                 &hp,
				SourceCharacterIDs: []ids.CharacterID{ids.CharacterID(" ")},
			},
			wantErr: "source_character_ids must not contain empty values",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateDamageAppliedPayload(mustJSONRawMessage(t, tt.payload))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateConditionSetPayload(t *testing.T) {
	t.Parallel()

	hidden := mustStandardConditionState(t, rules.ConditionHidden)
	restrained := mustStandardConditionState(t, rules.ConditionRestrained)

	tests := []struct {
		name    string
		before  []rules.ConditionState
		after   []rules.ConditionState
		added   []rules.ConditionState
		removed []rules.ConditionState
		wantErr string
	}{
		{
			name:   "accepts matching diff",
			before: []rules.ConditionState{hidden},
			after:  []rules.ConditionState{hidden, restrained},
			added:  []rules.ConditionState{restrained},
		},
		{
			name:    "rejects removed without before",
			after:   []rules.ConditionState{hidden},
			removed: []rules.ConditionState{restrained},
			wantErr: "conditions_before is required when removed are provided",
		},
		{
			name:    "rejects mismatched added diff",
			before:  []rules.ConditionState{hidden},
			after:   []rules.ConditionState{hidden, restrained},
			added:   []rules.ConditionState{hidden},
			wantErr: "added must match conditions_before and conditions_after diff",
		},
		{
			name:    "rejects unchanged conditions",
			before:  []rules.ConditionState{hidden},
			after:   []rules.ConditionState{hidden},
			wantErr: "conditions must change",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateConditionSetPayload(tt.before, tt.after, tt.added, tt.removed)
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateCharacterStatePatchPayload(t *testing.T) {
	t.Parallel()

	hpBefore := 6
	hpAfter := 4

	tests := []struct {
		name    string
		payload payload.CharacterStatePatchPayload
		wantErr string
	}{
		{
			name: "accepts hp mutation",
			payload: payload.CharacterStatePatchPayload{
				CharacterID: ids.CharacterID("char-1"),
				HPBefore:    &hpBefore,
				HPAfter:     &hpAfter,
			},
		},
		{
			name: "rejects no mutation",
			payload: payload.CharacterStatePatchPayload{
				CharacterID: ids.CharacterID("char-1"),
			},
			wantErr: "character_state patch must change at least one field",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCharacterStatePatchPayload(mustJSONRawMessage(t, tt.payload))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateCountdownPayloads(t *testing.T) {
	t.Parallel()

	t.Run("scene create rejects invalid starting roll range", func(t *testing.T) {
		t.Parallel()

		err := ValidateSceneCountdownCreatePayload(mustJSONRawMessage(t, payload.SceneCountdownCreatePayload{
			CountdownID:       "count-1",
			Name:              "Clock",
			Tone:              rules.CountdownToneConsequence,
			AdvancementPolicy: "manual",
			StartingValue:     6,
			RemainingValue:    3,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
			StartingRoll: &payload.CountdownStartingRollPayload{
				Min:   2,
				Max:   6,
				Value: 1,
			},
		}))
		assertErrorContains(t, err, "starting_roll value is out of range")
	})

	t.Run("campaign advance accepts triggered updates without counter change", func(t *testing.T) {
		t.Parallel()

		err := ValidateCampaignCountdownAdvancePayload(mustJSONRawMessage(t, payload.CampaignCountdownAdvancePayload{
			CountdownID:     "count-1",
			BeforeRemaining: 2,
			AfterRemaining:  2,
			AdvancedBy:      1,
			StatusBefore:    rules.CountdownStatusActive,
			StatusAfter:     rules.CountdownStatusActive,
			Triggered:       true,
		}))
		assertErrorContains(t, err, "")
	})

	t.Run("campaign advance rejects missing state change when not triggered", func(t *testing.T) {
		t.Parallel()

		err := ValidateCampaignCountdownAdvancePayload(mustJSONRawMessage(t, payload.CampaignCountdownAdvancePayload{
			CountdownID:     "count-1",
			BeforeRemaining: 2,
			AfterRemaining:  2,
			AdvancedBy:      1,
			StatusBefore:    rules.CountdownStatusActive,
			StatusAfter:     rules.CountdownStatusActive,
		}))
		assertErrorContains(t, err, "countdown advance must record a state change")
	})
}

func mustJSONRawMessage(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	return data
}

func mustStandardConditionState(t *testing.T, code string) rules.ConditionState {
	t.Helper()

	state, err := rules.StandardConditionState(code)
	if err != nil {
		t.Fatalf("rules.StandardConditionState(%q): %v", code, err)
	}
	return state
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if want == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}
