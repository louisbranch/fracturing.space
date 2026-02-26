package domain

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestGetDashboardRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeGameGateway{}, &fakeSocialGateway{}, &fakeNotificationsGateway{}, Config{})
	_, err := svc.GetDashboard(context.Background(), GetDashboardInput{})
	if !errors.Is(err, ErrUserIDRequired) {
		t.Fatalf("GetDashboard error = %v, want %v", err, ErrUserIDRequired)
	}
}

func TestGetDashboardBuildsAggregateAndActions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 0, 0, 0, time.UTC)
	clockNow := now
	game := &fakeGameGateway{
		campaignPage: CampaignPage{
			Campaigns: []CampaignPreview{
				{
					CampaignID:       "camp-1",
					Name:             "Sunfall",
					Status:           CampaignStatusActive,
					ParticipantCount: 4,
					CharacterCount:   4,
					UpdatedAt:        now.Add(-1 * time.Hour),
				},
			},
		},
		invitePage: InvitePage{
			Invites: []PendingInvite{{
				InviteID:      "inv-1",
				CampaignID:    "camp-1",
				CampaignName:  "Sunfall",
				ParticipantID: "part-2",
				CreatedAt:     now.Add(-30 * time.Minute),
			}},
		},
	}
	social := &fakeSocialGateway{
		profile: UserProfile{Username: "", Name: "Ari"},
	}
	notifications := &fakeNotificationsGateway{
		status: UnreadStatus{HasUnread: true, UnreadCount: 2},
	}

	svc := NewService(game, social, notifications, Config{Clock: func() time.Time { return clockNow }})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: " user-1 "})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if dashboard.Metadata.Freshness != FreshnessFresh {
		t.Fatalf("freshness = %v, want %v", dashboard.Metadata.Freshness, FreshnessFresh)
	}
	if dashboard.Metadata.CacheHit {
		t.Fatal("cache_hit = true, want false")
	}
	if dashboard.User.Discoverable {
		t.Fatal("discoverable = true, want false")
	}
	if !dashboard.User.NeedsProfileCompletion {
		t.Fatal("needs_profile_completion = false, want true")
	}
	if dashboard.Campaigns.ActiveCount != 1 {
		t.Fatalf("active_count = %d, want 1", dashboard.Campaigns.ActiveCount)
	}
	if dashboard.Notifications.UnreadCount != 2 {
		t.Fatalf("unread_count = %d, want 2", dashboard.Notifications.UnreadCount)
	}

	gotIDs := make([]DashboardActionID, 0, len(dashboard.NextActions))
	for _, action := range dashboard.NextActions {
		gotIDs = append(gotIDs, action.ID)
	}
	wantIDs := []DashboardActionID{
		DashboardActionReviewPendingInvites,
		DashboardActionCompleteProfile,
		DashboardActionContinueActiveCampaign,
		DashboardActionReviewNotifications,
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("action ids = %v, want %v", gotIDs, wantIDs)
	}
}

func TestGetDashboardUsesFreshCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 10, 0, 0, time.UTC)
	clockNow := now
	game := &fakeGameGateway{campaignPage: CampaignPage{Campaigns: []CampaignPreview{{CampaignID: "camp-1", Status: CampaignStatusActive}}}, invitePage: InvitePage{}}
	social := &fakeSocialGateway{profile: UserProfile{Username: "discoverable", Name: "Ari"}}
	notifications := &fakeNotificationsGateway{status: UnreadStatus{HasUnread: false, UnreadCount: 0}}
	svc := NewService(game, social, notifications, Config{
		Clock:         func() time.Time { return clockNow },
		CacheFreshTTL: 15 * time.Second,
		CacheStaleTTL: time.Minute,
	})

	if _, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"}); err != nil {
		t.Fatalf("first GetDashboard error: %v", err)
	}
	clockNow = now.Add(5 * time.Second)
	if _, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"}); err != nil {
		t.Fatalf("second GetDashboard error: %v", err)
	}

	if game.campaignCalls != 1 {
		t.Fatalf("campaign calls = %d, want 1", game.campaignCalls)
	}
	if social.calls != 1 {
		t.Fatalf("social calls = %d, want 1", social.calls)
	}
	if notifications.calls != 1 {
		t.Fatalf("notification calls = %d, want 1", notifications.calls)
	}
}

