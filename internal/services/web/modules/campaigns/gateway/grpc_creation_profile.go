package gateway

import (
	"context"
	"strconv"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CharacterCreationProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) CharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProfile, error) {
	if g.CharacterClient == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.CharacterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProfile{}, err
	}
	if resp == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, nil
	}
	if resp.GetProfile() == nil || resp.GetProfile().GetDaggerheart() == nil {
		return campaignapp.CampaignCharacterCreationProfile{
			CharacterName: characterDisplayName(resp.GetCharacter()),
		}, nil
	}
	profile := resp.GetProfile().GetDaggerheart()
	characterName := characterDisplayName(resp.GetCharacter())

	startingWeaponIDs := make([]string, 0, len(profile.GetStartingWeaponIds()))
	for _, weaponID := range profile.GetStartingWeaponIds() {
		trimmedWeaponID := strings.TrimSpace(weaponID)
		if trimmedWeaponID == "" {
			continue
		}
		startingWeaponIDs = append(startingWeaponIDs, trimmedWeaponID)
	}
	primaryWeaponID := ""
	secondaryWeaponID := ""
	if len(startingWeaponIDs) > 0 {
		primaryWeaponID = startingWeaponIDs[0]
	}
	if len(startingWeaponIDs) > 1 {
		secondaryWeaponID = startingWeaponIDs[1]
	}

	domainCardIDs := make([]string, 0, len(profile.GetDomainCardIds()))
	for _, domainCardID := range profile.GetDomainCardIds() {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
	}

	experiences := make([]campaignapp.CampaignCharacterCreationExperience, 0, len(profile.GetExperiences()))
	for _, exp := range profile.GetExperiences() {
		if exp == nil {
			continue
		}
		name := strings.TrimSpace(exp.GetName())
		if name == "" {
			continue
		}
		experiences = append(experiences, campaignapp.CampaignCharacterCreationExperience{
			Name:     name,
			Modifier: strconv.FormatInt(int64(exp.GetModifier()), 10),
		})
	}

	return campaignapp.CampaignCharacterCreationProfile{
		CharacterName:     characterName,
		ClassID:           strings.TrimSpace(profile.GetClassId()),
		SubclassID:        strings.TrimSpace(profile.GetSubclassId()),
		AncestryID:        strings.TrimSpace(profile.GetAncestryId()),
		CommunityID:       strings.TrimSpace(profile.GetCommunityId()),
		Agility:           int32ValueString(profile.GetAgility()),
		Strength:          int32ValueString(profile.GetStrength()),
		Finesse:           int32ValueString(profile.GetFinesse()),
		Instinct:          int32ValueString(profile.GetInstinct()),
		Presence:          int32ValueString(profile.GetPresence()),
		Knowledge:         int32ValueString(profile.GetKnowledge()),
		PrimaryWeaponID:   primaryWeaponID,
		SecondaryWeaponID: secondaryWeaponID,
		ArmorID:           strings.TrimSpace(profile.GetStartingArmorId()),
		PotionItemID:      strings.TrimSpace(profile.GetStartingPotionItemId()),
		Background:        strings.TrimSpace(profile.GetBackground()),
		Description:       strings.TrimSpace(profile.GetDescription()),
		Experiences:       experiences,
		DomainCardIDs:     domainCardIDs,
		Connections:       strings.TrimSpace(profile.GetConnections()),
	}, nil
}
