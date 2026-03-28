package app

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCGameGatewayMapsCampaignAndReadinessData(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 18, 0, 0, 0, time.UTC)
	campaigns := &campaignServiceServerStub{
		listCampaignsFunc: func(_ context.Context, req *gamev1.ListCampaignsRequest) (*gamev1.ListCampaignsResponse, error) {
			if len(req.GetStatuses()) == 0 {
				return &gamev1.ListCampaignsResponse{
					Campaigns: []*gamev1.Campaign{
						{
							Id:               "camp-preview",
							Name:             "Preview",
							Status:           gamev1.CampaignStatus_ACTIVE,
							ParticipantCount: 3,
							CharacterCount:   2,
							UpdatedAt:        timestamppb.New(now),
							LatestSessionAt:  timestamppb.New(now.Add(-48 * time.Hour)),
						},
					},
					NextPageToken: "next-page",
				}, nil
			}
			if req.GetPageToken() == "" {
				return &gamev1.ListCampaignsResponse{
					Campaigns: []*gamev1.Campaign{
						{
							Id:               "camp-1",
							Name:             "First",
							Status:           gamev1.CampaignStatus_DRAFT,
							ParticipantCount: 1,
							CharacterCount:   0,
							UpdatedAt:        timestamppb.New(now.Add(-2 * time.Hour)),
							LatestSessionAt:  timestamppb.New(now.Add(-8 * 24 * time.Hour)),
						},
					},
					NextPageToken: "page-2",
				}, nil
			}
			return &gamev1.ListCampaignsResponse{
				Campaigns: []*gamev1.Campaign{
					{
						Id:               "camp-2",
						Name:             "Second",
						Status:           gamev1.CampaignStatus_ACTIVE,
						ParticipantCount: 2,
						CharacterCount:   2,
						UpdatedAt:        timestamppb.New(now.Add(-1 * time.Hour)),
					},
				},
			}, nil
		},
		getCampaignSessionReadinessFunc: func(_ context.Context, req *gamev1.GetCampaignSessionReadinessRequest) (*gamev1.GetCampaignSessionReadinessResponse, error) {
			if req.GetCampaignId() != "camp-1" {
				t.Fatalf("GetCampaignSessionReadiness campaign_id = %q, want %q", req.GetCampaignId(), "camp-1")
			}
			return &gamev1.GetCampaignSessionReadinessResponse{
				Readiness: &gamev1.CampaignSessionReadiness{
					Blockers: []*gamev1.CampaignSessionReadinessBlocker{
						{
							Code:    "SESSION_READINESS_PLAYER_REQUIRED",
							Message: "Invite a player.",
							Action: &gamev1.CampaignSessionReadinessAction{
								ResponsibleUserIds:  []string{" user-1 ", "", "user-2", "user-1"},
								ResolutionKind:      gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_INVITE_PLAYER,
								TargetParticipantId: " p-1 ",
								TargetCharacterId:   " c-1 ",
							},
						},
					},
				},
			}, nil
		},
	}
	authorization := &authorizationServiceServerStub{
		batchCanFunc: func(_ context.Context, req *gamev1.BatchCanRequest) (*gamev1.BatchCanResponse, error) {
			if len(req.GetChecks()) != 2 {
				t.Fatalf("len(checks) = %d, want 2", len(req.GetChecks()))
			}
			if got := req.GetChecks()[0].GetResource(); got != gamev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION {
				t.Fatalf("resource = %v, want session", got)
			}
			return &gamev1.BatchCanResponse{
				Results: []*gamev1.BatchCanResult{
					{CheckId: "camp-1", Allowed: true},
					{Allowed: false},
					nil,
				},
			}, nil
		},
	}
	conn := newGatewayTestConn(t, gatewayTestServers{
		campaigns:     campaigns,
		authorization: authorization,
	})
	gateway := &grpcGameGateway{
		campaigns:     gamev1.NewCampaignServiceClient(conn),
		authorization: gamev1.NewAuthorizationServiceClient(conn),
	}

	page, err := gateway.ListCampaignPreviews(context.Background(), "user-1", 5)
	if err != nil {
		t.Fatalf("ListCampaignPreviews() error = %v", err)
	}
	if !page.HasMore || len(page.Campaigns) != 1 {
		t.Fatalf("campaign page = %#v, want one campaign and has_more=true", page)
	}
	if got := page.Campaigns[0].LatestSessionAt; got == nil || !got.Equal(now.Add(-48*time.Hour)) {
		t.Fatalf("LatestSessionAt = %v, want %v", got, now.Add(-48*time.Hour))
	}

	readinessCampaigns, err := gateway.ListReadinessCampaigns(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListReadinessCampaigns() error = %v", err)
	}
	if len(readinessCampaigns) != 2 {
		t.Fatalf("len(readinessCampaigns) = %d, want 2", len(readinessCampaigns))
	}
	if !readinessCampaigns[0].CanManageSession {
		t.Fatalf("CanManageSession for camp-1 = false, want true")
	}
	if readinessCampaigns[1].CanManageSession {
		t.Fatalf("CanManageSession for camp-2 = true, want false")
	}
	if readinessCampaigns[1].LatestSessionAt != nil {
		t.Fatalf("LatestSessionAt for camp-2 = %v, want nil", readinessCampaigns[1].LatestSessionAt)
	}
	if got := campaigns.userIDs; len(got) != 3 || got[0] != "user-1" || got[1] != "user-1" || got[2] != "user-1" {
		t.Fatalf("campaign service user ids = %#v, want repeated user-1 metadata", got)
	}

	readiness, err := gateway.GetCampaignReadiness(context.Background(), "user-1", "camp-1")
	if err != nil {
		t.Fatalf("GetCampaignReadiness() error = %v", err)
	}
	if len(readiness.Blockers) != 1 {
		t.Fatalf("len(blockers) = %d, want 1", len(readiness.Blockers))
	}
	blocker := readiness.Blockers[0]
	if blocker.ActionKind != domain.CampaignStartNudgeActionInvitePlayer {
		t.Fatalf("ActionKind = %v, want invite player", blocker.ActionKind)
	}
	if got := blocker.ResponsibleUserIDs; len(got) != 2 || got[0] != "user-1" || got[1] != "user-2" {
		t.Fatalf("ResponsibleUserIDs = %#v, want deduped trimmed ids", got)
	}
	if blocker.TargetParticipantID != "p-1" || blocker.TargetCharacterID != "c-1" {
		t.Fatalf("target ids = (%q,%q), want (%q,%q)", blocker.TargetParticipantID, blocker.TargetCharacterID, "p-1", "c-1")
	}
}

