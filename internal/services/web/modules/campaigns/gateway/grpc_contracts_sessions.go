package gateway

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
)

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
