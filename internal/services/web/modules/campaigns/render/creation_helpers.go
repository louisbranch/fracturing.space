package render

import "strings"

const campaignCreationRequirementCompanionSheet = "companion_sheet_required"
const campaignCreationSubclassBeastboundID = "subclass.beastbound"

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

// campaignCreationRequiresCompanion reports whether a subclass or profile selection requires companion input.
func campaignCreationRequiresCompanion(requirements []string) bool {
	for _, requirement := range requirements {
		if strings.EqualFold(strings.TrimSpace(requirement), campaignCreationRequirementCompanionSheet) {
			return true
		}
	}
	return false
}

// campaignCreationSubclassRequiresCompanion keeps the UI aligned with current
// enforced subclass requirements even when a stale catalog entry drops the
// explicit requirement list before render.
func campaignCreationSubclassRequiresCompanion(subclassID string, requirements []string) bool {
	if campaignCreationRequiresCompanion(requirements) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(subclassID), campaignCreationSubclassBeastboundID)
}

// campaignCreationHeritageMode derives the current heritage selection mode.
func campaignCreationHeritageMode(view CampaignCharacterCreationView) string {
	firstID := strings.TrimSpace(view.Heritage.FirstFeatureAncestryID)
	secondID := strings.TrimSpace(view.Heritage.SecondFeatureAncestryID)
	if firstID != "" && secondID != "" && firstID != secondID {
		return "mixed"
	}
	return "single"
}

// campaignCreationHeritageDisplayName resolves the ancestry-facing heritage label.
func campaignCreationHeritageDisplayName(view CampaignCharacterCreationView) string {
	if campaignCreationHeritageMode(view) == "mixed" {
		if label := strings.TrimSpace(view.Heritage.AncestryLabel); label != "" {
			return label
		}
	}
	firstName := creationNameByID(view.Ancestries, view.Heritage.FirstFeatureAncestryID, func(h CampaignCreationHeritageView) string { return h.ID }, func(h CampaignCreationHeritageView) string { return h.Name })
	secondName := creationNameByID(view.Ancestries, view.Heritage.SecondFeatureAncestryID, func(h CampaignCreationHeritageView) string { return h.ID }, func(h CampaignCreationHeritageView) string { return h.Name })
	switch {
	case firstName == "":
		return ""
	case strings.TrimSpace(view.Heritage.SecondFeatureAncestryID) == "" || strings.TrimSpace(view.Heritage.SecondFeatureAncestryID) == strings.TrimSpace(view.Heritage.FirstFeatureAncestryID):
		return firstName
	case secondName == "" || secondName == firstName:
		return firstName
	default:
		return firstName + " / " + secondName
	}
}

// campaignCreationHeritageFeatureSummary returns a short granted-feature summary.
func campaignCreationHeritageFeatureSummary(view CampaignCharacterCreationView) string {
	names := []string{}
	for _, feature := range campaignCreationSelectedHeritageFeatures(view) {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return strings.Join(names, " + ")
}

// campaignCreationHeritageStepSummary returns the navigator summary text.
func campaignCreationHeritageStepSummary(view CampaignCharacterCreationView) string {
	heritage := campaignCreationHeritageDisplayName(view)
	community := creationNameByID(view.Communities, view.Heritage.CommunityID, func(h CampaignCreationHeritageView) string { return h.ID }, func(h CampaignCreationHeritageView) string { return h.Name })
	features := campaignCreationHeritageFeatureSummary(view)
	summary := strings.Trim(strings.Join([]string{heritage, community}, ", "), ", ")
	if summary == "" {
		return features
	}
	if features == "" {
		return summary
	}
	return summary + " · " + features
}

// campaignCreationSelectedHeritageFeatures returns the resolved ancestry features.
func campaignCreationSelectedHeritageFeatures(view CampaignCharacterCreationView) []CampaignCreationClassFeatureView {
	features := []CampaignCreationClassFeatureView{}
	if first, ok := campaignCreationSelectedAncestry(view, 0); ok {
		features = append(features, campaignCreationHeritageSlotFeatures(first, 0)...)
	}
	if second, ok := campaignCreationSelectedAncestry(view, 1); ok {
		features = append(features, campaignCreationHeritageSlotFeatures(second, 1)...)
	}
	return features
}

// campaignCreationSelectedAncestryName resolves one selected ancestry name.
func campaignCreationSelectedAncestryName(view CampaignCharacterCreationView, slotIndex int) string {
	ancestryID := strings.TrimSpace(view.Heritage.FirstFeatureAncestryID)
	if slotIndex == 1 {
		ancestryID = strings.TrimSpace(view.Heritage.SecondFeatureAncestryID)
	}
	return creationNameByID(view.Ancestries, ancestryID, func(h CampaignCreationHeritageView) string { return h.ID }, func(h CampaignCreationHeritageView) string { return h.Name })
}

// campaignCreationSelectedAncestryFeatureNames resolves one selected ancestry slot summary.
func campaignCreationSelectedAncestryFeatureNames(view CampaignCharacterCreationView, slotIndex int) string {
	ancestry, ok := campaignCreationSelectedAncestry(view, slotIndex)
	if !ok {
		return ""
	}
	if slotIndex == 0 && campaignCreationHeritageMode(view) == "single" {
		return campaignCreationFeatureNames(ancestry.Features)
	}
	return campaignCreationFeatureNames(campaignCreationHeritageSlotFeatures(ancestry, slotIndex))
}

// campaignCreationSelectedCommunityName resolves the chosen community name.
func campaignCreationSelectedCommunityName(view CampaignCharacterCreationView) string {
	return creationNameByID(view.Communities, view.Heritage.CommunityID, func(h CampaignCreationHeritageView) string { return h.ID }, func(h CampaignCreationHeritageView) string { return h.Name })
}

// campaignCreationSelectedCommunityFeatures returns the chosen community features.
func campaignCreationSelectedCommunityFeatures(view CampaignCharacterCreationView) []CampaignCreationClassFeatureView {
	community, ok := campaignCreationSelectedCommunity(view)
	if !ok {
		return nil
	}
	return append([]CampaignCreationClassFeatureView(nil), community.Features...)
}

// campaignCreationSelectedAncestry resolves the chosen ancestry for one slot.
func campaignCreationSelectedAncestry(view CampaignCharacterCreationView, slotIndex int) (CampaignCreationHeritageView, bool) {
	ancestryID := strings.TrimSpace(view.Heritage.FirstFeatureAncestryID)
	if slotIndex == 1 {
		ancestryID = strings.TrimSpace(view.Heritage.SecondFeatureAncestryID)
	}
	return campaignCreationHeritageByID(view.Ancestries, ancestryID)
}

// campaignCreationSelectedCommunity resolves the chosen community.
func campaignCreationSelectedCommunity(view CampaignCharacterCreationView) (CampaignCreationHeritageView, bool) {
	return campaignCreationHeritageByID(view.Communities, view.Heritage.CommunityID)
}

// campaignCreationHeritageByID resolves one heritage entry by ID.
func campaignCreationHeritageByID(items []CampaignCreationHeritageView, id string) (CampaignCreationHeritageView, bool) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return CampaignCreationHeritageView{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == trimmedID {
			return item, true
		}
	}
	return CampaignCreationHeritageView{}, false
}

