package campaigns

import (
	"context"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// NewGRPCGateway builds the production campaigns gateway from shared dependencies.
func NewGRPCGateway(deps module.Dependencies) CampaignGateway {
	if deps.CampaignClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{
		client:              deps.CampaignClient,
		participantClient:   deps.ParticipantClient,
		characterClient:     deps.CharacterClient,
		daggerheartClient:   deps.DaggerheartContentClient,
		sessionClient:       deps.SessionClient,
		inviteClient:        deps.InviteClient,
		authorizationClient: deps.AuthorizationClient,
		assetBaseURL:        deps.AssetBaseURL,
	}
}

type grpcGateway struct {
	client              module.CampaignClient
	participantClient   module.ParticipantClient
	characterClient     module.CharacterClient
	daggerheartClient   module.DaggerheartContentClient
	sessionClient       module.SessionClient
	inviteClient        module.InviteClient
	authorizationClient module.AuthorizationClient
	assetBaseURL        string
}

func (g grpcGateway) ListCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	resp, err := g.client.ListCampaigns(ctx, &statev1.ListCampaignsRequest{PageSize: 10})
	if err != nil {
		return nil, err
	}
	items := make([]CampaignSummary, 0, len(resp.GetCampaigns()))
	for _, campaign := range resp.GetCampaigns() {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		items = append(items, CampaignSummary{
			ID:                campaignID,
			Name:              name,
			Theme:             truncateCampaignTheme(campaign.GetThemePrompt()),
			CoverImageURL:     campaignCoverImageURL(g.assetBaseURL, campaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
			ParticipantCount:  strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:    strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			CreatedAtUnixNano: campaignCreatedAtUnixNano(campaign),
		})
	}
	return items, nil
}

func (g grpcGateway) CampaignName(ctx context.Context, campaignID string) (string, error) {
	resp, err := g.client.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return "", err
	}
	if resp.GetCampaign() == nil {
		return "", nil
	}
	return strings.TrimSpace(resp.GetCampaign().GetName()), nil
}

func (g grpcGateway) CampaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	resp, err := g.client.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return CampaignWorkspace{}, err
	}
	if resp.GetCampaign() == nil {
		return CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
	}
	campaign := resp.GetCampaign()
	resolvedCampaignID := strings.TrimSpace(campaign.GetId())
	if resolvedCampaignID == "" {
		resolvedCampaignID = strings.TrimSpace(campaignID)
	}
	name := strings.TrimSpace(campaign.GetName())
	if name == "" {
		name = resolvedCampaignID
	}
	return CampaignWorkspace{
		ID:            resolvedCampaignID,
		Name:          name,
		Theme:         strings.TrimSpace(campaign.GetThemePrompt()),
		System:        campaignSystemLabel(campaign.GetSystem()),
		GMMode:        campaignGMModeLabel(campaign.GetGmMode()),
		CoverImageURL: campaignCoverImageURL(g.assetBaseURL, resolvedCampaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
	}, nil
}

func (g grpcGateway) CampaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	if g.participantClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignParticipant{}, nil
	}

	participants := make([]CampaignParticipant, 0)
	pageToken := ""
	for {
		resp, err := g.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			participantID := strings.TrimSpace(participant.GetId())
			avatarEntityID := participantID
			if avatarEntityID == "" {
				avatarEntityID = strings.TrimSpace(participant.GetUserId())
			}
			if avatarEntityID == "" {
				avatarEntityID = campaignID
			}
			participants = append(participants, CampaignParticipant{
				ID:             participantID,
				UserID:         strings.TrimSpace(participant.GetUserId()),
				Name:           participantDisplayName(participant),
				Role:           participantRoleLabel(participant.GetRole()),
				CampaignAccess: participantCampaignAccessLabel(participant.GetCampaignAccess()),
				Controller:     participantControllerLabel(participant.GetController()),
				Pronouns:       strings.TrimSpace(participant.GetPronouns()),
				AvatarURL: websupport.AvatarImageURL(
					g.assetBaseURL,
					catalog.AvatarRoleParticipant,
					avatarEntityID,
					strings.TrimSpace(participant.GetAvatarSetId()),
					strings.TrimSpace(participant.GetAvatarAssetId()),
				),
			})
		}

		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return participants, nil
}

