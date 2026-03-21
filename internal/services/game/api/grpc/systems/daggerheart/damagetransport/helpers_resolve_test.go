package damagetransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestDamageSeverityString(t *testing.T) {
	tests := []struct {
		severity daggerheart.DamageSeverity
		want     string
	}{
		{daggerheart.DamageMinor, "minor"},
		{daggerheart.DamageMajor, "major"},
		{daggerheart.DamageSevere, "severe"},
		{daggerheart.DamageMassive, "massive"},
		{daggerheart.DamageSeverity(99), "none"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := DamageSeverityString(tc.severity); got != tc.want {
				t.Fatalf("DamageSeverityString(%v) = %q, want %q", tc.severity, got, tc.want)
			}
		})
	}
}

func TestDamageTypeString(t *testing.T) {
	tests := []struct {
		damageType pb.DaggerheartDamageType
		want       string
	}{
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL, "physical"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC, "magic"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED, "mixed"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED, "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := DamageTypeString(tc.damageType); got != tc.want {
				t.Fatalf("DamageTypeString(%v) = %q, want %q", tc.damageType, got, tc.want)
			}
		})
	}
}

func TestResolveCharacterDamageMitigatedToZero(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{MajorThreshold: 5, SevereThreshold: 10}
	state := projectionstore.DaggerheartCharacterState{Hp: 10, Armor: 1}

	result, mitigated, err := ResolveCharacterDamage(&pb.DaggerheartDamageRequest{
		Amount:     1,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}, profile, state, nil)
	if err != nil {
		t.Fatalf("ResolveCharacterDamage returned error: %v", err)
	}
	if !mitigated {
		t.Fatal("expected mitigated damage")
	}
	if result.HPAfter != state.Hp {
		t.Fatalf("hp_after = %d, want %d", result.HPAfter, state.Hp)
	}
}

func TestResolveAdversaryDamageDirect(t *testing.T) {
	adversary := projectionstore.DaggerheartAdversary{HP: 10, Armor: 5, Major: 5, Severe: 8}
	result, _, err := ResolveAdversaryDamage(&pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		Direct:     true,
	}, adversary)
	if err != nil {
		t.Fatalf("ResolveAdversaryDamage returned error: %v", err)
	}
	if result.ArmorSpent != 0 {
		t.Fatalf("armor_spent = %d, want 0", result.ArmorSpent)
	}
	if result.HPAfter != 9 {
		t.Fatalf("hp_after = %d, want 9", result.HPAfter)
	}
}
