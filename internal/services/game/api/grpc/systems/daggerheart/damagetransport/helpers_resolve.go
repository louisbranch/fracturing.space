package damagetransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// ResolveCharacterDamage applies a Daggerheart damage request to one character
// projection snapshot.
func ResolveCharacterDamage(req *pb.DaggerheartDamageRequest, profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState, armor *contentstore.DaggerheartArmor) (rules.DamageApplication, bool, error) {
	target := rules.DamageTarget{
		HP:              state.Hp,
		Stress:          state.Stress,
		Armor:           state.Armor,
		MajorThreshold:  profile.MajorThreshold,
		SevereThreshold: profile.SevereThreshold,
	}
	if state.SubclassState != nil {
		target.SevereThreshold += state.SubclassState.TranscendenceSevereThresholdBonus
		if strings.EqualFold(strings.TrimSpace(state.SubclassState.ElementalChannel), daggerheartstate.ElementalChannelEarth) && profile.Proficiency > 0 {
			target.MajorThreshold += profile.Proficiency
			target.SevereThreshold += profile.Proficiency
		}
	}
	for _, mod := range state.StatModifiers {
		switch mod.Target {
		case "major_threshold":
			target.MajorThreshold += mod.Delta
		case "severe_threshold":
			target.SevereThreshold += mod.Delta
		}
	}
	if armor != nil {
		armorRules := rules.EffectiveArmorRules(armor)
		baseArmor := rules.CurrentBaseArmor(state, profile.ArmorMax)
		if armorRules.ThresholdBonusWhenArmorDepleted > 0 && baseArmor == 0 && profile.ArmorMax > 0 {
			target.MajorThreshold += armorRules.ThresholdBonusWhenArmorDepleted
			target.SevereThreshold += armorRules.ThresholdBonusWhenArmorDepleted
		}
		target.ArmorRules = rules.ArmorDamageRules{
			MitigationMode:                  string(armorRules.MitigationMode),
			SeverityReductionSteps:          armorRules.SeverityReductionSteps,
			StressOnMark:                    armorRules.StressOnMark,
			WardedMagicReduction:            armorRules.WardedMagicReduction,
			WardedReductionAmount:           armor.ArmorScore,
			ThresholdBonusWhenArmorDepleted: armorRules.ThresholdBonusWhenArmorDepleted,
		}
	}
	return rules.ResolveDamageApplication(target, damageApplyInputFromProto(req))
}

// ResolveAdversaryDamage applies a Daggerheart damage request to one adversary
// projection snapshot.
func ResolveAdversaryDamage(req *pb.DaggerheartDamageRequest, adversary projectionstore.DaggerheartAdversary) (rules.DamageApplication, bool, error) {
	return rules.ResolveDamageApplication(
		rules.DamageTarget{
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
func DamageSeverityString(severity rules.DamageSeverity) string {
	switch severity {
	case rules.DamageMinor:
		return "minor"
	case rules.DamageMajor:
		return "major"
	case rules.DamageSevere:
		return "severe"
	case rules.DamageMassive:
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

func applyCharacterDamageResult(current projectionstore.DaggerheartCharacterState, result rules.DamageApplication) projectionstore.DaggerheartCharacterState {
	current.Hp = result.HPAfter
	current.Stress = result.StressAfter
	current.Armor = result.ArmorAfter
	return current
}

func applyAdversaryDamageResult(current projectionstore.DaggerheartAdversary, result rules.DamageApplication) projectionstore.DaggerheartAdversary {
	current.HP = result.HPAfter
	current.Armor = result.ArmorAfter
	return current
}
