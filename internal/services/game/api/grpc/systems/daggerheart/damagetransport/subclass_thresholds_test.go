package damagetransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestResolveCharacterDamageAppliesSubclassThresholdBonuses(t *testing.T) {
	t.Run("earth channel raises both thresholds by proficiency", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{MajorThreshold: 5, SevereThreshold: 10, Proficiency: 2}
		state := projectionstore.DaggerheartCharacterState{
			Hp: 10,
			SubclassState: &projectionstore.DaggerheartSubclassState{
				ElementalChannel: daggerheart.ElementalChannelEarth,
			},
		}
		result, _, err := ResolveCharacterDamage(&pb.DaggerheartDamageRequest{
			Amount:     6,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		}, profile, state, nil)
		if err != nil {
			t.Fatalf("ResolveCharacterDamage returned error: %v", err)
		}
		if result.Result.Severity != daggerheart.DamageMinor {
			t.Fatalf("severity = %v, want minor with earth threshold bonus", result.Result.Severity)
		}
	})

	t.Run("transcendence raises severe threshold", func(t *testing.T) {
		profile := projectionstore.DaggerheartCharacterProfile{MajorThreshold: 5, SevereThreshold: 10}
		state := projectionstore.DaggerheartCharacterState{
			Hp: 10,
			SubclassState: &projectionstore.DaggerheartSubclassState{
				TranscendenceSevereThresholdBonus: 4,
			},
		}
		result, _, err := ResolveCharacterDamage(&pb.DaggerheartDamageRequest{
			Amount:     12,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		}, profile, state, nil)
		if err != nil {
			t.Fatalf("ResolveCharacterDamage returned error: %v", err)
		}
		if result.Result.Severity != daggerheart.DamageMajor {
			t.Fatalf("severity = %v, want major below boosted severe threshold", result.Result.Severity)
		}
	})
}