func (g grpcGateway) CampaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	if g.characterClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignCharacter{}, nil
	}

	participantNamesByID := map[string]string{}
	if g.participantClient != nil {
		participantPageToken := ""
		for {
			participantResp, err := g.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  participantPageToken,
			})
			if err != nil {
				return nil, err
			}
			if participantResp == nil {
				break
			}

			for _, participant := range participantResp.GetParticipants() {
				if participant == nil {
					continue
				}
				participantID := strings.TrimSpace(participant.GetId())
				if participantID == "" {
					continue
				}
				participantNamesByID[participantID] = participantDisplayName(participant)
			}

			nextToken := strings.TrimSpace(participantResp.GetNextPageToken())
			if nextToken == "" {
				break
			}
			participantPageToken = nextToken
		}
	}

	characters := make([]CampaignCharacter, 0)
	pageToken := ""
	for {
		resp, err := g.characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for _, character := range resp.GetCharacters() {
			if character == nil {
				continue
			}
			characterID := strings.TrimSpace(character.GetId())
			avatarEntityID := characterID
			if avatarEntityID == "" {
				avatarEntityID = campaignID
			}
			controllerParticipantID := strings.TrimSpace(character.GetParticipantId().GetValue())
			controllerLabel := strings.TrimSpace(participantNamesByID[controllerParticipantID])
			if controllerLabel == "" {
				if controllerParticipantID == "" {
					controllerLabel = "Unassigned"
				} else {
					controllerLabel = controllerParticipantID
				}
			}

			characters = append(characters, CampaignCharacter{
				ID:         characterID,
				Name:       characterDisplayName(character),
				Kind:       characterKindLabel(character.GetKind()),
				Controller: controllerLabel,
				Pronouns:   strings.TrimSpace(character.GetPronouns()),
				Aliases:    append([]string(nil), character.GetAliases()...),
				AvatarURL: websupport.AvatarImageURL(
					g.assetBaseURL,
					catalog.AvatarRoleCharacter,
					avatarEntityID,
					strings.TrimSpace(character.GetAvatarSetId()),
					strings.TrimSpace(character.GetAvatarAssetId()),
				),
			})
		}

		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return characters, nil
}

func (g grpcGateway) CampaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	if g.sessionClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.session_service_client_is_not_configured", "session service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignSession{}, nil
	}

	sessions := make([]CampaignSession, 0)
	pageToken := ""
	for {
		resp, err := g.sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for _, session := range resp.GetSessions() {
			if session == nil {
				continue
			}
			sessions = append(sessions, CampaignSession{
				ID:        strings.TrimSpace(session.GetId()),
				Name:      strings.TrimSpace(session.GetName()),
				Status:    sessionStatusLabel(session.GetStatus()),
				StartedAt: timestampString(session.GetStartedAt()),
				UpdatedAt: timestampString(session.GetUpdatedAt()),
				EndedAt:   timestampString(session.GetEndedAt()),
			})
		}

		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return sessions, nil
}

func (g grpcGateway) CampaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	if g.inviteClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignInvite{}, nil
	}

	invites := make([]CampaignInvite, 0)
	pageToken := ""
	for {
		resp, err := g.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for _, invite := range resp.GetInvites() {
			if invite == nil {
				continue
			}
			invites = append(invites, CampaignInvite{
				ID:              strings.TrimSpace(invite.GetId()),
				ParticipantID:   strings.TrimSpace(invite.GetParticipantId()),
				RecipientUserID: strings.TrimSpace(invite.GetRecipientUserId()),
				Status:          inviteStatusLabel(invite.GetStatus()),
			})
		}

		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return invites, nil
}

