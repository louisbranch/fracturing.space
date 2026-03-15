package damagetransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	}, profile, state)
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

func TestCampaignSupportsDaggerheart(t *testing.T) {
	if !campaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}) {
		t.Fatal("expected daggerheart campaign to be supported")
	}
	if campaignSupportsDaggerheart(storage.CampaignRecord{System: systembridge.SystemID("not-a-system")}) {
		t.Fatal("unexpected support for non-daggerheart campaign")
	}
}

func TestRequireDaggerheartSystem(t *testing.T) {
	if err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}, "unsupported"); err != nil {
		t.Fatalf("requireDaggerheartSystem returned error for daggerheart: %v", err)
	}
	err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

type gateStoreStub struct {
	gate storage.SessionGate
	err  error
}

func (s gateStoreStub) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return s.gate, s.err
}

func TestEnsureNoOpenSessionGate(t *testing.T) {
	err := ensureNoOpenSessionGate(context.Background(), gateStoreStub{err: storage.ErrNotFound}, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("ensureNoOpenSessionGate returned error for missing gate: %v", err)
	}

	err = ensureNoOpenSessionGate(context.Background(), gateStoreStub{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	err = ensureNoOpenSessionGate(context.Background(), gateStoreStub{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestContainsString(t *testing.T) {
	if containsString([]string{"a", "b"}, "c") {
		t.Fatal("containsString reported missing value as present")
	}
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatal("containsString did not find expected value")
	}
	if containsString([]string{"a"}, "") {
		t.Fatal("containsString matched empty target")
	}
}

func TestStringsToCharacterIDs(t *testing.T) {
	if got := stringsToCharacterIDs(nil); got != nil {
		t.Fatalf("stringsToCharacterIDs(nil) = %v, want nil", got)
	}
	got := stringsToCharacterIDs([]string{"char-1", "char-2"})
	if len(got) != 2 || string(got[0]) != "char-1" || string(got[1]) != "char-2" {
		t.Fatalf("stringsToCharacterIDs = %v, want [char-1 char-2]", got)
	}
}