func TestAncillaryGRPCGatewaysMapResponsesAndErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 19, 0, 0, 0, time.UTC)
	auth := &authServiceServerStub{
		getUserFunc: func(_ context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
			if req.GetUserId() != "user-42" {
				t.Fatalf("GetUser user_id = %q, want %q", req.GetUserId(), "user-42")
			}
			return &authv1.GetUserResponse{
				User: &authv1.User{
					Id:       "user-42",
					Username: "rook",
				},
			}, nil
		},
	}
	invites := &inviteServiceServerStub{
		listPendingInvitesForUserFunc: func(_ context.Context, req *invitev1.ListPendingInvitesForUserRequest) (*invitev1.ListPendingInvitesForUserResponse, error) {
			if req.GetPageSize() != 4 {
				t.Fatalf("invite page_size = %d, want 4", req.GetPageSize())
			}
			return &invitev1.ListPendingInvitesForUserResponse{
				Invites: []*invitev1.PendingInviteForUserEntry{
					{
						Invite: &invitev1.Invite{
							Id:            "invite-1",
							CampaignId:    "camp-1",
							ParticipantId: "p-1",
							CreatedAt:     timestamppb.New(now.Add(-1 * time.Hour)),
						},
						Campaign:    &invitev1.InviteCampaignSummary{Name: "Skyline"},
						Participant: &invitev1.InviteParticipantSummary{Name: "Rook"},
					},
				},
				NextPageToken: "more",
			}, nil
		},
	}
	sessions := &sessionServiceServerStub{
		listActiveSessionsForUserFunc: func(_ context.Context, req *gamev1.ListActiveSessionsForUserRequest) (*gamev1.ListActiveSessionsForUserResponse, error) {
			if req.GetPageSize() != 3 {
				t.Fatalf("session page_size = %d, want 3", req.GetPageSize())
			}
			return &gamev1.ListActiveSessionsForUserResponse{
				Sessions: []*gamev1.ActiveUserSession{
					{
						CampaignId:   "camp-1",
						CampaignName: "Skyline",
						SessionId:    "sess-1",
						SessionName:  "First Light",
						StartedAt:    timestamppb.New(now),
					},
				},
				HasMore: true,
			}, nil
		},
	}
	social := &socialServiceServerStub{
		getUserProfileFunc: func(_ context.Context, req *socialv1.GetUserProfileRequest) (*socialv1.GetUserProfileResponse, error) {
			if req.GetUserId() == "missing" {
				return nil, status.Error(codes.NotFound, "missing")
			}
			return &socialv1.GetUserProfileResponse{
				UserProfile: &socialv1.UserProfile{Name: "Rook"},
			}, nil
		},
	}
	notifications := &notificationServiceServerStub{
		getUnreadNotificationStatusFunc: func(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
			return &notificationsv1.GetUnreadNotificationStatusResponse{
				HasUnread:   true,
				UnreadCount: 7,
			}, nil
		},
	}
	conn := newGatewayTestConn(t, gatewayTestServers{
		auth:          auth,
		invites:       invites,
		sessions:      sessions,
		social:        social,
		notifications: notifications,
	})

	authGateway := newGRPCAuthGateway(authv1.NewAuthServiceClient(conn))
	identity, err := authGateway.GetUserIdentity(context.Background(), "user-42")
	if err != nil {
		t.Fatalf("GetUserIdentity() error = %v", err)
	}
	if identity.Username != "rook" {
		t.Fatalf("Username = %q, want %q", identity.Username, "rook")
	}

	gameGateway := &grpcGameGateway{
		invites:  invitev1.NewInviteServiceClient(conn),
		sessions: gamev1.NewSessionServiceClient(conn),
	}
	invitePage, err := gameGateway.ListPendingInvitePreviews(context.Background(), "user-42", 4)
	if err != nil {
		t.Fatalf("ListPendingInvitePreviews() error = %v", err)
	}
	if !invitePage.HasMore || len(invitePage.Invites) != 1 {
		t.Fatalf("invite page = %#v, want one invite and has_more=true", invitePage)
	}
	if invitePage.Invites[0].CampaignName != "Skyline" || invitePage.Invites[0].ParticipantName != "Rook" {
		t.Fatalf("invite preview = %#v", invitePage.Invites[0])
	}

	activePage, err := gameGateway.ListActiveSessionPreviews(context.Background(), "user-42", 3)
	if err != nil {
		t.Fatalf("ListActiveSessionPreviews() error = %v", err)
	}
	if !activePage.HasMore || len(activePage.Sessions) != 1 {
		t.Fatalf("active session page = %#v, want one session and has_more=true", activePage)
	}
	if activePage.Sessions[0].SessionName != "First Light" {
		t.Fatalf("SessionName = %q, want %q", activePage.Sessions[0].SessionName, "First Light")
	}

	socialGateway := newGRPCSocialGateway(socialv1.NewSocialServiceClient(conn))
	profile, err := socialGateway.GetUserProfile(context.Background(), "user-42")
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if profile.Name != "Rook" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Rook")
	}
	if _, err := socialGateway.GetUserProfile(context.Background(), "missing"); !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("missing profile error = %v, want ErrProfileNotFound", err)
	}

	notificationsGateway := newGRPCNotificationsGateway(notificationsv1.NewNotificationServiceClient(conn))
	unread, err := notificationsGateway.GetUnreadStatus(context.Background(), "user-42")
	if err != nil {
		t.Fatalf("GetUnreadStatus() error = %v", err)
	}
	if !unread.HasUnread || unread.UnreadCount != 7 {
		t.Fatalf("unread = %#v, want HasUnread=true UnreadCount=7", unread)
	}

	if got := auth.userIDs; len(got) != 1 || got[0] != "user-42" {
		t.Fatalf("auth service user ids = %#v, want one user-42 request", got)
	}
}

