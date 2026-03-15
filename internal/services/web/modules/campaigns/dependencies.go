package campaigns

import (
	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpc "google.golang.org/grpc"

	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// Dependencies contains campaign feature clients.
type Dependencies struct {
	CampaignClient           CampaignClient
	InteractionClient        campaigngateway.InteractionClient
	DiscoveryClient          campaigngateway.DiscoveryClient
	AgentClient              campaigngateway.AgentClient
	CampaignArtifactClient   campaigngateway.CampaignArtifactClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient campaigngateway.DaggerheartContentClient
	DaggerheartAssetClient   campaigngateway.DaggerheartAssetClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	SocialClient             campaigngateway.SocialClient
	AuthClient               campaigngateway.AuthClient
	AuthorizationClient      campaigngateway.AuthorizationClient
	ForkClient               campaigngateway.ForkClient
}

// BindAuthDependency wires auth-backed clients into the campaigns dependency set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AuthClient = authv1.NewAuthServiceClient(conn)
}

// BindSocialDependency wires social-backed clients into the campaigns dependency set.
func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.SocialClient = socialv1.NewSocialServiceClient(conn)
}

// BindGameDependency wires game-backed clients into the campaigns dependency set.
func BindGameDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.CampaignClient = statev1.NewCampaignServiceClient(conn)
	deps.InteractionClient = statev1.NewInteractionServiceClient(conn)
	deps.ForkClient = statev1.NewForkServiceClient(conn)
	deps.ParticipantClient = statev1.NewParticipantServiceClient(conn)
	deps.CharacterClient = statev1.NewCharacterServiceClient(conn)
	deps.DaggerheartContentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
	deps.DaggerheartAssetClient = daggerheartv1.NewDaggerheartAssetServiceClient(conn)
	deps.SessionClient = statev1.NewSessionServiceClient(conn)
	deps.InviteClient = statev1.NewInviteServiceClient(conn)
	deps.AuthorizationClient = statev1.NewAuthorizationServiceClient(conn)
}

// BindDiscoveryDependency wires discovery-backed clients into the campaigns dependency set.
func BindDiscoveryDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.DiscoveryClient = discoveryv1.NewDiscoveryServiceClient(conn)
}

// BindAIDependency wires AI-backed clients into the campaigns dependency set.
func BindAIDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AgentClient = aiv1.NewAgentServiceClient(conn)
	deps.CampaignArtifactClient = aiv1.NewCampaignArtifactServiceClient(conn)
}
