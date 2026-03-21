package catalogimporter

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func toStorageSubclassFeatures(features []featureRecord) []contentstore.DaggerheartFeature {
	items := make([]contentstore.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, contentstore.DaggerheartFeature{
			ID:               feature.ID,
			Name:             feature.Name,
			Description:      feature.Description,
			Level:            feature.Level,
			AutomationStatus: subclassFeatureAutomationStatus(feature),
			SubclassRule:     deriveSubclassFeatureRule(feature),
		})
	}
	return items
}

func subclassFeatureAutomationStatus(feature featureRecord) contentstore.DaggerheartFeatureAutomationStatus {
	if deriveSubclassFeatureRule(feature) != nil {
		return contentstore.DaggerheartFeatureAutomationStatusSupported
	}
	return contentstore.DaggerheartFeatureAutomationStatusUnsupported
}

func deriveSubclassFeatureRule(feature featureRecord) *contentstore.DaggerheartSubclassFeatureRule {
	switch strings.TrimSpace(feature.ID) {
	case "feature.stalwart-unwavering":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
			Bonus:          1,
			ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeAll,
		}
	case "feature.stalwart-unrelenting":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
			Bonus:          2,
			ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeAll,
		}
	case "feature.stalwart-undaunted":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
			Bonus:          3,
			ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeAll,
		}
	case "feature.vengeance-at-ease":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:  contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus,
			Bonus: 1,
		}
	case "feature.school-war-battlemage":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:  contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus,
			Bonus: 1,
		}
	case "feature.school-war-face-your-fear":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
			DamageDiceCount: 1,
			DamageDieSides:  10,
		}
	case "feature.school-war-conjure-shield":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast,
			Bonus:           1,
			RequiredHopeMin: 2,
		}
	case "feature.school-war-fueled-by-fear":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
			DamageDiceCount: 2,
			DamageDieSides:  10,
		}
	case "feature.school-war-have-no-fear":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
			DamageDiceCount: 3,
			DamageDieSides:  10,
		}
	case "feature.nightwalker-adrenaline":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:              contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable,
			UseCharacterLevel: true,
		}
	case "feature.nightwalker-fleeting-shadow":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:  contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonus,
			Bonus: 1,
		}
	case "feature.call-brave-courage":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:  contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear,
			Bonus: 1,
		}
	case "feature.winged-sentinel-ascendant":
		return &contentstore.DaggerheartSubclassFeatureRule{
			Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
			Bonus:          4,
			ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeSevereOnly,
		}
	default:
		return nil
	}
}