func TestGRPCGatewaysRejectMissingClientsAndMapHelpers(t *testing.T) {
	t.Parallel()

	if _, err := (*grpcAuthGateway)(nil).GetUserIdentity(context.Background(), "user-1"); err == nil {
		t.Fatal("grpcAuthGateway.GetUserIdentity() error = nil, want error")
	}
	if _, err := (&grpcGameGateway{}).ListCampaignPreviews(context.Background(), "user-1", 1); err == nil {
		t.Fatal("grpcGameGateway.ListCampaignPreviews() error = nil, want error")
	}
	if _, err := (&grpcGameGateway{}).ListPendingInvitePreviews(context.Background(), "user-1", 1); err == nil {
		t.Fatal("grpcGameGateway.ListPendingInvitePreviews() error = nil, want error")
	}
	if _, err := (&grpcGameGateway{}).ListActiveSessionPreviews(context.Background(), "user-1", 1); err == nil {
		t.Fatal("grpcGameGateway.ListActiveSessionPreviews() error = nil, want error")
	}
	if _, err := (&grpcSocialGateway{}).GetUserProfile(context.Background(), "user-1"); err == nil {
		t.Fatal("grpcSocialGateway.GetUserProfile() error = nil, want error")
	}
	if _, err := (&grpcNotificationsGateway{}).GetUnreadStatus(context.Background(), "user-1"); err == nil {
		t.Fatal("grpcNotificationsGateway.GetUnreadStatus() error = nil, want error")
	}

	if got := normalizedIDs([]string{" user-1 ", "", "user-1", "user-2"}); len(got) != 2 || got[0] != "user-1" || got[1] != "user-2" {
		t.Fatalf("normalizedIDs() = %#v, want deduped trimmed ids", got)
	}
	if got := normalizedIDs([]string{"", "   "}); got != nil {
		t.Fatalf("normalizedIDs(empty) = %#v, want nil", got)
	}
	if got := protoTimePtr(nil); got != nil {
		t.Fatalf("protoTimePtr(nil) = %v, want nil", got)
	}
	point := timestamppb.New(time.Date(2026, time.March, 22, 0, 0, 0, 0, time.UTC))
	if got := protoTimePtr(point); got == nil || !got.Equal(point.AsTime()) {
		t.Fatalf("protoTimePtr() = %v, want %v", got, point.AsTime())
	}
	if got := campaignStatusFromProto(gamev1.CampaignStatus_ARCHIVED); got != domain.CampaignStatusArchived {
		t.Fatalf("campaignStatusFromProto(archived) = %v", got)
	}
	if got := campaignStatusFromProto(gamev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED); got != domain.CampaignStatusUnspecified {
		t.Fatalf("campaignStatusFromProto(unspecified) = %v", got)
	}
	if got := readinessActionKindFromProto(gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_MANAGE_PARTICIPANTS); got != domain.CampaignStartNudgeActionManageParticipants {
		t.Fatalf("readinessActionKindFromProto(manage_participants) = %v", got)
	}
	if got := readinessActionKindFromProto(gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_UNSPECIFIED); got != domain.CampaignStartNudgeActionUnspecified {
		t.Fatalf("readinessActionKindFromProto(unspecified) = %v", got)
	}
}

