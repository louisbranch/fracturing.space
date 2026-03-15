package gametest

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FakeAuthClient is a test double for auth RPC dependencies.
type FakeAuthClient struct {
	User               *authv1.User
	GetUserErr         error
	LastGetUserRequest *authv1.GetUserRequest
}

// FakeSocialClient is a test double for social RPC dependencies.
type FakeSocialClient struct {
	Profile               *socialv1.UserProfile
	GetUserProfileErr     error
	LastGetUserProfileReq *socialv1.GetUserProfileRequest
	GetUserProfileCalls   int
}

func (f *FakeAuthClient) IssueJoinGrant(ctx context.Context, req *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginAccountRegistration(ctx context.Context, req *authv1.BeginAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishAccountRegistration(ctx context.Context, req *authv1.FinishAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) AcknowledgeAccountRegistration(ctx context.Context, req *authv1.AcknowledgeAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.AcknowledgeAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) CheckUsernameAvailability(ctx context.Context, req *authv1.CheckUsernameAvailabilityRequest, opts ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.LastGetUserRequest = req
	if f.GetUserErr != nil {
		return nil, f.GetUserErr
	}
	return &authv1.GetUserResponse{User: f.User}, nil
}

func (f *FakeAuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) LeaseIntegrationOutboxEvents(ctx context.Context, req *authv1.LeaseIntegrationOutboxEventsRequest, opts ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) AckIntegrationOutboxEvent(ctx context.Context, req *authv1.AckIntegrationOutboxEventRequest, opts ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginAccountRecovery(ctx context.Context, req *authv1.BeginAccountRecoveryRequest, opts ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginRecoveryPasskeyRegistration(ctx context.Context, req *authv1.BeginRecoveryPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishRecoveryPasskeyRegistration(ctx context.Context, req *authv1.FinishRecoveryPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) CreateWebSession(ctx context.Context, req *authv1.CreateWebSessionRequest, opts ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest, opts ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) RevokeWebSession(ctx context.Context, req *authv1.RevokeWebSessionRequest, opts ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) ListPasskeys(ctx context.Context, req *authv1.ListPasskeysRequest, opts ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) LookupUserByUsername(ctx context.Context, req *authv1.LookupUserByUsernameRequest, opts ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeSocialClient) AddContact(context.Context, *socialv1.AddContactRequest, ...grpc.CallOption) (*socialv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) RemoveContact(context.Context, *socialv1.RemoveContactRequest, ...grpc.CallOption) (*socialv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) ListContacts(context.Context, *socialv1.ListContactsRequest, ...grpc.CallOption) (*socialv1.ListContactsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SyncDirectoryUser(context.Context, *socialv1.SyncDirectoryUserRequest, ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) GetUserProfile(_ context.Context, req *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	f.GetUserProfileCalls++
	f.LastGetUserProfileReq = req
	if f.GetUserProfileErr != nil {
		return nil, f.GetUserProfileErr
	}
	if f.Profile == nil {
		return &socialv1.GetUserProfileResponse{}, nil
	}
	return &socialv1.GetUserProfileResponse{UserProfile: f.Profile}, nil
}