func TestGetDashboardFallsBackToStaleOnDependencyError(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 20, 0, 0, time.UTC)
	clockNow := now
	game := &fakeGameGateway{campaignPage: CampaignPage{Campaigns: []CampaignPreview{{CampaignID: "camp-1", Status: CampaignStatusActive}}}, invitePage: InvitePage{}}
	social := &fakeSocialGateway{profile: UserProfile{Username: "discoverable", Name: "Ari"}}
	notifications := &fakeNotificationsGateway{status: UnreadStatus{HasUnread: false, UnreadCount: 0}}
	svc := NewService(game, social, notifications, Config{
		Clock:         func() time.Time { return clockNow },
		CacheFreshTTL: 10 * time.Second,
		CacheStaleTTL: time.Minute,
	})

	seed, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("seed GetDashboard error: %v", err)
	}
	clockNow = now.Add(20 * time.Second)
	notifications.err = errors.New("notifications unavailable")

	stale, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("fallback GetDashboard error: %v", err)
	}
	if stale.Metadata.Freshness != FreshnessStale {
		t.Fatalf("freshness = %v, want %v", stale.Metadata.Freshness, FreshnessStale)
	}
	if !stale.Metadata.CacheHit {
		t.Fatal("cache_hit = false, want true")
	}
	if !stale.Metadata.Degraded {
		t.Fatal("degraded = false, want true")
	}
	if !contains(stale.Metadata.DegradedDependencies, dependencyNotificationsRead) {
		t.Fatalf("degraded_dependencies = %v, want %q", stale.Metadata.DegradedDependencies, dependencyNotificationsRead)
	}
	if stale.Metadata.GeneratedAt != seed.Metadata.GeneratedAt {
		t.Fatalf("generated_at = %v, want %v", stale.Metadata.GeneratedAt, seed.Metadata.GeneratedAt)
	}
}

func TestGetDashboardReturnsCriticalDependencyErrorWithoutStale(t *testing.T) {
	t.Parallel()

	game := &fakeGameGateway{campaignErr: errors.New("game unavailable")}
	svc := NewService(game, &fakeSocialGateway{}, &fakeNotificationsGateway{}, Config{})

	_, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	var dependencyErr *DependencyUnavailableError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("error = %T, want %T", err, &DependencyUnavailableError{})
	}
	if dependencyErr.Dependency != dependencyGameCampaigns {
		t.Fatalf("dependency = %q, want %q", dependencyErr.Dependency, dependencyGameCampaigns)
	}
}

func TestGetDashboardHandlesProfileNotFoundWithoutDegrading(t *testing.T) {
	t.Parallel()

	game := &fakeGameGateway{campaignPage: CampaignPage{}, invitePage: InvitePage{}}
	social := &fakeSocialGateway{err: ErrProfileNotFound}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(game, social, notifications, Config{})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if dashboard.Metadata.Degraded {
		t.Fatalf("degraded = true, want false (%v)", dashboard.Metadata.DegradedDependencies)
	}
	if dashboard.User.ProfileAvailable {
		t.Fatal("profile_available = true, want false")
	}
	if !dashboard.User.NeedsProfileCompletion {
		t.Fatal("needs_profile_completion = false, want true")
	}
}

type fakeGameGateway struct {
	campaignPage  CampaignPage
	campaignErr   error
	campaignCalls int

	invitePage  InvitePage
	inviteErr   error
	inviteCalls int
}

func (f *fakeGameGateway) ListCampaignPreviews(_ context.Context, _ string, _ int) (CampaignPage, error) {
	f.campaignCalls++
	if f.campaignErr != nil {
		return CampaignPage{}, f.campaignErr
	}
	return f.campaignPage, nil
}

func (f *fakeGameGateway) ListPendingInvitePreviews(_ context.Context, _ string, _ int) (InvitePage, error) {
	f.inviteCalls++
	if f.inviteErr != nil {
		return InvitePage{}, f.inviteErr
	}
	return f.invitePage, nil
}

type fakeSocialGateway struct {
	profile UserProfile
	err     error
	calls   int
}

func (f *fakeSocialGateway) GetUserProfile(_ context.Context, _ string) (UserProfile, error) {
	f.calls++
	if f.err != nil {
		return UserProfile{}, f.err
	}
	return f.profile, nil
}

type fakeNotificationsGateway struct {
	status UnreadStatus
	err    error
	calls  int
}

func (f *fakeNotificationsGateway) GetUnreadStatus(_ context.Context, _ string) (UnreadStatus, error) {
	f.calls++
	if f.err != nil {
		return UnreadStatus{}, f.err
	}
	return f.status, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
