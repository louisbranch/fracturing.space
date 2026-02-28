// Package module defines the feature contract used by web composition.
package module

import (
	"context"
	"net/http"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"google.golang.org/grpc"
)

// Viewer contains user-facing chrome data for authenticated app pages.
type Viewer struct {
	DisplayName            string
	AvatarURL              string
	ProfileURL             string
	HasUnreadNotifications bool
}

// ResolveViewer resolves app chrome viewer state for a request.
type ResolveViewer func(*http.Request) Viewer

// ResolveUserID resolves the authenticated user id for a request.
type ResolveUserID func(*http.Request) string

// ResolveLanguage returns the effective request language.
type ResolveLanguage func(*http.Request) string

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

// CharacterClient exposes character listing for campaign workspace pages.
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

// AuthClient performs passkey and user bootstrap operations.
type AuthClient interface {
	CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error)
	BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error)
	BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error)
	FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error)
	CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error)
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error)
	RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error)
}

// AccountClient exposes account profile read/update operations.
type AccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
	UpdateProfile(context.Context, *authv1.UpdateProfileRequest, ...grpc.CallOption) (*authv1.UpdateProfileResponse, error)
}

// SocialClient exposes profile lookup and mutation operations.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error)
	SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error)
}

// CredentialClient exposes AI credential listing and mutation operations.
type CredentialClient interface {
	ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error)
	CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error)
	RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error)
}

// UserHubClient exposes user-dashboard aggregation operations.
type UserHubClient interface {
	GetDashboard(context.Context, *userhubv1.GetDashboardRequest, ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error)
}

// NotificationClient exposes notification inbox listing and acknowledgement operations.
type NotificationClient interface {
	ListNotifications(context.Context, *notificationsv1.ListNotificationsRequest, ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error)
	GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error)
	MarkNotificationRead(context.Context, *notificationsv1.MarkNotificationReadRequest, ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error)
}

// Dependencies carries shared runtime contracts to modules.
type Dependencies struct {
	ResolveViewer            ResolveViewer
	ResolveUserID            ResolveUserID
	ResolveLanguage          ResolveLanguage
	AssetBaseURL             string
	ChatFallbackPort         string
	CampaignClient           CampaignClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient DaggerheartContentClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	AuthorizationClient      AuthorizationClient
	AuthClient               AuthClient
	AccountClient            AccountClient
	SocialClient             SocialClient
	CredentialClient         CredentialClient
	UserHubClient            UserHubClient
	NotificationClient       NotificationClient
}

// Mount describes a module route mount.
type Mount struct {
	Prefix  string
	Handler http.Handler
}

// Module declares the minimum contract required by web composition.
type Module interface {
	ID() string
	Mount(Dependencies) (Mount, error)
}
