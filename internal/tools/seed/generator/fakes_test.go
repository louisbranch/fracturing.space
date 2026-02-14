package generator

import (
	"context"
	"fmt"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

// fakeCampaignCreator implements campaignCreator with injectable functions.
type fakeCampaignCreator struct {
	createCampaign  func(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error)
	endCampaign     func(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error)
	archiveCampaign func(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error)
}

func (f *fakeCampaignCreator) CreateCampaign(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	if f.createCampaign != nil {
		return f.createCampaign(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateCampaign: not implemented")
}

func (f *fakeCampaignCreator) EndCampaign(ctx context.Context, in *statev1.EndCampaignRequest, opts ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	if f.endCampaign != nil {
		return f.endCampaign(ctx, in, opts...)
	}
	return nil, fmt.Errorf("EndCampaign: not implemented")
}

func (f *fakeCampaignCreator) ArchiveCampaign(ctx context.Context, in *statev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	if f.archiveCampaign != nil {
		return f.archiveCampaign(ctx, in, opts...)
	}
	return nil, fmt.Errorf("ArchiveCampaign: not implemented")
}

// fakeParticipantCreator implements participantCreator with an injectable function.
type fakeParticipantCreator struct {
	create func(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error)
}

func (f *fakeParticipantCreator) CreateParticipant(ctx context.Context, in *statev1.CreateParticipantRequest, opts ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	if f.create != nil {
		return f.create(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateParticipant: not implemented")
}

// fakeInviteManager implements inviteManager with injectable functions.
type fakeInviteManager struct {
	createInvite func(context.Context, *statev1.CreateInviteRequest, ...grpc.CallOption) (*statev1.CreateInviteResponse, error)
	claimInvite  func(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error)
}

func (f *fakeInviteManager) CreateInvite(ctx context.Context, in *statev1.CreateInviteRequest, opts ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	if f.createInvite != nil {
		return f.createInvite(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateInvite: not implemented")
}

func (f *fakeInviteManager) ClaimInvite(ctx context.Context, in *statev1.ClaimInviteRequest, opts ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	if f.claimInvite != nil {
		return f.claimInvite(ctx, in, opts...)
	}
	return nil, fmt.Errorf("ClaimInvite: not implemented")
}

// fakeCharacterCreator implements characterCreator with injectable functions.
type fakeCharacterCreator struct {
	create            func(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	setDefaultControl func(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error)
}

func (f *fakeCharacterCreator) CreateCharacter(ctx context.Context, in *statev1.CreateCharacterRequest, opts ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	if f.create != nil {
		return f.create(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateCharacter: not implemented")
}

func (f *fakeCharacterCreator) SetDefaultControl(ctx context.Context, in *statev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	if f.setDefaultControl != nil {
		return f.setDefaultControl(ctx, in, opts...)
	}
	return nil, fmt.Errorf("SetDefaultControl: not implemented")
}

// fakeSessionManager implements sessionManager with injectable functions.
type fakeSessionManager struct {
	startSession func(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error)
	endSession   func(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error)
	listSessions func(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error)
}

func (f *fakeSessionManager) StartSession(ctx context.Context, in *statev1.StartSessionRequest, opts ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	if f.startSession != nil {
		return f.startSession(ctx, in, opts...)
	}
	return nil, fmt.Errorf("StartSession: not implemented")
}

func (f *fakeSessionManager) EndSession(ctx context.Context, in *statev1.EndSessionRequest, opts ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	if f.endSession != nil {
		return f.endSession(ctx, in, opts...)
	}
	return nil, fmt.Errorf("EndSession: not implemented")
}

func (f *fakeSessionManager) ListSessions(ctx context.Context, in *statev1.ListSessionsRequest, opts ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if f.listSessions != nil {
		return f.listSessions(ctx, in, opts...)
	}
	return nil, fmt.Errorf("ListSessions: not implemented")
}

// fakeEventAppender implements eventAppender with an injectable function.
type fakeEventAppender struct {
	appendEvent func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error)
}

func (f *fakeEventAppender) AppendEvent(ctx context.Context, in *statev1.AppendEventRequest, opts ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
	if f.appendEvent != nil {
		return f.appendEvent(ctx, in, opts...)
	}
	return nil, fmt.Errorf("AppendEvent: not implemented")
}

// fakeAuthProvider implements authProvider with injectable functions.
type fakeAuthProvider struct {
	createUser     func(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error)
	issueJoinGrant func(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error)
}

func (f *fakeAuthProvider) CreateUser(ctx context.Context, in *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	if f.createUser != nil {
		return f.createUser(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateUser: not implemented")
}

func (f *fakeAuthProvider) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	if f.issueJoinGrant != nil {
		return f.issueJoinGrant(ctx, in, opts...)
	}
	return nil, fmt.Errorf("IssueJoinGrant: not implemented")
}

// testDeps returns a generatorDeps wired to the given fakes.
func testDeps(
	camp *fakeCampaignCreator,
	part *fakeParticipantCreator,
	inv *fakeInviteManager,
	char *fakeCharacterCreator,
	sess *fakeSessionManager,
	evt *fakeEventAppender,
	auth *fakeAuthProvider,
) generatorDeps {
	return generatorDeps{
		campaigns:    camp,
		participants: part,
		invites:      inv,
		characters:   char,
		sessions:     sess,
		events:       evt,
		authClient:   auth,
	}
}
