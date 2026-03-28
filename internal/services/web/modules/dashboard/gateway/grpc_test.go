package gateway

import (
	"context"
	"net/http"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userHubClientStub struct {
	resp       *userhubv1.GetDashboardResponse
	lastReq    *userhubv1.GetDashboardRequest
	lastUserID string
	calls      int
}

func (s *userHubClientStub) GetDashboard(ctx context.Context, req *userhubv1.GetDashboardRequest, _ ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error) {
	s.calls++
	s.lastReq = req
	if md, ok := grpcmetadata.FromOutgoingContext(ctx); ok {
		values := md.Get(grpcmeta.UserIDHeader)
		if len(values) > 0 {
			s.lastUserID = values[0]
		}
	}
	return s.resp, nil
}

func TestNewGRPCGatewayWithoutClientFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil)
	_, err := gateway.LoadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayMapsSnapshotAndMetadata(t *testing.T) {
	t.Parallel()

	client := &userHubClientStub{resp: &userhubv1.GetDashboardResponse{
		User:      &userhubv1.UserSummary{NeedsProfileCompletion: true},
		Campaigns: &userhubv1.CampaignSummary{Campaigns: []*userhubv1.CampaignPreview{{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE}}},
		CampaignStartNudges: &userhubv1.CampaignStartNudgeSummary{Available: true, Nudges: []*userhubv1.CampaignStartNudge{{
			CampaignId:        "camp-2",
			CampaignName:      "Moonwake",
			BlockerCode:       "CHARACTER_SYSTEM_REQUIRED",
			BlockerMessage:    "Finish Aria",
			ActionKind:        userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_COMPLETE_CHARACTER,
			TargetCharacterId: "char-1",
		}}},
		ActiveSessions: &userhubv1.ActiveSessionSummary{Available: true, Sessions: []*userhubv1.ActiveSessionPreview{{CampaignId: "camp-1", CampaignName: "Sunfall", SessionId: "session-1", SessionName: "The Crossing"}}},
		Metadata:       &userhubv1.DashboardMetadata{DegradedDependencies: []string{" social.profile "}},
	}}
	gateway := GRPCGateway{Client: client}
	snapshot, err := gateway.LoadDashboard(context.Background(), "user-1", language.Und)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if !snapshot.NeedsProfileCompletion || !snapshot.HasDraftOrActiveCampaign {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if !snapshot.ActiveSessionsAvailable || len(snapshot.ActiveSessions) != 1 || snapshot.ActiveSessions[0].CampaignID != "camp-1" {
		t.Fatalf("ActiveSessions = %+v", snapshot.ActiveSessions)
	}
	if !snapshot.CampaignStartNudgesAvailable || len(snapshot.CampaignStartNudges) != 1 || snapshot.CampaignStartNudges[0].CampaignID != "camp-2" {
		t.Fatalf("CampaignStartNudges = %+v", snapshot.CampaignStartNudges)
	}
	if snapshot.CampaignStartNudges[0].ActionKind != dashboardapp.CampaignStartNudgeActionKindCompleteCharacter {
		t.Fatalf("CampaignStartNudges[0].ActionKind = %q, want %q", snapshot.CampaignStartNudges[0].ActionKind, dashboardapp.CampaignStartNudgeActionKindCompleteCharacter)
	}
	if snapshot.ActiveSessions[0].SessionID != "session-1" {
		t.Fatalf("ActiveSessions[0].SessionID = %q, want %q", snapshot.ActiveSessions[0].SessionID, "session-1")
	}
	if len(snapshot.DegradedDependencies) != 1 || snapshot.DegradedDependencies[0] != "social.profile" {
		t.Fatalf("DegradedDependencies = %v", snapshot.DegradedDependencies)
	}
	if client.lastReq.GetCampaignPreviewLimit() != MaxDashboardCampaignPreviewLimit {
		t.Fatalf("CampaignPreviewLimit = %d, want %d", client.lastReq.GetCampaignPreviewLimit(), MaxDashboardCampaignPreviewLimit)
	}
	if client.lastReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("Locale = %v, want %v", client.lastReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
	if client.lastUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", client.lastUserID, "user-1")
	}
}

func TestHasDraftOrActiveCampaign(t *testing.T) {
	t.Parallel()

	if !HasDraftOrActiveCampaign([]*userhubv1.CampaignPreview{{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT}}) {
		t.Fatalf("expected draft campaign to count as active")
	}
	if HasDraftOrActiveCampaign([]*userhubv1.CampaignPreview{{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED}}) {
		t.Fatalf("expected completed-only campaigns to return false")
	}
}

func TestLoadDashboardSkipsBlankUserID(t *testing.T) {
	t.Parallel()

	client := &userHubClientStub{resp: &userhubv1.GetDashboardResponse{}}
	gateway := GRPCGateway{Client: client}
	snapshot, err := gateway.LoadDashboard(context.Background(), "   ", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if snapshot.NeedsProfileCompletion || snapshot.HasDraftOrActiveCampaign || snapshot.CampaignsHasMore || snapshot.ActiveSessionsAvailable || len(snapshot.ActiveSessions) > 0 || len(snapshot.DegradedDependencies) > 0 {
		t.Fatalf("snapshot = %+v, want zero value", snapshot)
	}
	if client.calls != 0 {
		t.Fatalf("client calls = %d, want 0", client.calls)
	}
}

func TestLoadDashboardNormalizesUserID(t *testing.T) {
	t.Parallel()

	client := &userHubClientStub{resp: &userhubv1.GetDashboardResponse{}}
	gateway := GRPCGateway{Client: client}
	if _, err := gateway.LoadDashboard(context.Background(), " user-7 ", language.AmericanEnglish); err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("client calls = %d, want 1", client.calls)
	}
	if client.lastUserID != "user-7" {
		t.Fatalf("user id = %q, want %q", client.lastUserID, "user-7")
	}
}

func TestCampaignStartNudgeActionKindFromProtoMapsStartSession(t *testing.T) {
	t.Parallel()

	got := campaignStartNudgeActionKindFromProto(userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_START_SESSION)
	if got != dashboardapp.CampaignStartNudgeActionKindStartSession {
		t.Fatalf("action kind = %q, want %q", got, dashboardapp.CampaignStartNudgeActionKindStartSession)
	}
}

func TestDashboardGatewayHelperMappings(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 20, 0, 0, 0, time.UTC)

	invites := mapPendingInvites([]*userhubv1.PendingInvite{
		nil,
		{
			InviteId:        "invite-1",
			CampaignName:    "Moonwake",
			ParticipantName: "Aria",
		},
	})
	if len(invites) != 1 || invites[0].InviteID != "invite-1" {
		t.Fatalf("mapPendingInvites() = %#v", invites)
	}
	if got := mapPendingInvites(nil); got != nil {
		t.Fatalf("mapPendingInvites(nil) = %#v, want nil", got)
	}

	sessions := mapActiveSessions([]*userhubv1.ActiveSessionPreview{
		nil,
		{
			CampaignId:   "camp-1",
			CampaignName: "Sunfall",
			SessionId:    "sess-1",
			SessionName:  "The Crossing",
		},
	})
	if len(sessions) != 1 || sessions[0].SessionID != "sess-1" {
		t.Fatalf("mapActiveSessions() = %#v", sessions)
	}

	nudges := mapCampaignStartNudges([]*userhubv1.CampaignStartNudge{
		nil,
		{
			CampaignId:          "camp-2",
			CampaignName:        "Skyline",
			BlockerCode:         "START_SESSION_STALE",
			BlockerMessage:      "Start a new session.",
			ActionKind:          userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_START_SESSION,
			TargetParticipantId: "p-1",
			TargetCharacterId:   "char-1",
		},
	})
	if len(nudges) != 1 || nudges[0].ActionKind != dashboardapp.CampaignStartNudgeActionKindStartSession {
		t.Fatalf("mapCampaignStartNudges() = %#v", nudges)
	}

	if got := mapFreshness(userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_STALE); got != dashboardapp.DashboardFreshnessStale {
		t.Fatalf("mapFreshness(stale) = %v, want %v", got, dashboardapp.DashboardFreshnessStale)
	}
	if got := mapFreshness(userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_UNSPECIFIED); got != dashboardapp.DashboardFreshnessUnspecified {
		t.Fatalf("mapFreshness(unspecified) = %v, want %v", got, dashboardapp.DashboardFreshnessUnspecified)
	}
	if got := protoTime(nil); !got.IsZero() {
		t.Fatalf("protoTime(nil) = %v, want zero time", got)
	}
	if got := protoTime(timestamppb.New(now)); !got.Equal(now) {
		t.Fatalf("protoTime(now) = %v, want %v", got, now)
	}
	if got := normalizedDependencies([]string{" social.profile ", "", "worker.queue "}); len(got) != 2 || got[0] != "social.profile" || got[1] != "worker.queue" {
		t.Fatalf("normalizedDependencies() = %#v", got)
	}
	if got := normalizedDependencies([]string{"", "   "}); got != nil {
		t.Fatalf("normalizedDependencies(empty) = %#v, want nil", got)
	}
}
