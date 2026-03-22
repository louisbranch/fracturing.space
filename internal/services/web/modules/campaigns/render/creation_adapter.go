package render

import campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"

// NewCharacterCreationView adapts the workflow-owned creation model to the
// render contract used by templates and detail views.
func NewCharacterCreationView(view campaignworkflow.CharacterCreationView) CampaignCharacterCreationView {
	rv := CampaignCharacterCreationView{
		Ready:                        view.Ready,
		NextStep:                     view.NextStep,
		UnmetReasons:                 append([]string(nil), view.UnmetReasons...),
		ClassID:                      view.ClassID,
		SubclassID:                   view.SubclassID,
		SubclassCreationRequirements: append([]string(nil), view.SubclassCreationRequirements...),
		Heritage:                     mapHeritageSelectionView(view.Heritage),
		CompanionSheet:               mapCompanionView(view.CompanionSheet),
		CompanionExperiences:         mapCompanionExperienceOptionViews(view.CompanionExperiences),
		Agility:                      view.Agility,
		Strength:                     view.Strength,
		Finesse:                      view.Finesse,
		Instinct:                     view.Instinct,
		Presence:                     view.Presence,
		Knowledge:                    view.Knowledge,
		PrimaryWeaponID:              view.PrimaryWeaponID,
		SecondaryWeaponID:            view.SecondaryWeaponID,
		ArmorID:                      view.ArmorID,
		PotionItemID:                 view.PotionItemID,
		Background:                   view.Background,
		Description:                  view.Description,
		Experiences:                  mapExperienceViews(view.Experiences),
		DomainCardIDs:                append([]string(nil), view.DomainCardIDs...),
		Connections:                  view.Connections,
		NextStepPrefetchURLs:         append([]string(nil), view.NextStepPrefetchURLs...),
		Steps:                        mapStepViews(view.Steps),
		Classes:                      mapClassViews(view.Classes),
		Subclasses:                   mapSubclassViews(view.Subclasses),
		Ancestries:                   mapHeritageViews(view.Ancestries),
		Communities:                  mapHeritageViews(view.Communities),
		PrimaryWeapons:               mapWeaponViews(view.PrimaryWeapons),
		SecondaryWeapons:             mapWeaponViews(view.SecondaryWeapons),
		PrimaryWeaponGroups:          mapWeaponGroupViews(view.PrimaryWeaponGroups),
		SecondaryWeaponGroups:        mapWeaponGroupViews(view.SecondaryWeaponGroups),
		SecondaryWeaponNoneImageURL:  view.SecondaryWeaponNoneImageURL,
		Armor:                        mapArmorViews(view.Armor),
		PotionItems:                  mapItemViews(view.PotionItems),
		DomainCards:                  mapDomainCardViews(view.DomainCards),
	}
	rv.TraitOptions = daggerheartCreationTraitOptions(rv)
	return rv
}

// NewCharacterCreationPageView adapts one workflow-owned page result to the
// dedicated creation-page render contract.
func NewCharacterCreationPageView(campaignID string, characterID string, page campaignworkflow.PageData) CharacterCreationPageView {
	return CharacterCreationPageView{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Creation:    NewCharacterCreationView(page.Creation),
	}
}

