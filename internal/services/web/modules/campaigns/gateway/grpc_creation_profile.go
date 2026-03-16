package gateway

import (
	"context"
	"strconv"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CharacterCreationProfile centralizes this web behavior in one helper seam.
func (g characterCreationReadGateway) CharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProfile, error) {
	if g.read.Character == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.read.Character.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProfile{}, err
	}
	return mapCharacterCreationProfile(resp), nil
}

// mapCharacterCreationProfile converts a character sheet response into creation profile values.
func mapCharacterCreationProfile(resp *statev1.GetCharacterSheetResponse) campaignapp.CampaignCharacterCreationProfile {
	if resp == nil {
		return campaignapp.CampaignCharacterCreationProfile{}
	}

	characterName := characterDisplayName(resp.GetCharacter())
	if resp.GetProfile() == nil || resp.GetProfile().GetDaggerheart() == nil {
		return campaignapp.CampaignCharacterCreationProfile{CharacterName: characterName}
	}

	profile := resp.GetProfile().GetDaggerheart()
	primaryWeaponID, secondaryWeaponID := startingWeaponIDs(profile)

	return campaignapp.CampaignCharacterCreationProfile{
		CharacterName:                characterName,
		ClassID:                      strings.TrimSpace(profile.GetClassId()),
		SubclassID:                   strings.TrimSpace(profile.GetSubclassId()),
		SubclassCreationRequirements: trimNonEmptyProfileValues(profile.GetSubclassCreationRequirements()),
		Heritage:                     mapProfileHeritage(profile.GetHeritage()),
		CompanionSheet:               mapProfileCompanionSheet(profile.GetCompanionSheet()),
		Agility:                      int32ValueString(profile.GetAgility()),
		Strength:                     int32ValueString(profile.GetStrength()),
		Finesse:                      int32ValueString(profile.GetFinesse()),
		Instinct:                     int32ValueString(profile.GetInstinct()),
		Presence:                     int32ValueString(profile.GetPresence()),
		Knowledge:                    int32ValueString(profile.GetKnowledge()),
		PrimaryWeaponID:              primaryWeaponID,
		SecondaryWeaponID:            secondaryWeaponID,
		ArmorID:                      strings.TrimSpace(profile.GetStartingArmorId()),
		PotionItemID:                 strings.TrimSpace(profile.GetStartingPotionItemId()),
		Background:                   strings.TrimSpace(profile.GetBackground()),
		Description:                  strings.TrimSpace(profile.GetDescription()),
		Experiences:                  mapProfileExperiences(profile.GetExperiences()),
		DomainCardIDs:                trimNonEmptyProfileValues(profile.GetDomainCardIds()),
		Connections:                  strings.TrimSpace(profile.GetConnections()),
	}
}

// mapProfileHeritage projects structured heritage from the proto profile into the web DTO.
func mapProfileHeritage(heritage *daggerheartv1.DaggerheartHeritageSelection) campaignapp.CampaignCharacterCreationHeritageSelection {
	if heritage == nil {
		return campaignapp.CampaignCharacterCreationHeritageSelection{}
	}
	return campaignapp.CampaignCharacterCreationHeritageSelection{
		AncestryLabel:           strings.TrimSpace(heritage.GetAncestryLabel()),
		FirstFeatureAncestryID:  strings.TrimSpace(heritage.GetFirstFeatureAncestryId()),
		FirstFeatureID:          strings.TrimSpace(heritage.GetFirstFeatureId()),
		SecondFeatureAncestryID: strings.TrimSpace(heritage.GetSecondFeatureAncestryId()),
		SecondFeatureID:         strings.TrimSpace(heritage.GetSecondFeatureId()),
		CommunityID:             strings.TrimSpace(heritage.GetCommunityId()),
	}
}

// mapProfileCompanionSheet projects the stored companion sheet into the web DTO.
func mapProfileCompanionSheet(sheet *daggerheartv1.DaggerheartCompanionSheet) *campaignapp.CampaignCharacterCreationCompanionSheet {
	if sheet == nil {
		return nil
	}
	return &campaignapp.CampaignCharacterCreationCompanionSheet{
		AnimalKind:        strings.TrimSpace(sheet.GetAnimalKind()),
		Name:              strings.TrimSpace(sheet.GetName()),
		Evasion:           sheet.GetEvasion(),
		Experiences:       mapProfileCompanionExperiences(sheet.GetExperiences()),
		AttackDescription: strings.TrimSpace(sheet.GetAttackDescription()),
		AttackRange:       strings.TrimSpace(sheet.GetAttackRange()),
		DamageDieSides:    sheet.GetDamageDieSides(),
		DamageType:        strings.TrimSpace(sheet.GetDamageType()),
	}
}

// mapProfileCompanionExperiences preserves companion experience identity while
// translating the gRPC profile shape into the app-owned creation DTO.
func mapProfileCompanionExperiences(experiences []*daggerheartv1.DaggerheartCompanionExperience) []campaignapp.CampaignCharacterCreationExperience {
	mapped := make([]campaignapp.CampaignCharacterCreationExperience, 0, len(experiences))
	for _, experience := range experiences {
		if experience == nil {
			continue
		}
		mapped = append(mapped, campaignapp.CampaignCharacterCreationExperience{
			ID:       strings.TrimSpace(experience.GetExperienceId()),
			Name:     strings.TrimSpace(experience.GetName()),
			Modifier: strconv.Itoa(int(experience.GetModifier())),
		})
	}
	return mapped
}

// startingWeaponIDs extracts the first two starting weapon IDs as primary/secondary values.
func startingWeaponIDs(profile *daggerheartv1.DaggerheartProfile) (string, string) {
	if profile == nil {
		return "", ""
	}
	ids := trimNonEmptyProfileValues(profile.GetStartingWeaponIds())
	if len(ids) == 0 {
		return "", ""
	}
	if len(ids) == 1 {
		return ids[0], ""
	}
	return ids[0], ids[1]
}

// mapProfileExperiences maps profile experiences while removing blank names.
func mapProfileExperiences(experiences []*daggerheartv1.DaggerheartExperience) []campaignapp.CampaignCharacterCreationExperience {
	mapped := make([]campaignapp.CampaignCharacterCreationExperience, 0, len(experiences))
	for _, experience := range experiences {
		if experience == nil {
			continue
		}
		name := strings.TrimSpace(experience.GetName())
		if name == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CampaignCharacterCreationExperience{
			Name:     name,
			Modifier: strconv.FormatInt(int64(experience.GetModifier()), 10),
		})
	}
	return mapped
}

// trimNonEmptyProfileValues trims whitespace and drops empty values while preserving order.
func trimNonEmptyProfileValues(values []string) []string {
	mapped := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		mapped = append(mapped, trimmed)
	}
	return mapped
}
