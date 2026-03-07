package daggerheart

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func applyDaggerheartDamage(req *pb.DaggerheartDamageRequest, profile storage.DaggerheartCharacterProfile, state storage.DaggerheartCharacterState) (daggerheart.DamageApplication, bool, error) {
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

func applyDaggerheartAdversaryDamage(req *pb.DaggerheartDamageRequest, adversary storage.DaggerheartAdversary) (daggerheart.DamageApplication, bool, error) {
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

func daggerheartSeverityToString(severity daggerheart.DamageSeverity) string {
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

func daggerheartDamageTypeToString(t pb.DaggerheartDamageType) string {
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

func daggerheartStateToProto(state storage.DaggerheartCharacterState) *pb.DaggerheartCharacterState {
	temporaryArmorBuckets := make([]*pb.DaggerheartTemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		temporaryArmorBuckets = append(temporaryArmorBuckets, &pb.DaggerheartTemporaryArmorBucket{
			Source:   bucket.Source,
			Duration: bucket.Duration,
			SourceId: bucket.SourceID,
			Amount:   int32(bucket.Amount),
		})
	}

	return &pb.DaggerheartCharacterState{
		Hp:                    int32(state.Hp),
		Hope:                  int32(state.Hope),
		HopeMax:               int32(state.HopeMax),
		Stress:                int32(state.Stress),
		Armor:                 int32(state.Armor),
		Conditions:            daggerheartConditionsToProto(state.Conditions),
		TemporaryArmorBuckets: temporaryArmorBuckets,
		LifeState:             daggerheartLifeStateToProto(state.LifeState),
	}
}

func optionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}