type gatewayTestServers struct {
	auth          *authServiceServerStub
	campaigns     *campaignServiceServerStub
	authorization *authorizationServiceServerStub
	invites       *inviteServiceServerStub
	sessions      *sessionServiceServerStub
	social        *socialServiceServerStub
	notifications *notificationServiceServerStub
}

func newGatewayTestConn(t *testing.T, servers gatewayTestServers) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	if servers.auth != nil {
		authv1.RegisterAuthServiceServer(server, servers.auth)
	}
	if servers.campaigns != nil {
		gamev1.RegisterCampaignServiceServer(server, servers.campaigns)
	}
	if servers.authorization != nil {
		gamev1.RegisterAuthorizationServiceServer(server, servers.authorization)
	}
	if servers.invites != nil {
		invitev1.RegisterInviteServiceServer(server, servers.invites)
	}
	if servers.sessions != nil {
		gamev1.RegisterSessionServiceServer(server, servers.sessions)
	}
	if servers.social != nil {
		socialv1.RegisterSocialServiceServer(server, servers.social)
	}
	if servers.notifications != nil {
		notificationsv1.RegisterNotificationServiceServer(server, servers.notifications)
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

type authServiceServerStub struct {
	authv1.UnimplementedAuthServiceServer
	userIDs     []string
	getUserFunc func(context.Context, *authv1.GetUserRequest) (*authv1.GetUserResponse, error)
}

func (s *authServiceServerStub) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	s.userIDs = append(s.userIDs, incomingUserID(ctx))
	if s.getUserFunc == nil {
		return &authv1.GetUserResponse{}, nil
	}
	return s.getUserFunc(ctx, req)
}

type campaignServiceServerStub struct {
	gamev1.UnimplementedCampaignServiceServer
	userIDs                         []string
	listCampaignsFunc               func(context.Context, *gamev1.ListCampaignsRequest) (*gamev1.ListCampaignsResponse, error)
	getCampaignSessionReadinessFunc func(context.Context, *gamev1.GetCampaignSessionReadinessRequest) (*gamev1.GetCampaignSessionReadinessResponse, error)
}

