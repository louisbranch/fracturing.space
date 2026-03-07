package daggerheart

import (
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// buildCatalogCreation initializes the creation aggregate with normalized inputs.
func buildCatalogCreation(
	progress campaignapp.CampaignCharacterCreationProgress,
	profile campaignapp.CampaignCharacterCreationProfile,
) campaignapp.CampaignCharacterCreation {
	return campaignapp.CampaignCharacterCreation{
		Progress:         cloneProgress(progress),
		Profile:          normalizeProfile(profile),
		Classes:          []campaignapp.CatalogClass{},
		Subclasses:       []campaignapp.CatalogSubclass{},
		Ancestries:       []campaignapp.CatalogHeritage{},
		Communities:      []campaignapp.CatalogHeritage{},
		PrimaryWeapons:   []campaignapp.CatalogWeapon{},
		SecondaryWeapons: []campaignapp.CatalogWeapon{},
		Armor:            []campaignapp.CatalogArmor{},
		PotionItems:      []campaignapp.CatalogItem{},
		DomainCards:      []campaignapp.CatalogDomainCard{},
		Domains:          []campaignapp.CatalogDomain{},
	}
}

// cloneProgress copies progress slices so catalog assembly stays side-effect free.
func cloneProgress(progress campaignapp.CampaignCharacterCreationProgress) campaignapp.CampaignCharacterCreationProgress {
	return campaignapp.CampaignCharacterCreationProgress{
		Steps:        append([]campaignapp.CampaignCharacterCreationStep(nil), progress.Steps...),
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

// normalizeProfile trims profile fields and drops empty domain-card selections.
func normalizeProfile(profile campaignapp.CampaignCharacterCreationProfile) campaignapp.CampaignCharacterCreationProfile {
	selectedDomainCardIDs := make([]string, 0, len(profile.DomainCardIDs))
	for _, domainCardID := range profile.DomainCardIDs {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		selectedDomainCardIDs = append(selectedDomainCardIDs, trimmedDomainCardID)
	}

	return campaignapp.CampaignCharacterCreationProfile{
		CharacterName:     strings.TrimSpace(profile.CharacterName),
		ClassID:           strings.TrimSpace(profile.ClassID),
		SubclassID:        strings.TrimSpace(profile.SubclassID),
		AncestryID:        strings.TrimSpace(profile.AncestryID),
		CommunityID:       strings.TrimSpace(profile.CommunityID),
		Agility:           strings.TrimSpace(profile.Agility),
		Strength:          strings.TrimSpace(profile.Strength),
		Finesse:           strings.TrimSpace(profile.Finesse),
		Instinct:          strings.TrimSpace(profile.Instinct),
		Presence:          strings.TrimSpace(profile.Presence),
		Knowledge:         strings.TrimSpace(profile.Knowledge),
		PrimaryWeaponID:   strings.TrimSpace(profile.PrimaryWeaponID),
		SecondaryWeaponID: strings.TrimSpace(profile.SecondaryWeaponID),
		ArmorID:           strings.TrimSpace(profile.ArmorID),
		PotionItemID:      strings.TrimSpace(profile.PotionItemID),
		Background:        strings.TrimSpace(profile.Background),
		Description:       strings.TrimSpace(profile.Description),
		Experiences:       trimExperiences(profile.Experiences),
		DomainCardIDs:     selectedDomainCardIDs,
		Connections:       strings.TrimSpace(profile.Connections),
	}
}

// trimExperiences normalizes the experience slice from the profile.
func trimExperiences(exps []campaignapp.CampaignCharacterCreationExperience) []campaignapp.CampaignCharacterCreationExperience {
	result := make([]campaignapp.CampaignCharacterCreationExperience, 0, len(exps))
	for _, exp := range exps {
		name := strings.TrimSpace(exp.Name)
		if name == "" {
			continue
		}
		result = append(result, campaignapp.CampaignCharacterCreationExperience{
			Name:     name,
			Modifier: strings.TrimSpace(exp.Modifier),
		})
	}
	return result
}
