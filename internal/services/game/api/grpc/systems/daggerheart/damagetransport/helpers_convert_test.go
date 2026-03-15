package damagetransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestDamageApplyInputFromProtoNil(t *testing.T) {
	got := damageApplyInputFromProto(nil)
	if got.Amount != 0 {
		t.Fatalf("amount = %d, want 0", got.Amount)
	}
	if got.Direct {
		t.Fatal("direct = true, want false")
	}
	if got.AllowMassive {
		t.Fatal("allow_massive = true, want false")
	}
}

func TestDamageApplyInputFromProtoMixedDamage(t *testing.T) {
	got := damageApplyInputFromProto(&pb.DaggerheartDamageRequest{
		Amount:         4,
		DamageType:     pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED,
		ResistPhysical: true,
		ImmuneMagic:    true,
		Direct:         true,
		MassiveDamage:  true,
	})
	if got.Amount != 4 {
		t.Fatalf("amount = %d, want 4", got.Amount)
	}
	if !got.Types.Physical || !got.Types.Magic {
		t.Fatalf("types = %+v, want mixed damage", got.Types)
	}
	if !got.Resistance.ResistPhysical || !got.Resistance.ImmuneMagic {
		t.Fatalf("resistance = %+v, want physical resist and magic immunity", got.Resistance)
	}
	if !got.Direct {
		t.Fatal("direct = false, want true")
	}
	if !got.AllowMassive {
		t.Fatal("allow_massive = false, want true")
	}
}
