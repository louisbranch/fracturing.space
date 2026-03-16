package daggerheart

import (
	"strings"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

// catalogCreation keeps the assembled character-creation catalog local to the
// Daggerheart workflow package so app contracts stay on explicit reads only.
type catalogCreation struct {
	Progress             campaignworkflow.Progress
	Profile              campaignworkflow.Profile
	Classes              []campaignworkflow.Class
	Subclasses           []campaignworkflow.Subclass
	Ancestries           []campaignworkflow.Heritage
	Communities          []campaignworkflow.Heritage
	CompanionExperiences []campaignworkflow.CompanionExperience
	PrimaryWeapons       []campaignworkflow.Weapon
	SecondaryWeapons     []campaignworkflow.Weapon
	Armor                []campaignworkflow.Armor
	PotionItems          []campaignworkflow.Item
	DomainCards          []campaignworkflow.DomainCard
	Domains              []campaignworkflow.Domain
}

// buildCatalogCreation initializes the creation aggregate with normalized inputs.
func buildCatalogCreation(
	progress campaignworkflow.Progress,
	profile campaignworkflow.Profile,
) catalogCreation {
	return catalogCreation{
		Progress:             cloneProgress(progress),
		Profile:              normalizeProfile(profile),
		Classes:              []campaignworkflow.Class{},
		Subclasses:           []campaignworkflow.Subclass{},
		Ancestries:           []campaignworkflow.Heritage{},
		Communities:          []campaignworkflow.Heritage{},
		CompanionExperiences: []campaignworkflow.CompanionExperience{},
		PrimaryWeapons:       []campaignworkflow.Weapon{},
		SecondaryWeapons:     []campaignworkflow.Weapon{},
		Armor:                []campaignworkflow.Armor{},
		PotionItems:          []campaignworkflow.Item{},
		DomainCards:          []campaignworkflow.DomainCard{},
		Domains:              []campaignworkflow.Domain{},
	}
}

// cloneProgress copies progress slices so catalog assembly stays side-effect free.
func cloneProgress(progress campaignworkflow.Progress) campaignworkflow.Progress {
	return campaignworkflow.Progress{
		Steps:        append([]campaignworkflow.Step(nil), progress.Steps...),
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

// normalizeProfile trims profile fields and drops empty domain-card selections.
func normalizeProfile(profile campaignworkflow.Profile) campaignworkflow.Profile {
	selectedDomainCardIDs := make([]string, 0, len(profile.DomainCardIDs))
	for _, domainCardID := range profile.DomainCardIDs {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		selectedDomainCardIDs = append(selectedDomainCardIDs, trimmedDomainCardID)
	}

	return campaignworkflow.Profile{
		CharacterName:                strings.TrimSpace(profile.CharacterName),
		ClassID:                      strings.TrimSpace(profile.ClassID),
		SubclassID:                   strings.TrimSpace(profile.SubclassID),
		SubclassCreationRequirements: trimNonEmptyStrings(profile.SubclassCreationRequirements),
		Heritage:                     trimHeritageSelection(profile.Heritage),
		CompanionSheet:               trimCompanionSheet(profile.CompanionSheet),
		Agility:                      strings.TrimSpace(profile.Agility),
		Strength:                     strings.TrimSpace(profile.Strength),
		Finesse:                      strings.TrimSpace(profile.Finesse),
		Instinct:                     strings.TrimSpace(profile.Instinct),
		Presence:                     strings.TrimSpace(profile.Presence),
		Knowledge:                    strings.TrimSpace(profile.Knowledge),
		PrimaryWeaponID:              strings.TrimSpace(profile.PrimaryWeaponID),
		SecondaryWeaponID:            strings.TrimSpace(profile.SecondaryWeaponID),
		ArmorID:                      strings.TrimSpace(profile.ArmorID),
		PotionItemID:                 strings.TrimSpace(profile.PotionItemID),
		Background:                   strings.TrimSpace(profile.Background),
		Description:                  strings.TrimSpace(profile.Description),
		Experiences:                  trimExperiences(profile.Experiences),
		DomainCardIDs:                selectedDomainCardIDs,
		Connections:                  strings.TrimSpace(profile.Connections),
	}
}

// trimNonEmptyStrings normalizes small string slices used by web-facing profile DTOs.
func trimNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

// trimHeritageSelection keeps structured heritage fields normalized before view assembly.
func trimHeritageSelection(selection campaignworkflow.HeritageSelection) campaignworkflow.HeritageSelection {
	return campaignworkflow.HeritageSelection{
		AncestryLabel:           strings.TrimSpace(selection.AncestryLabel),
		FirstFeatureAncestryID:  strings.TrimSpace(selection.FirstFeatureAncestryID),
		FirstFeatureID:          strings.TrimSpace(selection.FirstFeatureID),
		SecondFeatureAncestryID: strings.TrimSpace(selection.SecondFeatureAncestryID),
		SecondFeatureID:         strings.TrimSpace(selection.SecondFeatureID),
		CommunityID:             strings.TrimSpace(selection.CommunityID),
	}
}

// trimCompanionSheet normalizes optional companion input for render consumers.
func trimCompanionSheet(sheet *campaignworkflow.CompanionSheet) *campaignworkflow.CompanionSheet {
	if sheet == nil {
		return nil
	}
	return &campaignworkflow.CompanionSheet{
		AnimalKind:        strings.TrimSpace(sheet.AnimalKind),
		Name:              strings.TrimSpace(sheet.Name),
		Evasion:           sheet.Evasion,
		Experiences:       trimExperiences(sheet.Experiences),
		AttackDescription: strings.TrimSpace(sheet.AttackDescription),
		AttackRange:       strings.TrimSpace(sheet.AttackRange),
		DamageDieSides:    sheet.DamageDieSides,
		DamageType:        strings.TrimSpace(sheet.DamageType),
	}
}

// trimExperiences normalizes the experience slice from the profile.
func trimExperiences(exps []campaignworkflow.Experience) []campaignworkflow.Experience {
	result := make([]campaignworkflow.Experience, 0, len(exps))
	for _, exp := range exps {
		name := strings.TrimSpace(exp.Name)
		if name == "" {
			continue
		}
		result = append(result, campaignworkflow.Experience{
			Name:     name,
			Modifier: strings.TrimSpace(exp.Modifier),
		})
	}
	return result
}