func (g grpcGateway) CharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	if g.characterClient == nil {
		return CampaignCharacterCreationProgress{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.characterClient.GetCharacterCreationProgress(ctx, &statev1.GetCharacterCreationProgressRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return CampaignCharacterCreationProgress{}, err
	}
	if resp == nil || resp.GetProgress() == nil {
		return CampaignCharacterCreationProgress{Steps: []CampaignCharacterCreationStep{}, UnmetReasons: []string{}}, nil
	}

	progress := resp.GetProgress()
	steps := make([]CampaignCharacterCreationStep, 0, len(progress.GetSteps()))
	for _, step := range progress.GetSteps() {
		if step == nil {
			continue
		}
		steps = append(steps, CampaignCharacterCreationStep{
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
	return CampaignCharacterCreationProgress{
		Steps:        steps,
		NextStep:     progress.GetNextStep(),
		Ready:        progress.GetReady(),
		UnmetReasons: unmetReasons,
	}, nil
}

func (g grpcGateway) CharacterCreationCatalog(ctx context.Context, locale commonv1.Locale) (CampaignCharacterCreationCatalog, error) {
	if g.daggerheartClient == nil {
		return CampaignCharacterCreationCatalog{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}

	resp, err := g.daggerheartClient.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
	if err != nil {
		return CampaignCharacterCreationCatalog{}, err
	}
	if resp == nil || resp.GetCatalog() == nil {
		return CampaignCharacterCreationCatalog{}, nil
	}

	catalogResp := resp.GetCatalog()
	catalog := CampaignCharacterCreationCatalog{}

	catalog.Classes = make([]DaggerheartCreationClass, 0, len(catalogResp.GetClasses()))
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
		catalog.Classes = append(catalog.Classes, DaggerheartCreationClass{
			ID:        classID,
			Name:      strings.TrimSpace(class.GetName()),
			DomainIDs: domainIDs,
		})
	}

	catalog.Subclasses = make([]DaggerheartCreationSubclass, 0, len(catalogResp.GetSubclasses()))
	for _, subclass := range catalogResp.GetSubclasses() {
		if subclass == nil {
			continue
		}
		subclassID := strings.TrimSpace(subclass.GetId())
		if subclassID == "" {
			continue
		}
		catalog.Subclasses = append(catalog.Subclasses, DaggerheartCreationSubclass{
			ID:      subclassID,
			Name:    strings.TrimSpace(subclass.GetName()),
			ClassID: strings.TrimSpace(subclass.GetClassId()),
		})
	}

	catalog.Heritages = make([]DaggerheartCreationHeritage, 0, len(catalogResp.GetHeritages()))
	for _, heritage := range catalogResp.GetHeritages() {
		if heritage == nil {
			continue
		}
		heritageID := strings.TrimSpace(heritage.GetId())
		if heritageID == "" {
			continue
		}
		catalog.Heritages = append(catalog.Heritages, DaggerheartCreationHeritage{
			ID:   heritageID,
			Name: strings.TrimSpace(heritage.GetName()),
			Kind: daggerheartHeritageKindLabel(heritage.GetKind()),
		})
	}

	catalog.Weapons = make([]DaggerheartCreationWeapon, 0, len(catalogResp.GetWeapons()))
	for _, weapon := range catalogResp.GetWeapons() {
		if weapon == nil {
			continue
		}
		weaponID := strings.TrimSpace(weapon.GetId())
		if weaponID == "" {
			continue
		}
		catalog.Weapons = append(catalog.Weapons, DaggerheartCreationWeapon{
			ID:       weaponID,
			Name:     strings.TrimSpace(weapon.GetName()),
			Category: daggerheartWeaponCategoryLabel(weapon.GetCategory()),
			Tier:     weapon.GetTier(),
		})
	}

	catalog.Armor = make([]DaggerheartCreationArmor, 0, len(catalogResp.GetArmor()))
	for _, armor := range catalogResp.GetArmor() {
		if armor == nil {
			continue
		}
		armorID := strings.TrimSpace(armor.GetId())
		if armorID == "" {
			continue
		}
		catalog.Armor = append(catalog.Armor, DaggerheartCreationArmor{
			ID:   armorID,
			Name: strings.TrimSpace(armor.GetName()),
			Tier: armor.GetTier(),
		})
	}

	catalog.Items = make([]DaggerheartCreationItem, 0, len(catalogResp.GetItems()))
	for _, item := range catalogResp.GetItems() {
		if item == nil {
			continue
		}
		itemID := strings.TrimSpace(item.GetId())
		if itemID == "" {
			continue
		}
		catalog.Items = append(catalog.Items, DaggerheartCreationItem{
			ID:   itemID,
			Name: strings.TrimSpace(item.GetName()),
		})
	}

	catalog.DomainCards = make([]DaggerheartCreationDomainCard, 0, len(catalogResp.GetDomainCards()))
	for _, domainCard := range catalogResp.GetDomainCards() {
		if domainCard == nil {
			continue
		}
		domainCardID := strings.TrimSpace(domainCard.GetId())
		if domainCardID == "" {
			continue
		}
		catalog.DomainCards = append(catalog.DomainCards, DaggerheartCreationDomainCard{
			ID:       domainCardID,
			Name:     strings.TrimSpace(domainCard.GetName()),
			DomainID: strings.TrimSpace(domainCard.GetDomainId()),
			Level:    domainCard.GetLevel(),
		})
	}

	return catalog, nil
}

func (g grpcGateway) CharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProfile, error) {
	if g.characterClient == nil {
		return CampaignCharacterCreationProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return CampaignCharacterCreationProfile{}, err
	}
	if resp == nil || resp.GetProfile() == nil || resp.GetProfile().GetDaggerheart() == nil {
		return CampaignCharacterCreationProfile{}, nil
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

	return CampaignCharacterCreationProfile{
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

func (g grpcGateway) CreateCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	locale := platformi18n.NormalizeLocale(input.Locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	resp, err := g.client.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:        input.Name,
		Locale:      locale,
		System:      input.System,
		GmMode:      input.GMMode,
		ThemePrompt: input.ThemePrompt,
	})
	if err != nil {
		return CreateCampaignResult{}, err
	}
	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		return CreateCampaignResult{}, apperrors.E(apperrors.KindUnknown, "created campaign id was empty")
	}
	return CreateCampaignResult{CampaignID: campaignID}, nil
}

func (g grpcGateway) CanCampaignAction(
	ctx context.Context,
	campaignID string,
	action statev1.AuthorizationAction,
	resource statev1.AuthorizationResource,
	target *statev1.AuthorizationTarget,
) (campaignAuthorizationDecision, error) {
	if g.authorizationClient == nil {
		return campaignAuthorizationDecision{}, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignAuthorizationDecision{}, nil
	}
	resp, err := g.authorizationClient.Can(ctx, &statev1.CanRequest{
		CampaignId: campaignID,
		Action:     action,
		Resource:   resource,
		Target:     target,
	})
	if err != nil {
		return campaignAuthorizationDecision{}, err
	}
	if resp == nil {
		return campaignAuthorizationDecision{}, nil
	}
	return campaignAuthorizationDecision{
		CheckID:    "",
		Evaluated:  true,
		Allowed:    resp.GetAllowed(),
		ReasonCode: strings.TrimSpace(resp.GetReasonCode()),
	}, nil
}

func (g grpcGateway) BatchCanCampaignAction(
	ctx context.Context,
	campaignID string,
	checks []campaignAuthorizationCheck,
) ([]campaignAuthorizationDecision, error) {
	if g.authorizationClient == nil {
		return nil, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || len(checks) == 0 {
		return nil, nil
	}

	protoChecks := make([]*statev1.BatchCanCheck, 0, len(checks))
	for _, check := range checks {
		protoChecks = append(protoChecks, &statev1.BatchCanCheck{
			CheckId:    strings.TrimSpace(check.CheckID),
			CampaignId: campaignID,
			Action:     check.Action,
			Resource:   check.Resource,
			Target:     check.Target,
		})
	}

	resp, err := g.authorizationClient.BatchCan(ctx, &statev1.BatchCanRequest{Checks: protoChecks})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	results := resp.GetResults()
	decisions := make([]campaignAuthorizationDecision, 0, len(results))
	for idx, result := range results {
		if result == nil {
			fallbackCheckID := ""
			if idx < len(checks) {
				fallbackCheckID = strings.TrimSpace(checks[idx].CheckID)
			}
			decisions = append(decisions, campaignAuthorizationDecision{CheckID: fallbackCheckID})
			continue
		}
		checkID := strings.TrimSpace(result.GetCheckId())
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		decisions = append(decisions, campaignAuthorizationDecision{
			CheckID:    checkID,
			Evaluated:  true,
			Allowed:    result.GetAllowed(),
			ReasonCode: strings.TrimSpace(result.GetReasonCode()),
		})
	}

	return decisions, nil
}

func (g grpcGateway) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *daggerheartv1.DaggerheartCreationStepInput) error {
	if g.characterClient == nil {
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
	_, err := g.characterClient.ApplyCharacterCreationStep(ctx, &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStep:  &statev1.ApplyCharacterCreationStepRequest_Daggerheart{Daggerheart: step},
	})
	return err
}

func (g grpcGateway) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	if g.characterClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	_, err := g.characterClient.ResetCharacterCreationWorkflow(ctx, &statev1.ResetCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	return err
}

// FIXME(web-cutover): session/participant/invite mutations remain scaffolded while campaigns can be mounted as stable defaults.
func (g grpcGateway) StartSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign start session is not implemented")
}

func (g grpcGateway) EndSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign end session is not implemented")
}

func (g grpcGateway) UpdateParticipants(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign participant updates are not implemented")
}

func (g grpcGateway) CreateCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	if g.characterClient == nil {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CreateCharacterResult{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	if input.Kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}

	resp, err := g.characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       input.Kind,
	})
	if err != nil {
		return CreateCharacterResult{}, err
	}
	createdCharacterID := strings.TrimSpace(resp.GetCharacter().GetId())
	if createdCharacterID == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_character_id_was_empty", "created character id was empty")
	}
	return CreateCharacterResult{CharacterID: createdCharacterID}, nil
}

