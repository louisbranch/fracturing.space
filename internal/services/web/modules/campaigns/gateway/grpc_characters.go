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
func (g GRPCGateway) CampaignCharacters(ctx context.Context, campaignID string, options campaignapp.CampaignCharactersReadOptions) ([]campaignapp.CampaignCharacter, error) {
	if g.Read.Character == nil {
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

	// Collect participant names so character controller labels can be resolved.
	type participantEntry struct {
		ID   string
		Name string
	}
	participantNamesByID := map[string]string{}
	if g.Read.Participant != nil {
		entries, err := grpcpaging.CollectPages[participantEntry, *statev1.Participant](
			ctx, 10,
			func(ctx context.Context, pageToken string) ([]*statev1.Participant, string, error) {
				resp, err := g.Read.Participant.ListParticipants(ctx, &statev1.ListParticipantsRequest{
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
				return participantEntry{ID: id, Name: participantDisplayName(p)}, true
			},
		)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			participantNamesByID[e.ID] = e.Name
		}
	}

	return grpcpaging.CollectPages[campaignapp.CampaignCharacter, *statev1.Character](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Character, string, error) {
			resp, err := g.Read.Character.ListCharacters(ctx, &statev1.ListCharactersRequest{
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
			return campaignapp.CampaignCharacter{
				ID:                      characterID,
				Name:                    characterDisplayName(character),
				Kind:                    characterKindLabel(character.GetKind()),
				Controller:              controllerLabel,
				ControllerParticipantID: controllerParticipantID,
				Pronouns:                pronouns.FromProto(character.GetPronouns()),
				Aliases:                 append([]string(nil), character.GetAliases()...),
				Daggerheart:             daggerheartSummariesByCharacterID[characterID],
				AvatarURL: websupport.AvatarImageURL(
					g.AssetBaseURL,
					catalog.AvatarRoleCharacter,
					avatarEntityID,
					strings.TrimSpace(character.GetAvatarSetId()),
					strings.TrimSpace(character.GetAvatarAssetId()),
					campaignAvatarCardDeliveryWidthPX,
				),
			}, true
		},
	)
}

// daggerheartCharacterSummaries batches profile and catalog reads so the
// characters page can render localized Daggerheart card summaries without N+1
// sheet requests.
func (g GRPCGateway) daggerheartCharacterSummaries(
	ctx context.Context,
	campaignID string,
	options campaignapp.CampaignCharactersReadOptions,
) (map[string]*campaignapp.CampaignCharacterDaggerheartSummary, error) {
	if !strings.EqualFold(strings.TrimSpace(options.System), "Daggerheart") {
		return nil, nil
	}
	if g.Read.DaggerheartContent == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}

	profiles, err := grpcpaging.CollectPages[*statev1.CharacterProfile, *statev1.CharacterProfile](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.CharacterProfile, string, error) {
			resp, err := g.Read.Character.ListCharacterProfiles(ctx, &statev1.ListCharacterProfilesRequest{
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
	catalogResp, err := g.Read.DaggerheartContent.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
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
		ancestryName := strings.TrimSpace(ancestryNames[strings.TrimSpace(profileData.GetAncestryId())])
		communityName := strings.TrimSpace(communityNames[strings.TrimSpace(profileData.GetCommunityId())])
		if className == "" || subclassName == "" || ancestryName == "" || communityName == "" {
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
			AncestryName:  ancestryName,
			CommunityName: communityName,
		}
	}
	return summaries, nil
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

// UpdateCharacter applies a character update via gRPC.
func (g GRPCGateway) UpdateCharacter(ctx context.Context, campaignID string, characterID string, input campaignapp.UpdateCharacterInput) error {
	if g.Mutation.Character == nil {
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

	_, err := g.Mutation.Character.UpdateCharacter(ctx, req)
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
func (g GRPCGateway) DeleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	if g.Mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.Mutation.Character.DeleteCharacter(ctx, &statev1.DeleteCharacterRequest{
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

// SetCharacterController applies an explicit controller update via gRPC.
func (g GRPCGateway) SetCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	if g.Mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.Mutation.Character.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(strings.TrimSpace(participantID)),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_set_character_controller",
			FallbackMessage: "failed to set character controller",
		})
	}
	return nil
}

// ClaimCharacterControl claims character control via gRPC.
func (g GRPCGateway) ClaimCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if g.Mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.Mutation.Character.ClaimCharacterControl(ctx, &statev1.ClaimCharacterControlRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_claim_character_control",
			FallbackMessage: "failed to claim character control",
		})
	}
	return nil
}

// ReleaseCharacterControl releases character control via gRPC.
func (g GRPCGateway) ReleaseCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if g.Mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.Mutation.Character.ReleaseCharacterControl(ctx, &statev1.ReleaseCharacterControlRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_release_character_control",
			FallbackMessage: "failed to release character control",
		})
	}
	return nil
}

// CreateCharacter executes package-scoped creation behavior for this flow.
func (g GRPCGateway) CreateCharacter(ctx context.Context, campaignID string, input campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	if g.Mutation.Character == nil {
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

	resp, err := g.Mutation.Character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
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
