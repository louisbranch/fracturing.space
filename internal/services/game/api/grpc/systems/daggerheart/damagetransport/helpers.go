package damagetransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// ResolveCharacterDamage applies a Daggerheart damage request to one character
// projection snapshot.
func ResolveCharacterDamage(req *pb.DaggerheartDamageRequest, profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState) (daggerheart.DamageApplication, bool, error) {
	return daggerheart.ResolveDamageApplication(
		daggerheart.DamageTarget{
			HP:              state.Hp,
			Armor:           state.Armor,
			MajorThreshold:  profile.MajorThreshold,
			SevereThreshold: profile.SevereThreshold,
		},
		damageApplyInputFromProto(req),
	)
}

// ResolveAdversaryDamage applies a Daggerheart damage request to one adversary
// projection snapshot.
func ResolveAdversaryDamage(req *pb.DaggerheartDamageRequest, adversary projectionstore.DaggerheartAdversary) (daggerheart.DamageApplication, bool, error) {
	return daggerheart.ResolveDamageApplication(
		daggerheart.DamageTarget{
			HP:              adversary.HP,
			Armor:           adversary.Armor,
			MajorThreshold:  adversary.Major,
			SevereThreshold: adversary.Severe,
		},
		damageApplyInputFromProto(req),
	)
}

// DamageSeverityString maps a Daggerheart domain severity into the stable
// payload label used by transport and events.
func DamageSeverityString(severity daggerheart.DamageSeverity) string {
	switch severity {
	case daggerheart.DamageMinor:
		return "minor"
	case daggerheart.DamageMajor:
		return "major"
	case daggerheart.DamageSevere:
		return "severe"
	case daggerheart.DamageMassive:
		return "massive"
	default:
		return "none"
	}
}

// DamageTypeString maps the protobuf damage type into the stable payload label
// used by transport and events.
func DamageTypeString(t pb.DaggerheartDamageType) string {
	switch t {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		return "physical"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return "magic"
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return "mixed"
	default:
		return "unknown"
	}
}

func damageApplyInputFromProto(req *pb.DaggerheartDamageRequest) daggerheart.DamageApplyInput {
	if req == nil {
		return daggerheart.DamageApplyInput{}
	}
	return daggerheart.DamageApplyInput{
		Amount:       int(req.GetAmount()),
		Types:        damageTypesFromProto(req.GetDamageType()),
		Resistance:   resistanceFromProto(req),
		Direct:       req.GetDirect(),
		AllowMassive: req.GetMassiveDamage(),
	}
}

func resistanceFromProto(req *pb.DaggerheartDamageRequest) daggerheart.ResistanceProfile {
	if req == nil {
		return daggerheart.ResistanceProfile{}
	}
	return daggerheart.ResistanceProfile{
		ResistPhysical: req.GetResistPhysical(),
		ResistMagic:    req.GetResistMagic(),
		ImmunePhysical: req.GetImmunePhysical(),
		ImmuneMagic:    req.GetImmuneMagic(),
	}
}

func damageTypesFromProto(damageType pb.DaggerheartDamageType) daggerheart.DamageTypes {
	damageTypes := daggerheart.DamageTypes{}
	switch damageType {
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL:
		damageTypes.Physical = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		damageTypes.Magic = true
	case pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		damageTypes.Physical = true
		damageTypes.Magic = true
	}
	return damageTypes
}

func containsString(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringsToCharacterIDs(values []string) []ids.CharacterID {
	if len(values) == 0 {
		return nil
	}
	result := make([]ids.CharacterID, len(values))
	for i, value := range values {
		result[i] = ids.CharacterID(value)
	}
	return result
}
