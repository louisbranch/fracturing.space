package damagetransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func mitigationDecisionFromProto(value *pb.DaggerheartDamageMitigationDecision) (rules.BaseArmorDecision, *pb.DaggerheartDamageArmorReaction) {
	if value == nil {
		return rules.BaseArmorDecisionAuto, nil
	}
	switch value.GetBaseArmor() {
	case pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_SPEND:
		return rules.BaseArmorDecisionSpend, value.GetArmorReaction()
	case pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_DECLINE:
		return rules.BaseArmorDecisionDecline, value.GetArmorReaction()
	default:
		return rules.BaseArmorDecisionAuto, value.GetArmorReaction()
	}
}

func mitigationDecisionIsExplicit(value *pb.DaggerheartDamageMitigationDecision, legacy *pb.DaggerheartDamageArmorReaction) bool {
	if legacy != nil {
		return true
	}
	if value == nil {
		return false
	}
	if value.GetBaseArmor() != pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_UNSPECIFIED {
		return true
	}
	return value.GetArmorReaction() != nil
}

func damageChoiceRequired(
	characterID string,
	reason string,
	optionCodes []string,
	decline rules.DamageApplication,
	spend rules.DamageApplication,
) *pb.DaggerheartCombatChoiceRequired {
	choice := &pb.DaggerheartCombatChoiceRequired{
		Stage:       pb.DaggerheartCombatChoiceStage_DAGGERHEART_COMBAT_CHOICE_STAGE_DAMAGE_MITIGATION,
		CharacterId: strings.TrimSpace(characterID),
		OptionCodes: optionCodes,
		Reason:      strings.TrimSpace(reason),
	}
	if preview := damagePreviewFromApplication(decline); preview != nil {
		choice.DeclinePreview = preview
	}
	if preview := damagePreviewFromApplication(spend); preview != nil {
		choice.SpendBaseArmorPreview = preview
	}
	return choice
}

func damagePreviewFromApplication(app rules.DamageApplication) *pb.DaggerheartDamagePreview {
	return &pb.DaggerheartDamagePreview{
		Severity:     DamageSeverityString(app.Result.Severity),
		Marks:        int32(app.Result.Marks),
		HpBefore:     int32(app.HPBefore),
		HpAfter:      int32(app.HPAfter),
		StressBefore: int32(app.StressBefore),
		StressAfter:  int32(app.StressAfter),
		ArmorBefore:  int32(app.ArmorBefore),
		ArmorAfter:   int32(app.ArmorAfter),
		ArmorSpent:   int32(app.ArmorSpent),
	}
}

func baseArmorChoiceIsMeaningful(decline, spend rules.DamageApplication) bool {
	return decline.HPAfter != spend.HPAfter ||
		decline.StressAfter != spend.StressAfter ||
		decline.ArmorAfter != spend.ArmorAfter ||
		decline.Result.Marks != spend.Result.Marks ||
		decline.Result.Severity != spend.Result.Severity
}

func availableMitigationOptionCodes(
	req *pb.DaggerheartDamageRequest,
	profile projectionstore.DaggerheartCharacterProfile,
	state projectionstore.DaggerheartCharacterState,
	armor *contentstore.DaggerheartArmor,
	decline rules.DamageApplication,
	spend rules.DamageApplication,
) []string {
	options := make([]string, 0, 4)
	if baseArmorChoiceIsMeaningful(decline, spend) {
		options = append(options, "armor.base_slot", "armor.decline")
	}
	if armor == nil {
		return options
	}
	armorRules := rules.EffectiveArmorRules(armor)
	if armorRules.ResilientDieSides > 0 &&
		!req.GetDirect() &&
		rules.IsLastBaseArmorSlot(state, profile.ArmorMax) &&
		spend.ArmorSpent > 0 {
		options = append(options, "armor.resilient")
	}
	if armorRules.ImpenetrableUsesPerShortRest > 0 &&
		armorRules.ImpenetrableStressCost > 0 &&
		!state.ImpenetrableUsedThisShortRest &&
		spend.HPBefore == 1 &&
		spend.HPAfter == 0 &&
		state.Stress+armorRules.ImpenetrableStressCost <= profile.StressMax {
		options = append(options, "armor.impenetrable")
	}
	return options
}
