package damagetransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

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
