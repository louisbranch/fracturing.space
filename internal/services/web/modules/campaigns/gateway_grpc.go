package campaigns

import (
	"context"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (g grpcGateway) CreateCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	locale := platformi18n.NormalizeLocale(input.Locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	resp, err := g.client.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:               input.Name,
		Locale:             locale,
		System:             input.System,
		GmMode:             input.GMMode,
		ThemePrompt:        input.ThemePrompt,
		CreatorDisplayName: input.CreatorDisplayName,
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
	})
	if err != nil {
		return campaignAuthorizationDecision{}, err
	}
	if resp == nil {
		return campaignAuthorizationDecision{}, nil
	}
	return campaignAuthorizationDecision{
		Evaluated:  true,
		Allowed:    resp.GetAllowed(),
		ReasonCode: strings.TrimSpace(resp.GetReasonCode()),
	}, nil
}

// FIXME(web-cutover): session/participant/character/invite mutations remain scaffolded while campaigns can be mounted as stable defaults.
func (g grpcGateway) StartSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign start session is not implemented")
}

func (g grpcGateway) EndSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign end session is not implemented")
}

func (g grpcGateway) UpdateParticipants(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign participant updates are not implemented")
}

func (g grpcGateway) CreateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign character creation is not implemented")
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
