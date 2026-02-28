package gateway

import (
	"context"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func (g GRPCGateway) CharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProgress, error) {
	if g.CharacterClient == nil {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.CharacterClient.GetCharacterCreationProgress(ctx, &statev1.GetCharacterCreationProgressRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProgress{}, err
	}
	if resp == nil || resp.GetProgress() == nil {
		return campaignapp.CampaignCharacterCreationProgress{Steps: []campaignapp.CampaignCharacterCreationStep{}, UnmetReasons: []string{}}, nil
	}

	progress := resp.GetProgress()
	steps := make([]campaignapp.CampaignCharacterCreationStep, 0, len(progress.GetSteps()))
	for _, step := range progress.GetSteps() {
		if step == nil {
			continue
		}
		steps = append(steps, campaignapp.CampaignCharacterCreationStep{
			Step:     step.GetStep(),
			Key:      strings.TrimSpace(step.GetKey()),
			Complete: step.GetComplete(),
		})
	}
	unmetReasons := make([]string, 0, len(progress.GetUnmetReasons()))
	for _, reason := range progress.GetUnmetReasons() {
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason == "" {
			continue
		}
		unmetReasons = append(unmetReasons, trimmedReason)
	}
	return campaignapp.CampaignCharacterCreationProgress{
		Steps:        steps,
		NextStep:     progress.GetNextStep(),
		Ready:        progress.GetReady(),
		UnmetReasons: unmetReasons,
	}, nil
}

func (g GRPCGateway) CharacterCreationCatalog(ctx context.Context, localeTag language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	if g.DaggerheartClient == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}
	locale := platformi18n.LocaleForTag(localeTag)
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}

	resp, err := g.DaggerheartClient.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
	if err != nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, err
	}
	if resp == nil || resp.GetCatalog() == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, nil
	}

	catalogResp := resp.GetCatalog()
	catalog := campaignapp.CampaignCharacterCreationCatalog{}

	catalog.Classes = make([]campaignapp.CatalogClass, 0, len(catalogResp.GetClasses()))
	for _, class := range catalogResp.GetClasses() {
		if class == nil {
			continue
		}
		classID := strings.TrimSpace(class.GetId())
		if classID == "" {
			continue
		}
		domainIDs := make([]string, 0, len(class.GetDomainIds()))
		for _, domainID := range class.GetDomainIds() {
			trimmedDomainID := strings.TrimSpace(domainID)
			if trimmedDomainID == "" {
				continue
			}
			domainIDs = append(domainIDs, trimmedDomainID)
		}
		catalog.Classes = append(catalog.Classes, campaignapp.CatalogClass{
			ID:        classID,
			Name:      strings.TrimSpace(class.GetName()),
			DomainIDs: domainIDs,
		})
	}

	catalog.Subclasses = make([]campaignapp.CatalogSubclass, 0, len(catalogResp.GetSubclasses()))
	for _, subclass := range catalogResp.GetSubclasses() {
		if subclass == nil {
			continue
		}
		subclassID := strings.TrimSpace(subclass.GetId())
		if subclassID == "" {
			continue
		}
		catalog.Subclasses = append(catalog.Subclasses, campaignapp.CatalogSubclass{
			ID:      subclassID,
			Name:    strings.TrimSpace(subclass.GetName()),
			ClassID: strings.TrimSpace(subclass.GetClassId()),
		})
	}

	catalog.Heritages = make([]campaignapp.CatalogHeritage, 0, len(catalogResp.GetHeritages()))
	for _, heritage := range catalogResp.GetHeritages() {
		if heritage == nil {
			continue
		}
		heritageID := strings.TrimSpace(heritage.GetId())
		if heritageID == "" {
			continue
		}
		catalog.Heritages = append(catalog.Heritages, campaignapp.CatalogHeritage{
			ID:   heritageID,
			Name: strings.TrimSpace(heritage.GetName()),
			Kind: daggerheartHeritageKindLabel(heritage.GetKind()),
		})
	}

	catalog.Weapons = make([]campaignapp.CatalogWeapon, 0, len(catalogResp.GetWeapons()))
	for _, weapon := range catalogResp.GetWeapons() {
		if weapon == nil {
			continue
		}
		weaponID := strings.TrimSpace(weapon.GetId())
		if weaponID == "" {
			continue
		}
		catalog.Weapons = append(catalog.Weapons, campaignapp.CatalogWeapon{
			ID:       weaponID,
			Name:     strings.TrimSpace(weapon.GetName()),
			Category: daggerheartWeaponCategoryLabel(weapon.GetCategory()),
			Tier:     weapon.GetTier(),
		})
	}

	catalog.Armor = make([]campaignapp.CatalogArmor, 0, len(catalogResp.GetArmor()))
	for _, armor := range catalogResp.GetArmor() {
		if armor == nil {
			continue
		}
		armorID := strings.TrimSpace(armor.GetId())
		if armorID == "" {
			continue
		}
		catalog.Armor = append(catalog.Armor, campaignapp.CatalogArmor{
			ID:   armorID,
			Name: strings.TrimSpace(armor.GetName()),
			Tier: armor.GetTier(),
		})
	}

	catalog.Items = make([]campaignapp.CatalogItem, 0, len(catalogResp.GetItems()))
	for _, item := range catalogResp.GetItems() {
		if item == nil {
			continue
		}
		itemID := strings.TrimSpace(item.GetId())
		if itemID == "" {
			continue
		}
		catalog.Items = append(catalog.Items, campaignapp.CatalogItem{
			ID:   itemID,
			Name: strings.TrimSpace(item.GetName()),
		})
	}

	catalog.DomainCards = make([]campaignapp.CatalogDomainCard, 0, len(catalogResp.GetDomainCards()))
	for _, domainCard := range catalogResp.GetDomainCards() {
		if domainCard == nil {
			continue
		}
		domainCardID := strings.TrimSpace(domainCard.GetId())
		if domainCardID == "" {
			continue
		}
		catalog.DomainCards = append(catalog.DomainCards, campaignapp.CatalogDomainCard{
			ID:       domainCardID,
			Name:     strings.TrimSpace(domainCard.GetName()),
			DomainID: strings.TrimSpace(domainCard.GetDomainId()),
			Level:    domainCard.GetLevel(),
		})
	}

	return catalog, nil
}

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
	if resp == nil || resp.GetProfile() == nil || resp.GetProfile().GetDaggerheart() == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, nil
	}
	profile := resp.GetProfile().GetDaggerheart()

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

	experienceName := ""
	experienceModifier := ""
	if len(profile.GetExperiences()) > 0 && profile.GetExperiences()[0] != nil {
		experienceName = strings.TrimSpace(profile.GetExperiences()[0].GetName())
		experienceModifier = strconv.FormatInt(int64(profile.GetExperiences()[0].GetModifier()), 10)
	}

	return campaignapp.CampaignCharacterCreationProfile{
		ClassID:            strings.TrimSpace(profile.GetClassId()),
		SubclassID:         strings.TrimSpace(profile.GetSubclassId()),
		AncestryID:         strings.TrimSpace(profile.GetAncestryId()),
		CommunityID:        strings.TrimSpace(profile.GetCommunityId()),
		Agility:            int32ValueString(profile.GetAgility()),
		Strength:           int32ValueString(profile.GetStrength()),
		Finesse:            int32ValueString(profile.GetFinesse()),
		Instinct:           int32ValueString(profile.GetInstinct()),
		Presence:           int32ValueString(profile.GetPresence()),
		Knowledge:          int32ValueString(profile.GetKnowledge()),
		PrimaryWeaponID:    primaryWeaponID,
		SecondaryWeaponID:  secondaryWeaponID,
		ArmorID:            strings.TrimSpace(profile.GetStartingArmorId()),
		PotionItemID:       strings.TrimSpace(profile.GetStartingPotionItemId()),
		Background:         strings.TrimSpace(profile.GetBackground()),
		ExperienceName:     experienceName,
		ExperienceModifier: experienceModifier,
		DomainCardIDs:      domainCardIDs,
		Connections:        strings.TrimSpace(profile.GetConnections()),
	}, nil
}

