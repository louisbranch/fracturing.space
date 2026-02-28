package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
)

func (g GRPCGateway) CampaignCharacters(ctx context.Context, campaignID string) ([]campaignapp.CampaignCharacter, error) {
	if g.CharacterClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignCharacter{}, nil
	}

	// Collect participant names so character controller labels can be resolved.
	type participantEntry struct {
		ID   string
		Name string
	}
	participantNamesByID := map[string]string{}
	if g.ParticipantClient != nil {
		entries, err := grpcpaging.CollectPages[participantEntry, *statev1.Participant](
			ctx, 10,
			func(ctx context.Context, pageToken string) ([]*statev1.Participant, string, error) {
				resp, err := g.ParticipantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
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
			resp, err := g.CharacterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
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
				ID:         characterID,
				Name:       characterDisplayName(character),
				Kind:       characterKindLabel(character.GetKind()),
				Controller: controllerLabel,
				Pronouns:   pronouns.FromProto(character.GetPronouns()),
				Aliases:    append([]string(nil), character.GetAliases()...),
				AvatarURL: websupport.AvatarImageURL(
					g.AssetBaseURL,
					catalog.AvatarRoleCharacter,
					avatarEntityID,
					strings.TrimSpace(character.GetAvatarSetId()),
					strings.TrimSpace(character.GetAvatarAssetId()),
				),
			}, true
		},
	)
}

func (g GRPCGateway) CreateCharacter(ctx context.Context, campaignID string, input campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	if g.CharacterClient == nil {
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

	resp, err := g.CharacterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       kind,
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

// TODO(mutation-activation): see gateway_grpc_sessions.go for activation criteria.
func (g GRPCGateway) UpdateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign character updates are not implemented")
}

func (g GRPCGateway) ControlCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign character control is not implemented")
}