// campaignCreationHeritageSlotFeatures returns only the feature granted by one ancestry slot.
func campaignCreationHeritageSlotFeatures(heritage CampaignCreationHeritageView, slotIndex int) []CampaignCreationClassFeatureView {
	if slotIndex < 0 || slotIndex >= len(heritage.Features) {
		return nil
	}
	return []CampaignCreationClassFeatureView{heritage.Features[slotIndex]}
}

// campaignCreationAncestryCardFeatures returns the ancestry features shown on one card.
func campaignCreationAncestryCardFeatures(view CampaignCharacterCreationView, heritage CampaignCreationHeritageView, _ int) []CampaignCreationClassFeatureView {
	return append([]CampaignCreationClassFeatureView(nil), heritage.Features...)
}

// campaignCreationAncestryCardSummaryFeatures returns the granted feature summary for one ancestry card.
func campaignCreationAncestryCardSummaryFeatures(view CampaignCharacterCreationView, heritage CampaignCreationHeritageView, slotIndex int) string {
	if slotIndex == 0 && campaignCreationHeritageMode(view) == "single" {
		return campaignCreationFeatureNames(heritage.Features)
	}
	return campaignCreationFeatureNames(campaignCreationHeritageSlotFeatures(heritage, slotIndex))
}

// campaignCreationMutedAncestryFeatureIndex returns the muted feature index for one slot.
func campaignCreationMutedAncestryFeatureIndex(view CampaignCharacterCreationView, slotIndex int) int {
	if campaignCreationHeritageMode(view) != "mixed" {
		return -1
	}
	if slotIndex == 0 {
		return 1
	}
	return 0
}

// campaignCreationFeatureNames joins feature names for compact summaries.
func campaignCreationFeatureNames(features []CampaignCreationClassFeatureView) string {
	names := make([]string, 0, len(features))
	for _, feature := range features {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// campaignCreationHeritageAutoLabel returns the derived mixed-heritage display fallback.
func campaignCreationHeritageAutoLabel(view CampaignCharacterCreationView) string {
	if campaignCreationHeritageMode(view) != "mixed" {
		return ""
	}
	if label := strings.TrimSpace(view.Heritage.AncestryLabel); label != "" {
		return label
	}
	return campaignCreationHeritageDisplayName(view)
}

// campaignCreationCompanionExperienceName returns one companion experience name or empty.
func campaignCreationCompanionExperienceName(companion *CampaignCreationCompanionView, index int) string {
	if companion == nil || index < 0 || index >= len(companion.Experiences) {
		return ""
	}
	return strings.TrimSpace(companion.Experiences[index].Name)
}

// campaignCreationCompanionExperienceID returns one companion experience ID so
// form fields can stay stable when companion data is partially populated.
func campaignCreationCompanionExperienceID(companion *CampaignCreationCompanionView, index int) string {
	if companion == nil || index < 0 || index >= len(companion.Experiences) {
		return ""
	}
	return strings.TrimSpace(companion.Experiences[index].ID)
}

// campaignCreationCompanionText returns one companion text field safely when companion data is optional.
func campaignCreationCompanionText(companion *CampaignCreationCompanionView, field string) string {
	if companion == nil {
		return ""
	}
	switch field {
	case "animal_kind":
		return strings.TrimSpace(companion.AnimalKind)
	case "name":
		return strings.TrimSpace(companion.Name)
	case "attack_description":
		return strings.TrimSpace(companion.AttackDescription)
	case "damage_type":
		return strings.TrimSpace(companion.DamageType)
	default:
		return ""
	}
}

// campaignCreationHasCompanionData reports whether any companion fields are populated.
func campaignCreationHasCompanionData(companion *CampaignCreationCompanionView) bool {
	return campaignCreationCompanionText(companion, "animal_kind") != "" ||
		campaignCreationCompanionText(companion, "name") != "" ||
		campaignCreationCompanionText(companion, "attack_description") != "" ||
		campaignCreationCompanionText(companion, "damage_type") != "" ||
		campaignCreationCompanionExperienceID(companion, 0) != "" ||
		campaignCreationCompanionExperienceID(companion, 1) != ""
}
