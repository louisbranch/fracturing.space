package gateway

import (
	"context"
	"strconv"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CampaignClient exposes discovery entry, lookup, and creation from the game service.
type CampaignClient interface {
	ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error)
	GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error)
	GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error)
	CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error)
	UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error)
	SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error)
	ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error)
}

// AgentClient exposes AI agent listing used for owner-only campaign binding UX.
type AgentClient interface {
	ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error)
}

// ParticipantClient exposes participant listing for campaign workspace pages.
type ParticipantClient interface {
	ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error)
	GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error)
	CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error)
	UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error)
}

// CharacterClient exposes character operations for campaign workspace pages.
type CharacterClient interface {
	ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error)
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error)
	DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error)
	SetDefaultControl(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error)
	ClaimCharacterControl(context.Context, *statev1.ClaimCharacterControlRequest, ...grpc.CallOption) (*statev1.ClaimCharacterControlResponse, error)
	ReleaseCharacterControl(context.Context, *statev1.ReleaseCharacterControlRequest, ...grpc.CallOption) (*statev1.ReleaseCharacterControlResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
	ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error)
	ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error)
}

// DaggerheartContentClient exposes Daggerheart content catalog operations.
type DaggerheartContentClient interface {
	GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error)
}

// DaggerheartAssetClient exposes Daggerheart content-asset map operations.
type DaggerheartAssetClient interface {
	GetAssetMap(context.Context, *daggerheartv1.GetDaggerheartAssetMapRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartAssetMapResponse, error)
}

// SessionClient exposes session listing for campaign workspace pages.
type SessionClient interface {
	ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error)
	StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error)
	EndSession(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error)
}

// InviteClient exposes invite listing for campaign workspace pages.
type InviteClient interface {
	ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error)
	CreateInvite(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error)
	RevokeInvite(context.Context, *statev1.RevokeInviteRequest, ...grpc.CallOption) (*statev1.RevokeInviteResponse, error)
}

// AuthClient resolves auth-owned users from usernames for invite targeting.
type AuthClient interface {
	LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error)
}

// AuthorizationClient exposes campaign authorization checks.
type AuthorizationClient interface {
	Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error)
	BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error)
}

// GRPCGatewayDeps carries the explicit client dependencies for the campaigns gRPC gateway.
type GRPCGatewayDeps struct {
	CampaignClient           CampaignClient
	AgentClient              AgentClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient DaggerheartContentClient
	DaggerheartAssetClient   DaggerheartAssetClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	AuthClient               AuthClient
	AuthorizationClient      AuthorizationClient
	AssetBaseURL             string
}

// NewGRPCGateway builds the production campaigns gateway from explicit client dependencies.
// All required clients must be present — a partial set would report healthy while
// individual campaign operations fail.
func NewGRPCGateway(deps GRPCGatewayDeps) campaignapp.CampaignGateway {
	if deps.CampaignClient == nil || deps.ParticipantClient == nil || deps.CharacterClient == nil ||
		deps.DaggerheartContentClient == nil || deps.DaggerheartAssetClient == nil ||
		deps.SessionClient == nil || deps.InviteClient == nil || deps.AuthClient == nil || deps.AuthorizationClient == nil {
		return campaignapp.NewUnavailableGateway()
	}
	return GRPCGateway{
		Client:              deps.CampaignClient,
		AgentClient:         deps.AgentClient,
		ParticipantClient:   deps.ParticipantClient,
		CharacterClient:     deps.CharacterClient,
		DaggerheartContent:  deps.DaggerheartContentClient,
		DaggerheartAsset:    deps.DaggerheartAssetClient,
		SessionClient:       deps.SessionClient,
		InviteClient:        deps.InviteClient,
		AuthClient:          deps.AuthClient,
		AuthorizationClient: deps.AuthorizationClient,
		AssetBaseURL:        deps.AssetBaseURL,
	}
}

// GRPCGateway defines an internal contract used at this web package boundary.
type GRPCGateway struct {
	Client              CampaignClient
	AgentClient         AgentClient
	ParticipantClient   ParticipantClient
	CharacterClient     CharacterClient
	DaggerheartContent  DaggerheartContentClient
	DaggerheartAsset    DaggerheartAssetClient
	SessionClient       SessionClient
	InviteClient        InviteClient
	AuthClient          AuthClient
	AuthorizationClient AuthorizationClient
	AssetBaseURL        string
}

// ListCampaigns returns the package view collection for this workflow.
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

// CampaignName centralizes this web behavior in one helper seam.
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

// CampaignWorkspace centralizes this web behavior in one helper seam.
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
		AIAgentID:        strings.TrimSpace(campaign.GetAiAgentId()),
		Status:           campaignStatusLabel(campaign.GetStatus()),
		Locale:           campaignLocaleLabel(campaign.GetLocale()),
		Intent:           campaignIntentLabel(campaign.GetIntent()),
		AccessPolicy:     campaignAccessPolicyLabel(campaign.GetAccessPolicy()),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		CoverImageURL:    campaignapp.CampaignCoverImageURL(g.AssetBaseURL, resolvedCampaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
	}, nil
}

// CreateCampaign executes package-scoped creation behavior for this flow.
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

// UpdateCampaign applies this package workflow transition.
func (g GRPCGateway) UpdateCampaign(ctx context.Context, campaignID string, input campaignapp.UpdateCampaignInput) error {
	if g.Client == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "campaign service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	req := &statev1.UpdateCampaignRequest{CampaignId: campaignID}
	if input.Name != nil {
		req.Name = wrapperspb.String(strings.TrimSpace(*input.Name))
	}
	if input.ThemePrompt != nil {
		req.ThemePrompt = wrapperspb.String(strings.TrimSpace(*input.ThemePrompt))
	}
	if input.Locale != nil {
		locale, ok := platformi18n.ParseLocale(strings.TrimSpace(*input.Locale))
		if !ok {
			return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_locale_value_is_invalid", "campaign locale value is invalid")
		}
		req.Locale = locale
	}

	_, err := g.Client.UpdateCampaign(ctx, req)
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_update_campaign",
			FallbackMessage: "failed to update campaign",
		})
	}
	return nil
}
