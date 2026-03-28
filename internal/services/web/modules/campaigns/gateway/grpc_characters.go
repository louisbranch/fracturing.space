package gateway

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CampaignCharacters centralizes this web behavior in one helper seam.
func (g characterReadGateway) CampaignCharacters(ctx context.Context, campaignID string, options campaignapp.CharacterReadContext) ([]campaignapp.CampaignCharacter, error) {
	if g.read.Character == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignCharacter{}, nil
	}

	daggerheartSummariesByCharacterID, err := g.daggerheartCharacterSummaries(ctx, campaignID, options)
	if err != nil {
		return nil, err
	}
	participantLabels, err := g.characterParticipantLabels(ctx, campaignID, options)
	if err != nil {
		return nil, err
	}

	return grpcpaging.CollectPages[campaignapp.CampaignCharacter, *statev1.Character](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Character, string, error) {
			resp, err := g.read.Character.ListCharacters(ctx, &statev1.ListCharactersRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetCharacters(), resp.GetNextPageToken(), nil
		},
		func(character *statev1.Character) (campaignapp.CampaignCharacter, bool) {
			if character == nil {
				return campaignapp.CampaignCharacter{}, false
			}
			characterID := strings.TrimSpace(character.GetId())
			return g.mapCharacter(character, campaignID, participantLabels, daggerheartSummariesByCharacterID[characterID]), true
		},
	)
}

// CampaignCharacter loads one character entity for detail and edit flows
// without first materializing the full character collection.
func (g characterReadGateway) CampaignCharacter(ctx context.Context, campaignID string, characterID string, options campaignapp.CharacterReadContext) (campaignapp.CampaignCharacter, error) {
	if g.read.Character == nil {
		return campaignapp.CampaignCharacter{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacter{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.read.Character.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacter{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_load_character",
			FallbackMessage: "failed to load character",
		})
	}
	if resp == nil || resp.GetCharacter() == nil {
		return campaignapp.CampaignCharacter{}, apperrors.E(apperrors.KindNotFound, "character not found")
	}

	participantLabels, err := g.characterParticipantLabels(ctx, campaignID, options)
	if err != nil {
		return campaignapp.CampaignCharacter{}, err
	}
	var daggerheartSummary *campaignapp.CampaignCharacterDaggerheartSummary
	if summaryMap, err := g.daggerheartCharacterSummaries(ctx, campaignID, options); err != nil {
		return campaignapp.CampaignCharacter{}, err
	} else {
		daggerheartSummary = summaryMap[characterID]
	}
	return g.mapCharacter(resp.GetCharacter(), campaignID, participantLabels, daggerheartSummary), nil
}

// daggerheartCharacterSummaries batches profile and catalog reads so the
// characters page can render localized Daggerheart card summaries without N+1
// sheet requests.
func (g characterReadGateway) daggerheartCharacterSummaries(
	ctx context.Context,
	campaignID string,
	options campaignapp.CharacterReadContext,
) (map[string]*campaignapp.CampaignCharacterDaggerheartSummary, error) {
	if !strings.EqualFold(strings.TrimSpace(options.System), "Daggerheart") {
		return nil, nil
	}
	if g.read.DaggerheartContent == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}

	profiles, err := grpcpaging.CollectPages[*statev1.CharacterProfile, *statev1.CharacterProfile](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.CharacterProfile, string, error) {
			resp, err := g.read.Character.ListCharacterProfiles(ctx, &statev1.ListCharacterProfilesRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetProfiles(), resp.GetNextPageToken(), nil
		},
		func(profile *statev1.CharacterProfile) (*statev1.CharacterProfile, bool) {
			if profile == nil || profile.GetDaggerheart() == nil {
				return nil, false
			}
			if strings.TrimSpace(profile.GetCharacterId()) == "" {
				return nil, false
			}
			return profile, true
		},
	)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}

	locale := platformi18n.NormalizeLocale(platformi18n.LocaleForTag(options.Locale))
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	catalogResp, err := g.read.DaggerheartContent.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
	if err != nil {
		return nil, err
	}
	if catalogResp == nil || catalogResp.GetCatalog() == nil {
		return nil, nil
	}

	classNames, subclassNames, ancestryNames, communityNames := daggerheartCharacterCardNameMaps(catalogResp.GetCatalog())
	summaries := make(map[string]*campaignapp.CampaignCharacterDaggerheartSummary, len(profiles))
	for _, profile := range profiles {
		profileData := profile.GetDaggerheart()
		if profileData == nil || profileData.GetLevel() <= 0 {
			continue
		}

		className := strings.TrimSpace(classNames[strings.TrimSpace(profileData.GetClassId())])
		subclassName := strings.TrimSpace(subclassNames[strings.TrimSpace(profileData.GetSubclassId())])
		heritageName := daggerheartProfileHeritageName(profileData.GetHeritage(), ancestryNames)
		communityName := daggerheartProfileCommunityName(profileData.GetHeritage(), communityNames)
		if className == "" || subclassName == "" || heritageName == "" || communityName == "" {
			continue
		}

		characterID := strings.TrimSpace(profile.GetCharacterId())
		if characterID == "" {
			continue
		}
		summaries[characterID] = &campaignapp.CampaignCharacterDaggerheartSummary{
			Level:         profileData.GetLevel(),
			ClassName:     className,
			SubclassName:  subclassName,
			HeritageName:  heritageName,
			CommunityName: communityName,
		}
	}
	return summaries, nil
}

