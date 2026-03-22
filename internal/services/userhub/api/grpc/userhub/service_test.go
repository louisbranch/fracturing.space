package userhub

import (
	"context"
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetDashboardRequiresUserIdentity(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeDashboardDomain{})
	_, err := svc.GetDashboard(context.Background(), &userhubv1.GetDashboardRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func TestGetDashboardMapsDomainResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 30, 0, 0, time.UTC)
	fakeDomain := &fakeDashboardDomain{
		result: domain.Dashboard{
			Metadata: domain.DashboardMetadata{
				Freshness:            domain.FreshnessFresh,
				CacheHit:             false,
				Degraded:             false,
				DegradedDependencies: nil,
				GeneratedAt:          now,
			},
			User: domain.UserSummary{
				UserID:                 "user-1",
				Username:               "ari",
				Name:                   "Ari",
				ProfileAvailable:       true,
				Discoverable:           true,
				NeedsProfileCompletion: false,
			},
			Invites: domain.InviteSummary{
				Available:   true,
				ListedCount: 1,
				Pending: []domain.PendingInvite{{
					InviteID:      "inv-1",
					CampaignID:    "camp-1",
					CampaignName:  "Sunfall",
					ParticipantID: "part-1",
					CreatedAt:     now.Add(-5 * time.Minute),
				}},
			},
			Notifications: domain.NotificationSummary{
				Available:   true,
				HasUnread:   true,
				UnreadCount: 3,
			},
			Campaigns: domain.CampaignSummary{
				Available:   true,
				ListedCount: 1,
				ActiveCount: 1,
				Campaigns: []domain.CampaignPreview{{
					CampaignID:       "camp-1",
					Name:             "Sunfall",
					Status:           domain.CampaignStatusActive,
					ParticipantCount: 4,
					CharacterCount:   4,
					UpdatedAt:        now.Add(-1 * time.Hour),
				}},
			},
			CampaignStartNudges: domain.CampaignStartNudgeSummary{
				Available:   true,
				ListedCount: 1,
				HasMore:     false,
				Nudges: []domain.CampaignStartNudge{{
					CampaignID:          "camp-1",
					CampaignName:        "Sunfall",
					CampaignUpdatedAt:   now.Add(-30 * time.Minute),
					BlockerCode:         "SESSION_READINESS_PLAYER_CHARACTER_REQUIRED",
					BlockerMessage:      "Create a character before starting.",
					ActionKind:          domain.CampaignStartNudgeActionCreateCharacter,
					TargetParticipantID: "part-1",
				}},
			},
			ActiveSessions: domain.ActiveSessionSummary{
				Available:   true,
				ListedCount: 1,
				Sessions: []domain.ActiveSessionPreview{{
					CampaignID:   "camp-1",
					CampaignName: "Sunfall",
					SessionID:    "session-1",
					SessionName:  "The Crossing",
					StartedAt:    now.Add(-10 * time.Minute),
				}},
			},
			NextActions: []domain.DashboardAction{{
				ID:       domain.DashboardActionReviewPendingInvites,
				Priority: 100,
			}},
		},
	}
	svc := NewService(fakeDomain)
	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(grpcmeta.UserIDHeader, "user-1"))

	resp, err := svc.GetDashboard(ctx, &userhubv1.GetDashboardRequest{
		Locale:               commonv1.Locale_LOCALE_PT_BR,
		CampaignPreviewLimit: 5,
		InvitePreviewLimit:   4,
	})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if fakeDomain.lastInput.UserID != "user-1" {
		t.Fatalf("domain user id = %q, want %q", fakeDomain.lastInput.UserID, "user-1")
	}
	if fakeDomain.lastInput.Locale != commonv1.Locale_LOCALE_PT_BR.String() {
		t.Fatalf("domain locale = %q, want %q", fakeDomain.lastInput.Locale, commonv1.Locale_LOCALE_PT_BR.String())
	}
	if resp.GetMetadata().GetFreshness() != userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_FRESH {
		t.Fatalf("freshness = %v, want %v", resp.GetMetadata().GetFreshness(), userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_FRESH)
	}
	if got := len(resp.GetNextActions()); got != 1 {
		t.Fatalf("actions len = %d, want 1", got)
	}
	if got := resp.GetNextActions()[0].GetId(); got != userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_PENDING_INVITES {
		t.Fatalf("action id = %v, want %v", got, userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_PENDING_INVITES)
	}
	if got := resp.GetActiveSessions().GetSessions(); len(got) != 1 || got[0].GetSessionId() != "session-1" {
		t.Fatalf("active sessions = %+v, want one session-1 entry", got)
	}
	if got := resp.GetCampaignStartNudges().GetNudges(); len(got) != 1 {
		t.Fatalf("campaign start nudges = %+v, want one entry", got)
	} else {
		if got[0].GetActionKind() != userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_CREATE_CHARACTER {
			t.Fatalf("nudge action kind = %v, want %v", got[0].GetActionKind(), userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_CREATE_CHARACTER)
		}
		if got[0].GetTargetParticipantId() != "part-1" {
			t.Fatalf("nudge target participant id = %q, want %q", got[0].GetTargetParticipantId(), "part-1")
		}
	}
}

func TestGetDashboardMapsDependencyErrorToUnavailable(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeDashboardDomain{
		err: &domain.DependencyUnavailableError{Dependency: "game.campaigns", Err: errors.New("boom")},
	})
	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(grpcmeta.UserIDHeader, "user-1"))

	_, err := svc.GetDashboard(ctx, &userhubv1.GetDashboardRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.Unavailable)
	}
}

func TestGetDashboardRejectsNilRequest(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeDashboardDomain{})
	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(grpcmeta.UserIDHeader, "user-1"))

	_, err := svc.GetDashboard(ctx, nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCampaignStartNudgeActionKindToProtoMapsStartSession(t *testing.T) {
	t.Parallel()

	got := campaignStartNudgeActionKindToProto(domain.CampaignStartNudgeActionStartSession)
	if got != userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_START_SESSION {
		t.Fatalf("action kind = %v, want %v", got, userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_START_SESSION)
	}
}

type fakeDashboardDomain struct {
	result    domain.Dashboard
	err       error
	lastInput domain.GetDashboardInput
}

func (f *fakeDashboardDomain) GetDashboard(_ context.Context, input domain.GetDashboardInput) (domain.Dashboard, error) {
	f.lastInput = input
	if f.err != nil {
		return domain.Dashboard{}, f.err
	}
	return f.result, nil
}

var _ dashboardGetter = (*fakeDashboardDomain)(nil)
