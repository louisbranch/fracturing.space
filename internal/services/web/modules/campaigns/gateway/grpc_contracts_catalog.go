package gateway

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

// CampaignReadClient exposes campaign query operations from the game service.
type CampaignReadClient interface {
	ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error)
	GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error)
	GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error)
}

// CampaignMutationClient exposes campaign mutation operations from the game service.
type CampaignMutationClient interface {
	CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error)
	UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error)
	SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error)
	ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error)
}

// DiscoveryClient exposes discovery reads required by the protected starter flow.
type DiscoveryClient interface {
	GetDiscoveryEntry(context.Context, *discoveryv1.GetDiscoveryEntryRequest, ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error)
}

// ForkClient exposes campaign forking required by the protected starter flow.
type ForkClient interface {
	ForkCampaign(context.Context, *statev1.ForkCampaignRequest, ...grpc.CallOption) (*statev1.ForkCampaignResponse, error)
}

// AgentClient exposes AI agent listing used for owner-only campaign binding UX.
type AgentClient interface {
	ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error)
}

// CampaignArtifactClient exposes AI-owned campaign artifact seeding for starter launch.
type CampaignArtifactClient interface {
	EnsureCampaignArtifacts(context.Context, *aiv1.EnsureCampaignArtifactsRequest, ...grpc.CallOption) (*aiv1.EnsureCampaignArtifactsResponse, error)
}
