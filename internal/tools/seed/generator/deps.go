package generator

// deps.go defines narrow interfaces for external dependencies so the
// Generator can be tested with lightweight fakes instead of real gRPC clients.

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"google.golang.org/grpc"
)

// campaignCreator is the subset of CampaignServiceClient used by Generator.
type campaignCreator interface {
	CreateCampaign(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error)
	EndCampaign(ctx context.Context, in *statev1.EndCampaignRequest, opts ...grpc.CallOption) (*statev1.EndCampaignResponse, error)
	ArchiveCampaign(ctx context.Context, in *statev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error)
}

// participantCreator is the subset of ParticipantServiceClient used by Generator.
type participantCreator interface {
	CreateParticipant(ctx context.Context, in *statev1.CreateParticipantRequest, opts ...grpc.CallOption) (*statev1.CreateParticipantResponse, error)
}

// inviteManager is the subset of InviteServiceClient used by Generator.
type inviteManager interface {
	CreateInvite(ctx context.Context, in *invitev1.CreateInviteRequest, opts ...grpc.CallOption) (*invitev1.CreateInviteResponse, error)
	ClaimInvite(ctx context.Context, in *invitev1.ClaimInviteRequest, opts ...grpc.CallOption) (*invitev1.ClaimInviteResponse, error)
}

// characterCreator is the subset of CharacterServiceClient used by Generator.
type characterCreator interface {
	CreateCharacter(ctx context.Context, in *statev1.CreateCharacterRequest, opts ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	SetDefaultControl(ctx context.Context, in *statev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error)
}

// sessionManager is the subset of SessionServiceClient used by Generator.
type sessionManager interface {
	StartSession(ctx context.Context, in *statev1.StartSessionRequest, opts ...grpc.CallOption) (*statev1.StartSessionResponse, error)
	EndSession(ctx context.Context, in *statev1.EndSessionRequest, opts ...grpc.CallOption) (*statev1.EndSessionResponse, error)
	ListSessions(ctx context.Context, in *statev1.ListSessionsRequest, opts ...grpc.CallOption) (*statev1.ListSessionsResponse, error)
}

// eventAppender is the subset of EventServiceClient used by Generator.
type eventAppender interface {
	AppendEvent(ctx context.Context, in *statev1.AppendEventRequest, opts ...grpc.CallOption) (*statev1.AppendEventResponse, error)
}
