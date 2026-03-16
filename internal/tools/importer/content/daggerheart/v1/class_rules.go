package catalogimporter

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func toStorageClassFeatures(features []featureRecord) []contentstore.DaggerheartFeature {
	items := make([]contentstore.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, contentstore.DaggerheartFeature{
			ID:               feature.ID,
			Name:             feature.Name,
			Description:      feature.Description,
			Level:            feature.Level,
			AutomationStatus: classFeatureAutomationStatus(feature),
			ClassRule:        deriveClassFeatureRule(feature),
		})
	}
	return items
}

func toStorageClassHopeFeature(classID string, feature hopeFeatureRecord) contentstore.DaggerheartHopeFeature {
	rule := deriveHopeFeatureRule(classID, feature)
	status := contentstore.DaggerheartFeatureAutomationStatusUnsupported
	if rule != nil {
		status = contentstore.DaggerheartFeatureAutomationStatusSupported
	}
	return contentstore.DaggerheartHopeFeature{
		Name:             feature.Name,
		Description:      feature.Description,
		HopeCost:         feature.HopeCost,
		AutomationStatus: status,
		HopeFeatureRule:  rule,
	}
}

func classFeatureAutomationStatus(feature featureRecord) contentstore.DaggerheartFeatureAutomationStatus {
	if deriveClassFeatureRule(feature) != nil {
		return contentstore.DaggerheartFeatureAutomationStatusSupported
	}
	return contentstore.DaggerheartFeatureAutomationStatusUnsupported
}

func deriveClassFeatureRule(feature featureRecord) *contentstore.DaggerheartClassFeatureRule {
	switch strings.TrimSpace(feature.ID) {
	case "feature.guardian-unstoppable":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:     contentstore.DaggerheartClassFeatureRuleKindUnstoppable,
			DieSides: 4,
		}
	case "feature.bard-rally":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:     contentstore.DaggerheartClassFeatureRuleKindRally,
			DieSides: 6,
		}
	case "feature.druid-beastform":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:       contentstore.DaggerheartClassFeatureRuleKindBeastform,
			StressCost: 1,
		}
	case "feature.ranger-focus":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:     contentstore.DaggerheartClassFeatureRuleKindHuntersFocus,
			HopeCost: 1,
		}
	case "feature.rogue-cloaked":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind: contentstore.DaggerheartClassFeatureRuleKindCloaked,
		}
	case "feature.rogue-sneak-attack":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:              contentstore.DaggerheartClassFeatureRuleKindSneakAttack,
			DieSides:          6,
			UseCharacterLevel: true,
		}
	case "feature.seraph-prayer-dice":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:     contentstore.DaggerheartClassFeatureRuleKindPrayerDice,
			DieSides: 4,
		}
	case "feature.sorcerer-channel-raw-power":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind: contentstore.DaggerheartClassFeatureRuleKindChannelRawPower,
		}
	case "feature.warrior-attack-of-opportunity":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind: contentstore.DaggerheartClassFeatureRuleKindPartingStrike,
		}
	case "feature.warrior-combat-training":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind:              contentstore.DaggerheartClassFeatureRuleKindCombatTraining,
			UseCharacterLevel: true,
		}
	case "feature.wizard-strange-patterns":
		return &contentstore.DaggerheartClassFeatureRule{
			Kind: contentstore.DaggerheartClassFeatureRuleKindStrangePatterns,
		}
	default:
		return nil
	}
}

func deriveHopeFeatureRule(classID string, feature hopeFeatureRecord) *contentstore.DaggerheartHopeFeatureRule {
	kind, ok := hopeFeatureKindByClassID[strings.TrimSpace(classID)]
	if !ok {
		return nil
	}
	return &contentstore.DaggerheartHopeFeatureRule{
		Kind:     kind,
		Bonus:    hopeFeatureBonus(kind),
		HopeCost: feature.HopeCost,
	}
}

var hopeFeatureKindByClassID = map[string]contentstore.DaggerheartHopeFeatureRuleKind{
	"class.guardian": contentstore.DaggerheartHopeFeatureRuleKindFrontlineTank,
	"class.bard":     contentstore.DaggerheartHopeFeatureRuleKindMakeAScene,
	"class.druid":    contentstore.DaggerheartHopeFeatureRuleKindEvolution,
	"class.ranger":   contentstore.DaggerheartHopeFeatureRuleKindHoldThemOff,
	"class.rogue":    contentstore.DaggerheartHopeFeatureRuleKindRoguesDodge,
	"class.seraph":   contentstore.DaggerheartHopeFeatureRuleKindLifeSupport,
	"class.sorcerer": contentstore.DaggerheartHopeFeatureRuleKindVolatileMagic,
	"class.warrior":  contentstore.DaggerheartHopeFeatureRuleKindNoMercy,
	"class.wizard":   contentstore.DaggerheartHopeFeatureRuleKindNotThisTime,
}

func hopeFeatureBonus(kind contentstore.DaggerheartHopeFeatureRuleKind) int {
	switch kind {
	case contentstore.DaggerheartHopeFeatureRuleKindFrontlineTank:
		return 2
	case contentstore.DaggerheartHopeFeatureRuleKindMakeAScene:
		return -2
	case contentstore.DaggerheartHopeFeatureRuleKindLifeSupport:
		return 1
	case contentstore.DaggerheartHopeFeatureRuleKindRoguesDodge:
		return 2
	case contentstore.DaggerheartHopeFeatureRuleKindNoMercy:
		return 1
	default:
		return 0
	}
}

func activeClassFeatures(class contentstore.DaggerheartClass) []contentstore.DaggerheartFeature {
	features := append([]contentstore.DaggerheartFeature(nil), class.Features...)
	if strings.TrimSpace(class.ID) == "" {
		return features
	}
	if strings.TrimSpace(class.HopeFeature.Name) == "" {
		return features
	}
	features = append(features, contentstore.DaggerheartFeature{
		ID:               fmt.Sprintf("hope_feature:%s", class.ID),
		Name:             class.HopeFeature.Name,
		Description:      class.HopeFeature.Description,
		Level:            1,
		AutomationStatus: class.HopeFeature.AutomationStatus,
	})
	return features
}