// daggerheartProfileHeritageName resolves the display heritage label from the stored selection.
func daggerheartProfileHeritageName(heritage *daggerheartv1.DaggerheartHeritageSelection, ancestryNames map[string]string) string {
	if heritage == nil {
		return ""
	}
	if label := strings.TrimSpace(heritage.GetAncestryLabel()); label != "" {
		return label
	}
	firstID := strings.TrimSpace(heritage.GetFirstFeatureAncestryId())
	secondID := strings.TrimSpace(heritage.GetSecondFeatureAncestryId())
	firstName := strings.TrimSpace(ancestryNames[firstID])
	secondName := strings.TrimSpace(ancestryNames[secondID])
	switch {
	case firstName == "":
		return ""
	case secondID == "" || secondID == firstID || secondName == "":
		return firstName
	case firstName == secondName:
		return firstName
	default:
		return firstName + " / " + secondName
	}
}

// daggerheartProfileCommunityName resolves the stored community display name.
func daggerheartProfileCommunityName(heritage *daggerheartv1.DaggerheartHeritageSelection, communityNames map[string]string) string {
	if heritage == nil {
		return ""
	}
	return strings.TrimSpace(communityNames[strings.TrimSpace(heritage.GetCommunityId())])
}

// daggerheartCharacterCardNameMaps builds the minimal localized lookup tables
// needed to resolve Daggerheart card summaries from stored profile IDs.
func daggerheartCharacterCardNameMaps(
	catalog *daggerheartv1.DaggerheartContentCatalog,
) (map[string]string, map[string]string, map[string]string, map[string]string) {
	classNames := make(map[string]string, len(catalog.GetClasses()))
	for _, class := range catalog.GetClasses() {
		if class == nil {
			continue
		}
		classID := strings.TrimSpace(class.GetId())
		className := strings.TrimSpace(class.GetName())
		if classID == "" || className == "" {
			continue
		}
		classNames[classID] = className
	}

	subclassNames := make(map[string]string, len(catalog.GetSubclasses()))
	for _, subclass := range catalog.GetSubclasses() {
		if subclass == nil {
			continue
		}
		subclassID := strings.TrimSpace(subclass.GetId())
		subclassName := strings.TrimSpace(subclass.GetName())
		if subclassID == "" || subclassName == "" {
			continue
		}
		subclassNames[subclassID] = subclassName
	}

	ancestryNames := map[string]string{}
	communityNames := map[string]string{}
	for _, heritage := range catalog.GetHeritages() {
		if heritage == nil {
			continue
		}
		heritageID := strings.TrimSpace(heritage.GetId())
		heritageName := strings.TrimSpace(heritage.GetName())
		if heritageID == "" || heritageName == "" {
			continue
		}
		switch daggerheartHeritageKindLabel(heritage.GetKind()) {
		case "ancestry":
			ancestryNames[heritageID] = heritageName
		case "community":
			communityNames[heritageID] = heritageName
		}
	}

	return classNames, subclassNames, ancestryNames, communityNames
}

// characterParticipantLabels keeps participant display names and viewer ownership lookup together for character mapping.
type characterParticipantLabels struct {
	namesByID           map[string]string
	viewerParticipantID string
}