// mapStepViews copies workflow step rows into the template seam.
func mapStepViews(steps []campaignworkflow.CharacterCreationStepView) []CampaignCharacterCreationStepView {
	mapped := make([]CampaignCharacterCreationStepView, 0, len(steps))
	for _, step := range steps {
		mapped = append(mapped, CampaignCharacterCreationStepView{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	return mapped
}

// mapCompanionExperienceOptionViews copies workflow companion experience
// options into the template seam without leaking workflow types into render.
func mapCompanionExperienceOptionViews(experiences []campaignworkflow.CreationCompanionExperienceOptionView) []CampaignCreationCompanionExperienceOptionView {
	mapped := make([]CampaignCreationCompanionExperienceOptionView, 0, len(experiences))
	for _, experience := range experiences {
		mapped = append(mapped, CampaignCreationCompanionExperienceOptionView{
			ID:          experience.ID,
			Name:        experience.Name,
			Description: experience.Description,
		})
	}
	return mapped
}

// mapClassFeatureViews copies workflow feature rows into the template seam.
func mapClassFeatureViews(features []campaignworkflow.CreationClassFeatureView) []CampaignCreationClassFeatureView {
	mapped := make([]CampaignCreationClassFeatureView, 0, len(features))
	for _, feature := range features {
		mapped = append(mapped, CampaignCreationClassFeatureView{
			Name:        feature.Name,
			Description: feature.Description,
		})
	}
	return mapped
}

// mapDomainWatermarkViews copies workflow watermark rows into the template seam.
func mapDomainWatermarkViews(watermarks []campaignworkflow.CreationDomainWatermarkView) []CampaignCreationDomainWatermarkView {
	mapped := make([]CampaignCreationDomainWatermarkView, 0, len(watermarks))
	for _, watermark := range watermarks {
		mapped = append(mapped, CampaignCreationDomainWatermarkView{
			ID:      watermark.ID,
			Name:    watermark.Name,
			IconURL: watermark.IconURL,
		})
	}
	return mapped
}

// mapClassViews copies workflow class cards into the template seam.
func mapClassViews(classes []campaignworkflow.CreationClassView) []CampaignCreationClassView {
	mapped := make([]CampaignCreationClassView, 0, len(classes))
	for _, class := range classes {
		mapped = append(mapped, CampaignCreationClassView{
			ID:               class.ID,
			Name:             class.Name,
			ImageURL:         class.ImageURL,
			StartingHP:       class.StartingHP,
			StartingEvasion:  class.StartingEvasion,
			HopeFeature:      CampaignCreationClassFeatureView{Name: class.HopeFeature.Name, Description: class.HopeFeature.Description},
			Features:         mapClassFeatureViews(class.Features),
			DomainNames:      append([]string(nil), class.DomainNames...),
			DomainWatermarks: mapDomainWatermarkViews(class.DomainWatermarks),
		})
	}
	return mapped
}

// mapSubclassViews copies workflow subclass cards into the template seam.
func mapSubclassViews(subclasses []campaignworkflow.CreationSubclassView) []CampaignCreationSubclassView {
	mapped := make([]CampaignCreationSubclassView, 0, len(subclasses))
	for _, subclass := range subclasses {
		mapped = append(mapped, CampaignCreationSubclassView{
			ID:                   subclass.ID,
			Name:                 subclass.Name,
			ImageURL:             subclass.ImageURL,
			ClassID:              subclass.ClassID,
			SpellcastTrait:       subclass.SpellcastTrait,
			CreationRequirements: append([]string(nil), subclass.CreationRequirements...),
			Foundation:           mapClassFeatureViews(subclass.Foundation),
		})
	}
	return mapped
}

// mapHeritageViews copies workflow ancestry/community cards into the template seam.
func mapHeritageViews(heritages []campaignworkflow.CreationHeritageView) []CampaignCreationHeritageView {
	mapped := make([]CampaignCreationHeritageView, 0, len(heritages))
	for _, heritage := range heritages {
		mapped = append(mapped, CampaignCreationHeritageView{
			ID:       heritage.ID,
			Name:     heritage.Name,
			ImageURL: heritage.ImageURL,
			Features: mapClassFeatureViews(heritage.Features),
		})
	}
	return mapped
}

// mapWeaponViews copies workflow weapon cards into the template seam.
func mapWeaponViews(weapons []campaignworkflow.CreationWeaponView) []CampaignCreationWeaponView {
	mapped := make([]CampaignCreationWeaponView, 0, len(weapons))
	for _, weapon := range weapons {
		mapped = append(mapped, CampaignCreationWeaponView{
			ID:           weapon.ID,
			Name:         weapon.Name,
			ImageURL:     weapon.ImageURL,
			Burden:       weapon.Burden,
			Trait:        weapon.Trait,
			Range:        weapon.Range,
			Damage:       weapon.Damage,
			Feature:      weapon.Feature,
			DisplayGroup: weapon.DisplayGroup,
		})
	}
	return mapped
}

// mapWeaponGroupViews copies workflow weapon groups into the template seam.
func mapWeaponGroupViews(groups []campaignworkflow.CreationWeaponGroupView) []CampaignCreationWeaponGroupView {
	mapped := make([]CampaignCreationWeaponGroupView, 0, len(groups))
	for _, group := range groups {
		mapped = append(mapped, CampaignCreationWeaponGroupView{
			Key:     group.Key,
			Weapons: mapWeaponViews(group.Weapons),
		})
	}
	return mapped
}

// mapArmorViews copies workflow armor cards into the template seam.
func mapArmorViews(armor []campaignworkflow.CreationArmorView) []CampaignCreationArmorView {
	mapped := make([]CampaignCreationArmorView, 0, len(armor))
	for _, item := range armor {
		mapped = append(mapped, CampaignCreationArmorView{
			ID:             item.ID,
			Name:           item.Name,
			ImageURL:       item.ImageURL,
			ArmorScore:     item.ArmorScore,
			BaseThresholds: item.BaseThresholds,
			Feature:        item.Feature,
		})
	}
	return mapped
}

// mapItemViews copies workflow item cards into the template seam.
func mapItemViews(items []campaignworkflow.CreationItemView) []CampaignCreationItemView {
	mapped := make([]CampaignCreationItemView, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, CampaignCreationItemView{
			ID:          item.ID,
			Name:        item.Name,
			ImageURL:    item.ImageURL,
			Description: item.Description,
		})
	}
	return mapped
}

// mapExperienceViews copies workflow experience rows into the template seam.
func mapExperienceViews(experiences []campaignworkflow.CreationExperienceView) []CampaignCreationExperienceView {
	mapped := make([]CampaignCreationExperienceView, 0, len(experiences))
	for _, experience := range experiences {
		mapped = append(mapped, CampaignCreationExperienceView{
			ID:       experience.ID,
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}
	return mapped
}

// mapHeritageSelectionView copies workflow heritage state into the template seam.
func mapHeritageSelectionView(selection campaignworkflow.CreationHeritageSelectionView) CampaignCreationHeritageSelectionView {
	return CampaignCreationHeritageSelectionView{
		AncestryLabel:           selection.AncestryLabel,
		FirstFeatureAncestryID:  selection.FirstFeatureAncestryID,
		FirstFeatureID:          selection.FirstFeatureID,
		SecondFeatureAncestryID: selection.SecondFeatureAncestryID,
		SecondFeatureID:         selection.SecondFeatureID,
		CommunityID:             selection.CommunityID,
	}
}

// mapCompanionView copies workflow companion state into the template seam.
func mapCompanionView(sheet *campaignworkflow.CreationCompanionView) *CampaignCreationCompanionView {
	if sheet == nil {
		return nil
	}
	return &CampaignCreationCompanionView{
		AnimalKind:        sheet.AnimalKind,
		Name:              sheet.Name,
		Evasion:           sheet.Evasion,
		Experiences:       mapExperienceViews(sheet.Experiences),
		AttackDescription: sheet.AttackDescription,
		AttackRange:       sheet.AttackRange,
		DamageDieSides:    sheet.DamageDieSides,
		DamageType:        sheet.DamageType,
	}
}

// mapDomainCardViews copies workflow domain-card rows into the template seam.
func mapDomainCardViews(cards []campaignworkflow.CreationDomainCardView) []CampaignCreationDomainCardView {
	mapped := make([]CampaignCreationDomainCardView, 0, len(cards))
	for _, card := range cards {
		mapped = append(mapped, CampaignCreationDomainCardView{
			ID:          card.ID,
			Name:        card.Name,
			ImageURL:    card.ImageURL,
			DomainID:    card.DomainID,
			DomainName:  card.DomainName,
			Level:       card.Level,
			Type:        card.Type,
			RecallCost:  card.RecallCost,
			FeatureText: card.FeatureText,
		})
	}
	return mapped
}
