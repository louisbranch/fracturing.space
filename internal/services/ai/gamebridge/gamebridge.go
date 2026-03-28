package gamebridge

import (
	"context"
	"errors"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

var (
	// ErrCampaignAuthStateUnavailable reports that AI cannot read the game-owned
	// campaign auth state because the campaign-AI RPC dependency is unavailable.
	ErrCampaignAuthStateUnavailable = errors.New("campaign ai auth state client is unavailable")

	// ErrCampaignAuthorizationUnavailable reports that AI cannot ask game to
	// authorize campaign access because the authorization RPC dependency is unavailable.
	ErrCampaignAuthorizationUnavailable = errors.New("campaign authorization client is unavailable")

	// ErrMissingCallerIdentity reports that transport code did not provide a
	// user identity for a campaign authorization check.
	ErrMissingCallerIdentity = errors.New("missing caller identity")

	// ErrCampaignAccessDenied reports that game rejected the requested campaign access.
	ErrCampaignAccessDenied = errors.New("campaign access denied")
)

// Config declares the game-side RPC dependencies AI can use through the
// gateway. Any client may be nil when the game service is unavailable.
type Config struct {
	CampaignAI               gamev1.CampaignAIServiceClient
	Authorization            gamev1.AuthorizationServiceClient
	InternalServiceAllowlist map[string]struct{}
}

// Gateway is the AI-owned seam for game-side campaign auth-state, usage, and
// authorization lookups.
type Gateway struct {
	campaignAI               gamev1.CampaignAIServiceClient
	authorization            gamev1.AuthorizationServiceClient
	internalServiceAllowlist map[string]struct{}
}

// New constructs an AI-owned gateway over game-side collaborators.
func New(cfg Config) *Gateway {
	copiedAllowlist := make(map[string]struct{}, len(cfg.InternalServiceAllowlist))
	for serviceID := range cfg.InternalServiceAllowlist {
		normalized := strings.ToLower(strings.TrimSpace(serviceID))
		if normalized == "" {
			continue
		}
		copiedAllowlist[normalized] = struct{}{}
	}
	return &Gateway{
		campaignAI:               cfg.CampaignAI,
		authorization:            cfg.Authorization,
		internalServiceAllowlist: copiedAllowlist,
	}
}

// ActiveCampaignCount returns the number of active or draft campaigns bound to
// one agent. It degrades to zero when the campaign-AI RPC dependency is absent.
func (g *Gateway) ActiveCampaignCount(ctx context.Context, agentID string) (int32, error) {
	if g == nil || g.campaignAI == nil {
		return 0, nil
	}
	usage, err := g.campaignAI.GetCampaignAIBindingUsage(ctx, &gamev1.GetCampaignAIBindingUsageRequest{
		AiAgentId: agentID,
	})
	if err != nil {
		return 0, err
	}
	return usage.GetActiveCampaignCount(), nil
}

// CampaignAuthState returns the game-owned campaign AI runtime state for one
// campaign or ErrCampaignAuthStateUnavailable when the dependency is absent.
func (g *Gateway) CampaignAuthState(ctx context.Context, campaignID string) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	if g == nil || g.campaignAI == nil {
		return nil, ErrCampaignAuthStateUnavailable
	}
	return g.campaignAI.GetCampaignAIAuthState(ctx, &gamev1.GetCampaignAIAuthStateRequest{
		CampaignId: campaignID,
	})
}

// AuthorizeCampaign checks whether one caller can perform the requested action
// against a campaign. Internal service callers on the allowlist bypass the game RPC.
func (g *Gateway) AuthorizeCampaign(ctx context.Context, userID, campaignID string, action gamev1.AuthorizationAction) error {
	if g.IsAllowedInternalServiceCaller(ctx) {
		return nil
	}
	if strings.TrimSpace(userID) == "" {
		return ErrMissingCallerIdentity
	}
	if g == nil || g.authorization == nil {
		return ErrCampaignAuthorizationUnavailable
	}
	resp, err := g.authorization.Can(grpcauthctx.WithUserID(ctx, userID), &gamev1.CanRequest{
		CampaignId: campaignID,
		Action:     action,
		Resource:   gamev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
	})
	if err != nil {
		return err
	}
	if resp == nil || !resp.GetAllowed() {
		return ErrCampaignAccessDenied
	}
	return nil
}

// IsAllowedInternalServiceCaller reports whether the inbound caller is an
// explicitly allowed internal service for campaign-context operations.
func (g *Gateway) IsAllowedInternalServiceCaller(ctx context.Context) bool {
	if g == nil || len(g.internalServiceAllowlist) == 0 {
		return false
	}
	serviceID := strings.ToLower(strings.TrimSpace(grpcmeta.ServiceIDFromContext(ctx)))
	if serviceID == "" {
		return false
	}
	_, ok := g.internalServiceAllowlist[serviceID]
	return ok
}
