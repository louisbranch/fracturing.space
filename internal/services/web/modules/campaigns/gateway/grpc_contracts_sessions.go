package gateway

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
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
	ListInvites(context.Context, *invitev1.ListInvitesRequest, ...grpc.CallOption) (*invitev1.ListInvitesResponse, error)
}

// InviteMutationClient exposes invite mutations for campaign workspace pages.
type InviteMutationClient interface {
	CreateInvite(context.Context, *invitev1.CreateInviteRequest, ...grpc.CallOption) (*invitev1.CreateInviteResponse, error)
	RevokeInvite(context.Context, *invitev1.RevokeInviteRequest, ...grpc.CallOption) (*invitev1.RevokeInviteResponse, error)
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
