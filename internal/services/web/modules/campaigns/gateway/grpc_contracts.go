package gateway

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
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

// CommunicationClient exposes game-owned communication context for the web game surface.
type CommunicationClient interface {
	GetCommunicationContext(context.Context, *statev1.GetCommunicationContextRequest, ...grpc.CallOption) (*statev1.GetCommunicationContextResponse, error)
}

// AgentClient exposes AI agent listing used for owner-only campaign binding UX.
type AgentClient interface {
	ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error)
}

// ParticipantReadClient exposes participant queries for campaign workspace pages.
type ParticipantReadClient interface {
	ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error)
	GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error)
}

// ParticipantMutationClient exposes participant mutations for campaign workspace pages.
type ParticipantMutationClient interface {
	CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error)
	UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error)
}

// CharacterReadClient exposes character query operations for campaign workspace pages.
type CharacterReadClient interface {
	ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error)
	ListCharacterProfiles(context.Context, *statev1.ListCharacterProfilesRequest, ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
}

// CharacterMutationClient exposes character mutations for campaign workspace pages.
type CharacterMutationClient interface {
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error)
	DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error)
	SetDefaultControl(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error)
	ClaimCharacterControl(context.Context, *statev1.ClaimCharacterControlRequest, ...grpc.CallOption) (*statev1.ClaimCharacterControlResponse, error)
	ReleaseCharacterControl(context.Context, *statev1.ReleaseCharacterControlRequest, ...grpc.CallOption) (*statev1.ReleaseCharacterControlResponse, error)
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

// SessionReadClient exposes session queries for campaign workspace pages.
type SessionReadClient interface {
	ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error)
}

// SessionMutationClient exposes session mutations for campaign workspace pages.
type SessionMutationClient interface {
	StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error)
	EndSession(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error)
}

// InviteReadClient exposes invite queries for campaign workspace pages.
type InviteReadClient interface {
	ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error)
	GetPublicInvite(context.Context, *statev1.GetPublicInviteRequest, ...grpc.CallOption) (*statev1.GetPublicInviteResponse, error)
}

// InviteMutationClient exposes invite mutations for campaign workspace pages.
type InviteMutationClient interface {
	CreateInvite(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error)
	ClaimInvite(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error)
	DeclineInvite(context.Context, *statev1.DeclineInviteRequest, ...grpc.CallOption) (*statev1.DeclineInviteResponse, error)
	RevokeInvite(context.Context, *statev1.RevokeInviteRequest, ...grpc.CallOption) (*statev1.RevokeInviteResponse, error)
}

// AuthClient resolves auth-owned users from usernames for invite targeting.
type AuthClient interface {
	LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error)
	GetUser(context.Context, *authv1.GetUserRequest, ...grpc.CallOption) (*authv1.GetUserResponse, error)
	IssueJoinGrant(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error)
}

// SocialClient exposes invite-search operations backed by social data.
type SocialClient interface {
	SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error)
}

// AuthorizationClient exposes campaign authorization checks.
type AuthorizationClient interface {
	Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error)
	BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error)
}
