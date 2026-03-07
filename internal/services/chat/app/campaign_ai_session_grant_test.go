package server

import (
	"context"
	"errors"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeCampaignAIServiceClient struct {
	issueReq        *statev1.IssueCampaignAISessionGrantRequest
	issueCtx        context.Context
	issueResp       *statev1.IssueCampaignAISessionGrantResponse
	issueErr        error
	authStateReq    *statev1.GetCampaignAIAuthStateRequest
	authStateResp   *statev1.GetCampaignAIAuthStateResponse
	authStateErr    error
	bindingUsageErr error
}

func (f *fakeCampaignAIServiceClient) IssueCampaignAISessionGrant(ctx context.Context, req *statev1.IssueCampaignAISessionGrantRequest, _ ...grpc.CallOption) (*statev1.IssueCampaignAISessionGrantResponse, error) {
	f.issueCtx = ctx
	f.issueReq = req
	if f.issueErr != nil {
		return nil, f.issueErr
	}
	if f.issueResp == nil {
		return &statev1.IssueCampaignAISessionGrantResponse{}, nil
	}
	return f.issueResp, nil
}

func (f *fakeCampaignAIServiceClient) GetCampaignAIBindingUsage(context.Context, *statev1.GetCampaignAIBindingUsageRequest, ...grpc.CallOption) (*statev1.GetCampaignAIBindingUsageResponse, error) {
	if f.bindingUsageErr != nil {
		return nil, f.bindingUsageErr
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeCampaignAIServiceClient) GetCampaignAIAuthState(_ context.Context, req *statev1.GetCampaignAIAuthStateRequest, _ ...grpc.CallOption) (*statev1.GetCampaignAIAuthStateResponse, error) {
	f.authStateReq = req
	if f.authStateErr != nil {
		return nil, f.authStateErr
	}
	if f.authStateResp == nil {
		return &statev1.GetCampaignAIAuthStateResponse{}, nil
	}
	return f.authStateResp, nil
}

func TestIssueAISessionGrantForRoomSkipsWhenAIRelayDisabled(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setAISessionGrant("old-grant", 1, time.Now().UTC().Add(time.Minute))
	client := &fakeCampaignAIServiceClient{}

	if err := issueAISessionGrantForRoom(context.Background(), client, room, "user-1"); err != nil {
		t.Fatalf("issue ai session grant for room: %v", err)
	}
	if got := room.aiSessionGrantValue(); got != "" {
		t.Fatalf("grant = %q, want empty", got)
	}
	if client.issueReq != nil {
		t.Fatal("expected no issue request when ai relay is disabled")
	}
}

func TestIssueAISessionGrantForRoomRequiresSessionID(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("old-grant", 1, time.Now().UTC().Add(time.Minute))

	err := issueAISessionGrantForRoom(context.Background(), &fakeCampaignAIServiceClient{}, room, "user-1")
	if err == nil || err.Error() != "session id is required" {
		t.Fatalf("error = %v, want session id required", err)
	}
	if got := room.aiSessionGrantValue(); got != "" {
		t.Fatalf("grant = %q, want empty", got)
	}
}

func TestIssueAISessionGrantForRoomRequiresContext(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("old-grant", 1, time.Now().UTC().Add(time.Minute))

	err := issueAISessionGrantForRoom(nil, &fakeCampaignAIServiceClient{}, room, "user-1")
	if err == nil || err.Error() != "context is required" {
		t.Fatalf("error = %v, want context is required", err)
	}
	if got := room.aiSessionGrantValue(); got != "old-grant" {
		t.Fatalf("grant = %q, want %q", got, "old-grant")
	}
}

func TestIssueAISessionGrantForRoomClearsGrantOnIssueError(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("old-grant", 1, time.Now().UTC().Add(time.Minute))

	err := issueAISessionGrantForRoom(context.Background(), &fakeCampaignAIServiceClient{issueErr: errors.New("boom")}, room, "user-1")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v, want boom", err)
	}
	if got := room.aiSessionGrantValue(); got != "" {
		t.Fatalf("grant = %q, want empty", got)
	}
}

func TestIssueAISessionGrantForRoomRequiresIssuedToken(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")

	client := &fakeCampaignAIServiceClient{
		issueResp: &statev1.IssueCampaignAISessionGrantResponse{Grant: &statev1.AISessionGrant{}},
	}
	err := issueAISessionGrantForRoom(context.Background(), client, room, "user-1")
	if err == nil || err.Error() != "issued ai session grant token is empty" {
		t.Fatalf("error = %v, want empty token error", err)
	}
}

func TestIssueAISessionGrantForRoomSetsGrantAndMetadata(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	room := newCampaignRoom("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")

	client := &fakeCampaignAIServiceClient{
		issueResp: &statev1.IssueCampaignAISessionGrantResponse{
			Grant: &statev1.AISessionGrant{
				Token:     "grant-token",
				AuthEpoch: 9,
				ExpiresAt: timestamppb.New(now.Add(2 * time.Minute)),
			},
		},
	}
	if err := issueAISessionGrantForRoom(context.Background(), client, room, "user-42"); err != nil {
		t.Fatalf("issue ai session grant for room: %v", err)
	}

	if client.issueReq == nil {
		t.Fatal("expected issue request")
	}
	if client.issueReq.GetCampaignId() != "camp-1" || client.issueReq.GetSessionId() != "session-1" || client.issueReq.GetAiAgentId() != "agent-1" {
		t.Fatalf("unexpected issue request payload: %+v", client.issueReq)
	}
	if got := room.aiSessionGrantValue(); got != "grant-token" {
		t.Fatalf("grant = %q, want %q", got, "grant-token")
	}
	if room.aiAuthEpoch != 9 {
		t.Fatalf("auth epoch = %d, want %d", room.aiAuthEpoch, 9)
	}
	if !room.aiGrantExpiresAt.Equal(now.Add(2 * time.Minute).UTC()) {
		t.Fatalf("grant expires at = %s, want %s", room.aiGrantExpiresAt, now.Add(2*time.Minute).UTC())
	}

	md, ok := metadata.FromOutgoingContext(client.issueCtx)
	if !ok {
		t.Fatal("expected outgoing metadata in issue context")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-42" {
		t.Fatalf("user metadata = %v, want [user-42]", got)
	}
}

func TestSyncRoomAIContextFromGame(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("session-old")
	room.setAIBinding("hybrid", "agent-old")

	client := &fakeCampaignAIServiceClient{
		authStateResp: &statev1.GetCampaignAIAuthStateResponse{
			CampaignId:      "camp-1",
			AiAgentId:       " agent-new ",
			ActiveSessionId: " session-new ",
		},
	}
	if err := syncRoomAIContextFromGame(context.Background(), client, room); err != nil {
		t.Fatalf("sync room ai context: %v", err)
	}
	if client.authStateReq == nil || client.authStateReq.GetCampaignId() != "camp-1" {
		t.Fatalf("unexpected auth state request: %+v", client.authStateReq)
	}
	if got := room.currentSessionID(); got != "session-new" {
		t.Fatalf("session id = %q, want %q", got, "session-new")
	}
	if got := room.aiAgentIDValue(); got != "agent-new" {
		t.Fatalf("ai agent id = %q, want %q", got, "agent-new")
	}
	if got := room.gmModeValue(); got != "hybrid" {
		t.Fatalf("gm mode = %q, want %q", got, "hybrid")
	}
}

func TestSyncRoomAIContextFromGameRequiresContext(t *testing.T) {
	err := syncRoomAIContextFromGame(nil, &fakeCampaignAIServiceClient{}, newCampaignRoom("camp-1"))
	if err == nil || err.Error() != "context is required" {
		t.Fatalf("error = %v, want context is required", err)
	}
}

func TestSyncRoomAIContextFromGameReturnsClientError(t *testing.T) {
	err := syncRoomAIContextFromGame(context.Background(), &fakeCampaignAIServiceClient{authStateErr: errors.New("boom")}, newCampaignRoom("camp-1"))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v, want boom", err)
	}
}

func TestIsAICampaignContextEvent(t *testing.T) {
	for _, eventType := range []string{"campaign.ai_bound", "campaign.ai_unbound", "campaign.ai_auth_rotated", "session.started", "session.ended"} {
		if !isAICampaignContextEvent(eventType) {
			t.Fatalf("expected event type %q to be treated as ai campaign context", eventType)
		}
	}
	if isAICampaignContextEvent("campaign.updated") {
		t.Fatal("expected campaign.updated to be ignored")
	}
}