func (g grpcGateway) UpdateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign character updates are not implemented")
}

func (g grpcGateway) ControlCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign character control is not implemented")
}

func (g grpcGateway) CreateInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite creation is not implemented")
}

func (g grpcGateway) RevokeInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite revocation is not implemented")
}

func campaignSystemLabel(system commonv1.GameSystem) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "Daggerheart"
	default:
		return "Unspecified"
	}
}

func campaignGMModeLabel(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "Human"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "Hybrid"
	default:
		return "Unspecified"
	}
}

func participantDisplayName(participant *statev1.Participant) string {
	if participant == nil {
		return "Unknown participant"
	}
	if name := strings.TrimSpace(participant.GetName()); name != "" {
		return name
	}
	if userID := strings.TrimSpace(participant.GetUserId()); userID != "" {
		return userID
	}
	if participantID := strings.TrimSpace(participant.GetId()); participantID != "" {
		return participantID
	}
	return "Unknown participant"
}

func participantRoleLabel(role statev1.ParticipantRole) string {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM"
	case statev1.ParticipantRole_PLAYER:
		return "Player"
	default:
		return "Unspecified"
	}
}

func participantCampaignAccessLabel(access statev1.CampaignAccess) string {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return "Member"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return "Manager"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return "Owner"
	default:
		return "Unspecified"
	}
}