func (s *campaignServiceServerStub) ListCampaigns(ctx context.Context, req *gamev1.ListCampaignsRequest) (*gamev1.ListCampaignsResponse, error) {
	s.userIDs = append(s.userIDs, incomingUserID(ctx))
	if s.listCampaignsFunc == nil {
		return &gamev1.ListCampaignsResponse{}, nil
	}
	return s.listCampaignsFunc(ctx, req)
}

func (s *campaignServiceServerStub) GetCampaignSessionReadiness(ctx context.Context, req *gamev1.GetCampaignSessionReadinessRequest) (*gamev1.GetCampaignSessionReadinessResponse, error) {
	s.userIDs = append(s.userIDs, incomingUserID(ctx))
	if s.getCampaignSessionReadinessFunc == nil {
		return &gamev1.GetCampaignSessionReadinessResponse{}, nil
	}
	return s.getCampaignSessionReadinessFunc(ctx, req)
}

type authorizationServiceServerStub struct {
	gamev1.UnimplementedAuthorizationServiceServer
	batchCanFunc func(context.Context, *gamev1.BatchCanRequest) (*gamev1.BatchCanResponse, error)
}

func (s *authorizationServiceServerStub) BatchCan(ctx context.Context, req *gamev1.BatchCanRequest) (*gamev1.BatchCanResponse, error) {
	if s.batchCanFunc == nil {
		return &gamev1.BatchCanResponse{}, nil
	}
	return s.batchCanFunc(ctx, req)
}

func (*authorizationServiceServerStub) Can(context.Context, *gamev1.CanRequest) (*gamev1.CanResponse, error) {
	return &gamev1.CanResponse{}, nil
}

type inviteServiceServerStub struct {
	invitev1.UnimplementedInviteServiceServer
	listPendingInvitesForUserFunc func(context.Context, *invitev1.ListPendingInvitesForUserRequest) (*invitev1.ListPendingInvitesForUserResponse, error)
}

func (s *inviteServiceServerStub) ListPendingInvitesForUser(ctx context.Context, req *invitev1.ListPendingInvitesForUserRequest) (*invitev1.ListPendingInvitesForUserResponse, error) {
	if s.listPendingInvitesForUserFunc == nil {
		return &invitev1.ListPendingInvitesForUserResponse{}, nil
	}
	return s.listPendingInvitesForUserFunc(ctx, req)
}

type sessionServiceServerStub struct {
	gamev1.UnimplementedSessionServiceServer
	listActiveSessionsForUserFunc func(context.Context, *gamev1.ListActiveSessionsForUserRequest) (*gamev1.ListActiveSessionsForUserResponse, error)
}

func (s *sessionServiceServerStub) ListActiveSessionsForUser(ctx context.Context, req *gamev1.ListActiveSessionsForUserRequest) (*gamev1.ListActiveSessionsForUserResponse, error) {
	if s.listActiveSessionsForUserFunc == nil {
		return &gamev1.ListActiveSessionsForUserResponse{}, nil
	}
	return s.listActiveSessionsForUserFunc(ctx, req)
}

type socialServiceServerStub struct {
	socialv1.UnimplementedSocialServiceServer
	getUserProfileFunc func(context.Context, *socialv1.GetUserProfileRequest) (*socialv1.GetUserProfileResponse, error)
}

func (s *socialServiceServerStub) GetUserProfile(ctx context.Context, req *socialv1.GetUserProfileRequest) (*socialv1.GetUserProfileResponse, error) {
	if s.getUserProfileFunc == nil {
		return &socialv1.GetUserProfileResponse{}, nil
	}
	return s.getUserProfileFunc(ctx, req)
}

type notificationServiceServerStub struct {
	notificationsv1.UnimplementedNotificationServiceServer
	getUnreadNotificationStatusFunc func(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest) (*notificationsv1.GetUnreadNotificationStatusResponse, error)
}

func (s *notificationServiceServerStub) GetUnreadNotificationStatus(ctx context.Context, req *notificationsv1.GetUnreadNotificationStatusRequest) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	if s.getUnreadNotificationStatusFunc == nil {
		return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
	}
	return s.getUnreadNotificationStatusFunc(ctx, req)
}

func incomingUserID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