func (g GRPCGateway) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *campaignapp.CampaignCharacterCreationStepInput) error {
	if g.CharacterClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	if step == nil {
		return apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}
	systemStep, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		return err
	}

	_, err = g.CharacterClient.ApplyCharacterCreationStep(ctx, &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStep:  &statev1.ApplyCharacterCreationStepRequest_Daggerheart{Daggerheart: systemStep},
	})
	return err
}

func mapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	if step == nil {
		return nil, apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}

	active := 0
	if step.ClassSubclass != nil {
		active++
	}
	if step.Heritage != nil {
		active++
	}
	if step.Traits != nil {
		active++
	}
	if step.Details != nil {
		active++
	}
	if step.Equipment != nil {
		active++
	}
	if step.Background != nil {
		active++
	}
	if step.Experiences != nil {
		active++
	}
	if step.DomainCards != nil {
		active++
	}
	if step.Connections != nil {
		active++
	}
	if active != 1 {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}

	if step.ClassSubclass != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
				ClassId:    strings.TrimSpace(step.ClassSubclass.ClassID),
				SubclassId: strings.TrimSpace(step.ClassSubclass.SubclassID),
			}},
		}, nil
	}
	if step.Heritage != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
				AncestryId:  strings.TrimSpace(step.Heritage.AncestryID),
				CommunityId: strings.TrimSpace(step.Heritage.CommunityID),
			}},
		}, nil
	}
	if step.Traits != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{
				Agility:   step.Traits.Agility,
				Strength:  step.Traits.Strength,
				Finesse:   step.Traits.Finesse,
				Instinct:  step.Traits.Instinct,
				Presence:  step.Traits.Presence,
				Knowledge: step.Traits.Knowledge,
			}},
		}, nil
	}
	if step.Details != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{}},
		}, nil
	}
	if step.Equipment != nil {
		weaponIDs := make([]string, 0, len(step.Equipment.WeaponIDs))
		for _, weaponID := range step.Equipment.WeaponIDs {
			trimmedWeaponID := strings.TrimSpace(weaponID)
			if trimmedWeaponID == "" {
				continue
			}
			weaponIDs = append(weaponIDs, trimmedWeaponID)
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
				WeaponIds:    weaponIDs,
				ArmorId:      strings.TrimSpace(step.Equipment.ArmorID),
				PotionItemId: strings.TrimSpace(step.Equipment.PotionItemID),
			}},
		}, nil
	}
	if step.Background != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{
				Background: strings.TrimSpace(step.Background.Background),
			}},
		}, nil
	}
	if step.Experiences != nil {
		experiences := make([]*daggerheartv1.DaggerheartExperience, 0, len(step.Experiences.Experiences))
		for _, experience := range step.Experiences.Experiences {
			experienceName := strings.TrimSpace(experience.Name)
			if experienceName == "" {
				continue
			}
			experiences = append(experiences, &daggerheartv1.DaggerheartExperience{
				Name:     experienceName,
				Modifier: experience.Modifier,
			})
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{
				Experiences: experiences,
			}},
		}, nil
	}
	if step.DomainCards != nil {
		domainCardIDs := make([]string, 0, len(step.DomainCards.DomainCardIDs))
		for _, domainCardID := range step.DomainCards.DomainCardIDs {
			trimmedDomainCardID := strings.TrimSpace(domainCardID)
			if trimmedDomainCardID == "" {
				continue
			}
			domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
				DomainCardIds: domainCardIDs,
			}},
		}, nil
	}
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
			Connections: strings.TrimSpace(step.Connections.Connections),
		}},
	}, nil
}

// MapCampaignCharacterCreationStepToProto converts a domain step into the proto payload.
func MapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	return mapCampaignCharacterCreationStepToProto(step)
}

func (g GRPCGateway) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	if g.CharacterClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	_, err := g.CharacterClient.ResetCharacterCreationWorkflow(ctx, &statev1.ResetCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	return err
}