func participantControllerLabel(controller statev1.Controller) string {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "Human"
	case statev1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "Unspecified"
	}
}

func characterDisplayName(character *statev1.Character) string {
	if character == nil {
		return "Unknown character"
	}
	if name := strings.TrimSpace(character.GetName()); name != "" {
		return name
	}
	if characterID := strings.TrimSpace(character.GetId()); characterID != "" {
		return characterID
	}
	return "Unknown character"
}

func characterKindLabel(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "Unspecified"
	}
}

func sessionStatusLabel(status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return "Active"
	case statev1.SessionStatus_SESSION_ENDED:
		return "Ended"
	default:
		return "Unspecified"
	}
}

func inviteStatusLabel(status statev1.InviteStatus) string {
	switch status {
	case statev1.InviteStatus_PENDING:
		return "Pending"
	case statev1.InviteStatus_CLAIMED:
		return "Claimed"
	case statev1.InviteStatus_REVOKED:
		return "Revoked"
	default:
		return "Unspecified"
	}
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return strings.TrimSpace(ts.AsTime().UTC().Format("2006-01-02 15:04 UTC"))
}

func int32ValueString(value *wrapperspb.Int32Value) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(value.GetValue()), 10)
}

func daggerheartHeritageKindLabel(kind daggerheartv1.DaggerheartHeritageKind) string {
	switch kind {
	case daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY:
		return "ancestry"
	case daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY:
		return "community"
	default:
		return ""
	}
}

func daggerheartWeaponCategoryLabel(category daggerheartv1.DaggerheartWeaponCategory) string {
	switch category {
	case daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY:
		return "primary"
	case daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY:
		return "secondary"
	default:
		return ""
	}
}
