package damagetransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// ResolveCharacterDamage applies a Daggerheart damage request to one character
// projection snapshot.
func ResolveCharacterDamage(req *pb.DaggerheartDamageRequest, profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState, armor *contentstore.DaggerheartArmor) (daggerheart.DamageApplication, bool, error) {
	target := daggerheart.DamageTarget{
		HP:              state.Hp,
		Stress:          state.Stress,
		Armor:           state.Armor,
		MajorThreshold:  profile.MajorThreshold,
		SevereThreshold: profile.SevereThreshold,
	}
	if state.SubclassState != nil {
		target.SevereThreshold += state.SubclassState.TranscendenceSevereThresholdBonus
		if strings.EqualFold(strings.TrimSpace(state.SubclassState.ElementalChannel), daggerheart.ElementalChannelEarth) && profile.Proficiency > 0 {
			target.MajorThreshold += profile.Proficiency
			target.SevereThreshold += profile.Proficiency
		}
	}
	if armor != nil {
		rules := daggerheart.EffectiveArmorRules(armor)
		baseArmor := daggerheart.CurrentBaseArmor(state, profile.ArmorMax)
		if rules.ThresholdBonusWhenArmorDepleted > 0 && baseArmor == 0 && profile.ArmorMax > 0 {
			target.MajorThreshold += rules.ThresholdBonusWhenArmorDepleted
			target.SevereThreshold += rules.ThresholdBonusWhenArmorDepleted
		}
		target.ArmorRules = daggerheart.ArmorDamageRules{
			MitigationMode:                  string(rules.MitigationMode),
			SeverityReductionSteps:          rules.SeverityReductionSteps,
			StressOnMark:                    rules.StressOnMark,
			WardedMagicReduction:            rules.WardedMagicReduction,
			WardedReductionAmount:           armor.ArmorScore,
			ThresholdBonusWhenArmorDepleted: rules.ThresholdBonusWhenArmorDepleted,
		}
	}
	return daggerheart.ResolveDamageApplication(target, damageApplyInputFromProto(req))
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

func applyCharacterDamageResult(current projectionstore.DaggerheartCharacterState, result daggerheart.DamageApplication) projectionstore.DaggerheartCharacterState {
	current.Hp = result.HPAfter
	current.Stress = result.StressAfter
	current.Armor = result.ArmorAfter
	return current
}

func applyAdversaryDamageResult(current projectionstore.DaggerheartAdversary, result daggerheart.DamageApplication) projectionstore.DaggerheartAdversary {
	current.HP = result.HPAfter
	current.Armor = result.ArmorAfter
	return current
}
