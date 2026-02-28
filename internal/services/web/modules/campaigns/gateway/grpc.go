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
	"google.golang.org/grpc"
)

// CampaignClient exposes campaign listing, lookup, and creation from the game service.
type CampaignClient interface {
	ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error)
	GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error)
	CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error)
}

// ParticipantClient exposes participant listing for campaign workspace pages.
type ParticipantClient interface {
	ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error)
}

// CharacterClient exposes character operations for campaign workspace pages.
type CharacterClient interface {
	ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error)
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
	ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error)
	ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error)
}

// DaggerheartContentClient exposes Daggerheart content catalog operations.
type DaggerheartContentClient interface {
	GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error)
}

// SessionClient exposes session listing for campaign workspace pages.
type SessionClient interface {
	ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error)
}

// InviteClient exposes invite listing for campaign workspace pages.
type InviteClient interface {
	ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error)
}

// AuthorizationClient exposes campaign authorization checks.
type AuthorizationClient interface {
	Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error)
	BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error)
}

// GRPCGatewayDeps carries the explicit client dependencies for the campaigns gRPC gateway.
type GRPCGatewayDeps struct {
	CampaignClient           CampaignClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient DaggerheartContentClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	AuthorizationClient      AuthorizationClient
	AssetBaseURL             string
}

// NewGRPCGateway builds the production campaigns gateway from explicit client dependencies.
// All required clients must be present — a partial set would report healthy while
// individual campaign operations fail.
func NewGRPCGateway(deps GRPCGatewayDeps) campaignapp.CampaignGateway {
	if deps.CampaignClient == nil || deps.ParticipantClient == nil || deps.CharacterClient == nil ||
		deps.SessionClient == nil || deps.InviteClient == nil || deps.AuthorizationClient == nil {
		return campaignapp.NewUnavailableGateway()
	}
	return GRPCGateway{
		Client:              deps.CampaignClient,
		ParticipantClient:   deps.ParticipantClient,
		CharacterClient:     deps.CharacterClient,
		DaggerheartClient:   deps.DaggerheartContentClient,
		SessionClient:       deps.SessionClient,
		InviteClient:        deps.InviteClient,
		AuthorizationClient: deps.AuthorizationClient,
		AssetBaseURL:        deps.AssetBaseURL,
	}
}

type GRPCGateway struct {
	Client              CampaignClient
	ParticipantClient   ParticipantClient
	CharacterClient     CharacterClient
	DaggerheartClient   DaggerheartContentClient
	SessionClient       SessionClient
	InviteClient        InviteClient
	AuthorizationClient AuthorizationClient
	AssetBaseURL        string
}

func (g GRPCGateway) ListCampaigns(ctx context.Context) ([]campaignapp.CampaignSummary, error) {
	resp, err := g.Client.ListCampaigns(ctx, &statev1.ListCampaignsRequest{PageSize: 10})
	if err != nil {
		return nil, err
	}
	items := make([]campaignapp.CampaignSummary, 0, len(resp.GetCampaigns()))
	for _, campaign := range resp.GetCampaigns() {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		items = append(items, campaignapp.CampaignSummary{
			ID:                campaignID,
			Name:              name,
			Theme:             campaignapp.TruncateCampaignTheme(campaign.GetThemePrompt()),
			CoverImageURL:     campaignapp.CampaignCoverImageURL(g.AssetBaseURL, campaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
			ParticipantCount:  strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:    strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			CreatedAtUnixNano: campaignCreatedAtUnixNano(campaign),
			UpdatedAtUnixNano: campaignUpdatedAtUnixNano(campaign),
		})
	}
	return items, nil
}

func (g GRPCGateway) CampaignName(ctx context.Context, campaignID string) (string, error) {
	resp, err := g.Client.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return "", err
	}
	if resp.GetCampaign() == nil {
		return "", nil
	}
	return strings.TrimSpace(resp.GetCampaign().GetName()), nil
}

func (g GRPCGateway) CampaignWorkspace(ctx context.Context, campaignID string) (campaignapp.CampaignWorkspace, error) {
	resp, err := g.Client.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return campaignapp.CampaignWorkspace{}, err
	}
	if resp.GetCampaign() == nil {
		return campaignapp.CampaignWorkspace{}, apperrors.E(apperrors.KindNotFound, "campaign not found")
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
	return campaignapp.CampaignWorkspace{
		ID:               resolvedCampaignID,
		Name:             name,
		Theme:            strings.TrimSpace(campaign.GetThemePrompt()),
		System:           campaignSystemLabel(campaign.GetSystem()),
		GMMode:           campaignGMModeLabel(campaign.GetGmMode()),
		Status:           campaignStatusLabel(campaign.GetStatus()),
		Locale:           campaignLocaleLabel(campaign.GetLocale()),
		Intent:           campaignIntentLabel(campaign.GetIntent()),
		AccessPolicy:     campaignAccessPolicyLabel(campaign.GetAccessPolicy()),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		CoverImageURL:    campaignapp.CampaignCoverImageURL(g.AssetBaseURL, resolvedCampaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
	}, nil
}

func (g GRPCGateway) CreateCampaign(ctx context.Context, input campaignapp.CreateCampaignInput) (campaignapp.CreateCampaignResult, error) {
	locale := platformi18n.LocaleForTag(input.Locale)
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	resp, err := g.Client.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:        input.Name,
		Locale:      locale,
		System:      mapGameSystemToProto(input.System),
		GmMode:      mapGmModeToProto(input.GMMode),
		ThemePrompt: input.ThemePrompt,
	})
	if err != nil {
		return campaignapp.CreateCampaignResult{}, err
	}
	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		return campaignapp.CreateCampaignResult{}, apperrors.E(apperrors.KindUnknown, "created campaign id was empty")
	}
	return campaignapp.CreateCampaignResult{CampaignID: campaignID}, nil
}
