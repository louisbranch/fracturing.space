package render

import "strings"

// campaignCreationStepLabel maps workflow step keys to localized reader-facing copy.
func campaignCreationStepLabel(loc Localizer, stepKey string) string {
	switch strings.TrimSpace(stepKey) {
	case "class_subclass":
		return T(loc, "game.character_creation.step.class_subclass")
	case "heritage":
		return T(loc, "game.character_creation.step.heritage")
	case "traits":
		return T(loc, "game.character_creation.step.traits")
	case "details":
		return T(loc, "game.character_creation.step.details")
	case "equipment":
		return T(loc, "game.character_creation.step.equipment")
	case "background":
		return T(loc, "game.character_creation.step.background")
	case "experiences":
		return T(loc, "game.character_creation.step.experiences")
	case "domain_cards":
		return T(loc, "game.character_creation.step.domain_cards")
	case "connections":
		return T(loc, "game.character_creation.step.connections")
	default:
		return strings.TrimSpace(stepKey)
	}
}

// campaignCreationOptionSelected keeps single-select card state comparisons consistent.
func campaignCreationOptionSelected(value string, selected string) bool {
	return strings.TrimSpace(value) != "" && strings.TrimSpace(value) == strings.TrimSpace(selected)
}

// campaignCreationOptionInSet keeps multi-select card state comparisons consistent.
func campaignCreationOptionInSet(value string, selected []string) bool {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return false
	}
	for _, selectedValue := range selected {
		if trimmedValue == strings.TrimSpace(selectedValue) {
			return true
		}
	}
	return false
}

// campaignCreationNumericValue normalizes empty numeric inputs to the render fallback.
func campaignCreationNumericValue(raw string) string {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return "0"
	}
	return trimmedRaw
}

// campaignCreationIsStep marks the current in-progress workflow step.
func campaignCreationIsStep(view CampaignCharacterCreationView, step int32) bool {
	return !view.Ready && view.NextStep == step
}

// campaignCreationUnmetReason localizes workflow unmet reasons when they are message keys.
func campaignCreationUnmetReason(loc Localizer, reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	if strings.HasPrefix(reason, "game.") || strings.HasPrefix(reason, "error.") {
		return T(loc, reason)
	}
	return reason
}
