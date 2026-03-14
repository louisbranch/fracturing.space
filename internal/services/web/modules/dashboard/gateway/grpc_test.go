package gateway

import (
	"context"
	"net/http"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
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
