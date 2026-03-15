package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/damagetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func mustDamageAppliedPayloadJSON(t *testing.T, characterID string, damage *pb.DaggerheartDamageRequest, profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState) []byte {
	t.Helper()

	result, mitigated, err := damagetransport.ResolveCharacterDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpAfter := result.HPAfter
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := workflowtransport.NormalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
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

	payload := daggerheart.ConditionChangedPayload{
		CharacterID: ids.CharacterID(characterID),
		Conditions:  conditions,
		Added:       added,
	}
	conditionJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	return conditionJSON
}

func mustCharacterStatePatchedJSON(t *testing.T, characterID string, lifeState string) []byte {
	t.Helper()

	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: ids.CharacterID(characterID),
		LifeState:   &lifeState,
	}
	patchJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	return patchJSON
}
