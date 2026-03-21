package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/damagetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func mustDamageAppliedPayloadJSON(t *testing.T, characterID string, damage *pb.DaggerheartDamageRequest, profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState) []byte {
	t.Helper()

	result, mitigated, err := damagetransport.ResolveCharacterDamage(damage, profile, state, nil)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpAfter := result.HPAfter
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := workflowtransport.NormalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheartpayload.DamageAppliedPayload{
		CharacterID:        ids.CharacterID(characterID),
		Hp:                 &hpAfter,
		Armor:              &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           damagetransport.DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         damagetransport.DamageTypeString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: testStringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}
	return payloadJSON
}

func mustConditionChangedJSON(t *testing.T, characterID string, conditions []string, added []string) []byte {
	t.Helper()

	conditionStates := make([]rules.ConditionState, 0, len(conditions))
	for _, code := range conditions {
		state, err := rules.StandardConditionState(code)
		if err != nil {
			t.Fatalf("standard condition state %q: %v", code, err)
		}
		conditionStates = append(conditionStates, state)
	}
	addedStates := make([]rules.ConditionState, 0, len(added))
	for _, code := range added {
		state, err := rules.StandardConditionState(code)
		if err != nil {
			t.Fatalf("standard added condition state %q: %v", code, err)
		}
		addedStates = append(addedStates, state)
	}

	payload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: ids.CharacterID(characterID),
		Conditions:  conditionStates,
		Added:       addedStates,
	}
	conditionJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	return conditionJSON
}

func mustCharacterStatePatchedJSON(t *testing.T, characterID string, lifeState string) []byte {
	t.Helper()

	payload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: ids.CharacterID(characterID),
		LifeState:   &lifeState,
	}
	patchJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	return patchJSON
}