// characterParticipantLabels loads participant labels once so entity and list character reads share viewer/owner mapping.
func (g characterReadGateway) characterParticipantLabels(ctx context.Context, campaignID string, options campaignapp.CharacterReadContext) (characterParticipantLabels, error) {
	labels := characterParticipantLabels{namesByID: map[string]string{}}
	if g.read.Participant == nil {
		return labels, nil
	}
	viewerUserID := strings.TrimSpace(options.ViewerUserID)
	type participantEntry struct {
		ID     string
		UserID string
		Name   string
	}
	entries, err := grpcpaging.CollectPages[participantEntry, *statev1.Participant](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Participant, string, error) {
			resp, err := g.read.Participant.ListParticipants(ctx, &statev1.ListParticipantsRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetParticipants(), resp.GetNextPageToken(), nil
		},
		func(p *statev1.Participant) (participantEntry, bool) {
			if p == nil {
				return participantEntry{}, false
			}
			id := strings.TrimSpace(p.GetId())
			if id == "" {
				return participantEntry{}, false
			}
			return participantEntry{
				ID:     id,
				UserID: strings.TrimSpace(p.GetUserId()),
				Name:   participantDisplayName(p),
			}, true
		},
	)
	if err != nil {
		return characterParticipantLabels{}, err
	}
	for _, entry := range entries {
		labels.namesByID[entry.ID] = entry.Name
		if labels.viewerParticipantID == "" && viewerUserID != "" && strings.EqualFold(strings.TrimSpace(entry.UserID), viewerUserID) {
			labels.viewerParticipantID = entry.ID
		}
	}
	return labels, nil
}

// mapCharacter translates one character row into the app-facing character view model.
func (g characterReadGateway) mapCharacter(
	character *statev1.Character,
	campaignID string,
	participantLabels characterParticipantLabels,
	daggerheartSummary *campaignapp.CampaignCharacterDaggerheartSummary,
) campaignapp.CampaignCharacter {
	if character == nil {
		return campaignapp.CampaignCharacter{}
	}
	characterID := strings.TrimSpace(character.GetId())
	avatarEntityID := characterID
	if avatarEntityID == "" {
		avatarEntityID = campaignID
	}
	ownerParticipantID := strings.TrimSpace(character.GetOwnerParticipantId().GetValue())
	ownerLabel := strings.TrimSpace(participantLabels.namesByID[ownerParticipantID])
	if ownerLabel == "" {
		if ownerParticipantID == "" {
			ownerLabel = "Unassigned"
		} else {
			ownerLabel = ownerParticipantID
		}
	}
	return campaignapp.CampaignCharacter{
		ID:                 characterID,
		Name:               characterDisplayName(character),
		Kind:               characterKindLabel(character.GetKind()),
		Owner:              ownerLabel,
		OwnerParticipantID: ownerParticipantID,
		Pronouns:           pronouns.FromProto(character.GetPronouns()),
		Aliases:            append([]string(nil), character.GetAliases()...),
		OwnedByViewer:      participantLabels.viewerParticipantID != "" && ownerParticipantID == participantLabels.viewerParticipantID,
		Daggerheart:        daggerheartSummary,
		AvatarURL: websupport.AvatarImageURL(
			g.assetBaseURL,
			catalog.AvatarRoleCharacter,
			avatarEntityID,
			strings.TrimSpace(character.GetAvatarSetId()),
			strings.TrimSpace(character.GetAvatarAssetId()),
			campaignAvatarCardDeliveryWidthPX,
		),
	}
}

// UpdateCharacter applies a character update via gRPC.
func (g characterMutationGateway) UpdateCharacter(ctx context.Context, campaignID string, characterID string, input campaignapp.UpdateCharacterInput) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}

	req := &statev1.UpdateCharacterRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		Name:        wrapperspb.String(name),
	}
	pronounsValue := strings.TrimSpace(input.Pronouns)
	if pronounsValue != "" {
		req.Pronouns = pronouns.ToProto(pronounsValue)
	}

	_, err := g.mutation.Character.UpdateCharacter(ctx, req)
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_update_character",
			FallbackMessage: "failed to update character",
		})
	}
	return nil
}

// DeleteCharacter applies a character deletion via gRPC.
func (g characterMutationGateway) DeleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.mutation.Character.DeleteCharacter(ctx, &statev1.DeleteCharacterRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_delete_character",
			FallbackMessage: "failed to delete character",
		})
	}
	return nil
}

// CreateCharacter executes package-scoped creation behavior for this flow.
func (g characterMutationGateway) CreateCharacter(ctx context.Context, campaignID string, input campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	if g.mutation.Character == nil {
		return campaignapp.CreateCharacterResult{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignapp.CreateCharacterResult{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return campaignapp.CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	kind := mapCharacterKindToProto(input.Kind)
	if kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		return campaignapp.CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}

	resp, err := g.mutation.Character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       kind,
		Pronouns:   pronouns.ToProto(strings.TrimSpace(input.Pronouns)),
	})
	if err != nil {
		return campaignapp.CreateCharacterResult{}, err
	}
	createdCharacterID := strings.TrimSpace(resp.GetCharacter().GetId())
	if createdCharacterID == "" {
		return campaignapp.CreateCharacterResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_character_id_was_empty", "created character id was empty")
	}
	return campaignapp.CreateCharacterResult{CharacterID: createdCharacterID}, nil
}
