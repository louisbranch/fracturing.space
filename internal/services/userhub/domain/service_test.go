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

	svc := NewService(&fakeAuthGateway{}, &fakeGameGateway{}, &fakeSocialGateway{}, &fakeNotificationsGateway{}, Config{})
	_, err := svc.GetDashboard(context.Background(), GetDashboardInput{})
	if !errors.Is(err, ErrUserIDRequired) {
		t.Fatalf("GetDashboard error = %v, want %v", err, ErrUserIDRequired)
	}
}

func TestGetDashboardBuildsAggregateAndActions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 0, 0, 0, time.UTC)
	clockNow := now
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "ari"}}
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
		activeSessionPage: ActiveSessionPage{
			Sessions: []ActiveSessionPreview{{
				CampaignID:   "camp-1",
				CampaignName: "Sunfall",
				SessionID:    "session-1",
				SessionName:  "The Crossing",
				StartedAt:    now.Add(-15 * time.Minute),
			}},
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
		profile: UserProfile{Name: "Ari"},
	}
	notifications := &fakeNotificationsGateway{
		status: UnreadStatus{HasUnread: true, UnreadCount: 2},
	}

	svc := NewService(auth, game, social, notifications, Config{Clock: func() time.Time { return clockNow }})

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
	if !dashboard.User.Discoverable {
		t.Fatal("discoverable = false, want true")
	}
	if dashboard.User.NeedsProfileCompletion {
		t.Fatal("needs_profile_completion = true, want false")
	}
	if dashboard.Campaigns.ActiveCount != 1 {
		t.Fatalf("active_count = %d, want 1", dashboard.Campaigns.ActiveCount)
	}
	if !dashboard.ActiveSessions.Available {
		t.Fatal("active_sessions.available = false, want true")
	}
	if dashboard.ActiveSessions.ListedCount != 1 {
		t.Fatalf("active_sessions.listed_count = %d, want 1", dashboard.ActiveSessions.ListedCount)
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
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage:      CampaignPage{Campaigns: []CampaignPreview{{CampaignID: "camp-1", Status: CampaignStatusActive}}},
		activeSessionPage: ActiveSessionPage{Sessions: []ActiveSessionPreview{{CampaignID: "camp-1", SessionID: "session-1"}}},
		invitePage:        InvitePage{},
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{status: UnreadStatus{HasUnread: false, UnreadCount: 0}}
	svc := NewService(auth, game, social, notifications, Config{
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
	if auth.calls != 1 {
		t.Fatalf("auth calls = %d, want 1", auth.calls)
	}
	if social.calls != 1 {
		t.Fatalf("social calls = %d, want 1", social.calls)
	}
	if notifications.calls != 1 {
		t.Fatalf("notification calls = %d, want 1", notifications.calls)
	}
	if game.activeSessionCalls != 1 {
		t.Fatalf("active session calls = %d, want 1", game.activeSessionCalls)
	}
}

func TestGetDashboardFallsBackToStaleOnDependencyError(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 26, 4, 20, 0, 0, time.UTC)
	clockNow := now
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage:      CampaignPage{Campaigns: []CampaignPreview{{CampaignID: "camp-1", Status: CampaignStatusActive}}},
		activeSessionPage: ActiveSessionPage{Sessions: []ActiveSessionPreview{{CampaignID: "camp-1", SessionID: "session-1"}}},
		invitePage:        InvitePage{},
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{status: UnreadStatus{HasUnread: false, UnreadCount: 0}}
	svc := NewService(auth, game, social, notifications, Config{
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
	svc := NewService(&fakeAuthGateway{}, game, &fakeSocialGateway{}, &fakeNotificationsGateway{}, Config{})

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

	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{campaignPage: CampaignPage{}, invitePage: InvitePage{}}
	social := &fakeSocialGateway{err: ErrProfileNotFound}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(auth, game, social, notifications, Config{})

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

func TestGetDashboardNeedsProfileCompletionWhenSocialNameBlank(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}},
		&fakeGameGateway{campaignPage: CampaignPage{}, invitePage: InvitePage{}},
		&fakeSocialGateway{profile: UserProfile{Name: "   "}},
		&fakeNotificationsGateway{},
		Config{},
	)

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if !dashboard.User.Discoverable {
		t.Fatal("discoverable = false, want true")
	}
	if !dashboard.User.NeedsProfileCompletion {
		t.Fatal("needs_profile_completion = false, want true")
	}
}

func TestGetDashboardDegradesActiveSessionsWithoutFailingWholeDashboard(t *testing.T) {
	t.Parallel()

	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage: CampaignPage{
			Campaigns: []CampaignPreview{{CampaignID: "camp-1", Status: CampaignStatusActive}},
		},
		activeSessionErr: errors.New("sessions unavailable"),
		invitePage:       InvitePage{},
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(auth, game, social, notifications, Config{})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if dashboard.ActiveSessions.Available {
		t.Fatal("active_sessions.available = true, want false")
	}
	if !dashboard.Metadata.Degraded {
		t.Fatal("degraded = false, want true")
	}
	if !contains(dashboard.Metadata.DegradedDependencies, dependencyGameSessions) {
		t.Fatalf("degraded_dependencies = %v, want %q", dashboard.Metadata.DegradedDependencies, dependencyGameSessions)
	}
}

func TestGetDashboardIncludesContinueActionWhenActiveSessionFallsOutsidePreviewPage(t *testing.T) {
	t.Parallel()

	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage: CampaignPage{
			Campaigns: []CampaignPreview{{
				CampaignID: "camp-completed",
				Status:     CampaignStatusCompleted,
			}},
		},
		activeSessionPage: ActiveSessionPage{
			Sessions: []ActiveSessionPreview{{
				CampaignID:   "camp-active",
				CampaignName: "Sunfall",
				SessionID:    "session-1",
			}},
		},
		invitePage: InvitePage{},
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(auth, game, social, notifications, Config{})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1", CampaignPreviewLimit: 1})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if !containsAction(dashboard.NextActions, DashboardActionContinueActiveCampaign) {
		t.Fatalf("actions = %+v, want continue active campaign action", dashboard.NextActions)
	}
}

func TestGetDashboardIncludesCampaignStartNudgesForCurrentUser(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage: CampaignPage{},
		invitePage:   InvitePage{},
		readinessCampaigns: []CampaignPreview{
			{CampaignID: "camp-older", Name: "Older", Status: CampaignStatusDraft, UpdatedAt: now.Add(-2 * time.Hour)},
			{CampaignID: "camp-newer", Name: "Newer", Status: CampaignStatusActive, UpdatedAt: now.Add(-1 * time.Hour)},
		},
		readinessByCampaign: map[string]CampaignReadiness{
			"camp-older": {
				Blockers: []CampaignReadinessBlocker{{
					Code:                "CHARACTER_SYSTEM_REQUIRED",
					Message:             "Complete a character",
					ResponsibleUserIDs:  []string{"user-1"},
					ActionKind:          CampaignStartNudgeActionCreateCharacter,
					TargetParticipantID: "part-1",
				}},
			},
			"camp-newer": {
				Blockers: []CampaignReadinessBlocker{
					{
						Code:               "CHARACTER_SYSTEM_REQUIRED",
						Message:            "Someone else should complete a character",
						ResponsibleUserIDs: []string{"user-2"},
						ActionKind:         CampaignStartNudgeActionCreateCharacter,
					},
					{
						Code:               "CHARACTER_SYSTEM_REQUIRED",
						Message:            "Finish Aria",
						ResponsibleUserIDs: []string{"user-1"},
						ActionKind:         CampaignStartNudgeActionCompleteCharacter,
						TargetCharacterID:  "char-1",
					},
				},
			},
		},
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(auth, game, social, notifications, Config{})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1", CampaignPreviewLimit: 1})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if !dashboard.CampaignStartNudges.Available {
		t.Fatal("campaign_start_nudges.available = false, want true")
	}
	if dashboard.CampaignStartNudges.ListedCount != 1 {
		t.Fatalf("listed_count = %d, want 1", dashboard.CampaignStartNudges.ListedCount)
	}
	if !dashboard.CampaignStartNudges.HasMore {
		t.Fatal("has_more = false, want true")
	}
	nudge := dashboard.CampaignStartNudges.Nudges[0]
	if nudge.CampaignID != "camp-newer" {
		t.Fatalf("campaign id = %q, want %q", nudge.CampaignID, "camp-newer")
	}
	if nudge.ActionKind != CampaignStartNudgeActionCompleteCharacter {
		t.Fatalf("action kind = %v, want %v", nudge.ActionKind, CampaignStartNudgeActionCompleteCharacter)
	}
	if nudge.TargetCharacterID != "char-1" {
		t.Fatalf("target_character_id = %q, want %q", nudge.TargetCharacterID, "char-1")
	}
}

func TestGetDashboardIncludesStartSessionNudgeForReadyStaleCampaign(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	oldSession := now.Add(-8 * 24 * time.Hour)
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage: CampaignPage{},
		invitePage:   InvitePage{},
		readinessCampaigns: []CampaignPreview{{
			CampaignID:       "camp-ready",
			Name:             "Sunfall",
			Status:           CampaignStatusActive,
			UpdatedAt:        now.Add(-2 * time.Hour),
			LatestSessionAt:  &oldSession,
			CanManageSession: true,
		}},
		readinessByCampaign: map[string]CampaignReadiness{
			"camp-ready": {},
		},
	}
	svc := NewService(auth, game, &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}, &fakeNotificationsGateway{}, Config{
		Clock: func() time.Time { return now },
	})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if len(dashboard.CampaignStartNudges.Nudges) != 1 {
		t.Fatalf("nudges = %+v, want one entry", dashboard.CampaignStartNudges.Nudges)
	}
	nudge := dashboard.CampaignStartNudges.Nudges[0]
	if nudge.ActionKind != CampaignStartNudgeActionStartSession {
		t.Fatalf("action kind = %v, want %v", nudge.ActionKind, CampaignStartNudgeActionStartSession)
	}
	if nudge.BlockerCode != "START_SESSION_STALE" {
		t.Fatalf("blocker code = %q, want %q", nudge.BlockerCode, "START_SESSION_STALE")
	}
}

func TestGetDashboardSkipsStartSessionNudgeWhenCampaignIsRecentOrUnauthorized(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	recentSession := now.Add(-2 * 24 * time.Hour)
	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage: CampaignPage{},
		invitePage:   InvitePage{},
		readinessCampaigns: []CampaignPreview{
			{
				CampaignID:       "camp-recent",
				Name:             "Recent",
				Status:           CampaignStatusActive,
				UpdatedAt:        now.Add(-2 * time.Hour),
				LatestSessionAt:  &recentSession,
				CanManageSession: true,
			},
			{
				CampaignID:       "camp-no-access",
				Name:             "No Access",
				Status:           CampaignStatusDraft,
				UpdatedAt:        now.Add(-3 * time.Hour),
				CanManageSession: false,
			},
		},
		readinessByCampaign: map[string]CampaignReadiness{
			"camp-recent":    {},
			"camp-no-access": {},
		},
	}
	svc := NewService(auth, game, &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}, &fakeNotificationsGateway{}, Config{
		Clock: func() time.Time { return now },
	})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if len(dashboard.CampaignStartNudges.Nudges) != 0 {
		t.Fatalf("nudges = %+v, want none", dashboard.CampaignStartNudges.Nudges)
	}
}

func TestGetDashboardDegradesReadinessWithoutFailingWholeDashboard(t *testing.T) {
	t.Parallel()

	auth := &fakeAuthGateway{identity: UserIdentity{Username: "discoverable"}}
	game := &fakeGameGateway{
		campaignPage:          CampaignPage{},
		invitePage:            InvitePage{},
		readinessCampaignsErr: errors.New("readiness unavailable"),
	}
	social := &fakeSocialGateway{profile: UserProfile{Name: "Ari"}}
	notifications := &fakeNotificationsGateway{}
	svc := NewService(auth, game, social, notifications, Config{})

	dashboard, err := svc.GetDashboard(context.Background(), GetDashboardInput{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetDashboard error: %v", err)
	}
	if dashboard.CampaignStartNudges.Available {
		t.Fatal("campaign_start_nudges.available = true, want false")
	}
	if !dashboard.Metadata.Degraded {
		t.Fatal("degraded = false, want true")
	}
	if !contains(dashboard.Metadata.DegradedDependencies, dependencyGameReadiness) {
		t.Fatalf("degraded_dependencies = %v, want %q", dashboard.Metadata.DegradedDependencies, dependencyGameReadiness)
	}
}

type fakeGameGateway struct {
	campaignPage  CampaignPage
	campaignErr   error
	campaignCalls int

	readinessCampaigns     []CampaignPreview
	readinessCampaignsErr  error
	readinessCampaignCalls int
	readinessByCampaign    map[string]CampaignReadiness
	readinessErr           error
	readinessCalls         int

	activeSessionPage  ActiveSessionPage
	activeSessionErr   error
	activeSessionCalls int

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

func (f *fakeGameGateway) ListReadinessCampaigns(_ context.Context, _ string) ([]CampaignPreview, error) {
	f.readinessCampaignCalls++
	if f.readinessCampaignsErr != nil {
		return nil, f.readinessCampaignsErr
	}
	return append([]CampaignPreview{}, f.readinessCampaigns...), nil
}

func (f *fakeGameGateway) GetCampaignReadiness(_ context.Context, _ string, campaignID string) (CampaignReadiness, error) {
	f.readinessCalls++
	if f.readinessErr != nil {
		return CampaignReadiness{}, f.readinessErr
	}
	return f.readinessByCampaign[campaignID], nil
}

func (f *fakeGameGateway) ListPendingInvitePreviews(_ context.Context, _ string, _ int) (InvitePage, error) {
	f.inviteCalls++
	if f.inviteErr != nil {
		return InvitePage{}, f.inviteErr
	}
	return f.invitePage, nil
}

func (f *fakeGameGateway) ListActiveSessionPreviews(_ context.Context, _ string, _ int) (ActiveSessionPage, error) {
	f.activeSessionCalls++
	if f.activeSessionErr != nil {
		return ActiveSessionPage{}, f.activeSessionErr
	}
	return f.activeSessionPage, nil
}

type fakeAuthGateway struct {
	identity UserIdentity
	err      error
	calls    int
}

func (f *fakeAuthGateway) GetUserIdentity(_ context.Context, _ string) (UserIdentity, error) {
	f.calls++
	if f.err != nil {
		return UserIdentity{}, f.err
	}
	return f.identity, nil
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

func containsAction(actions []DashboardAction, want DashboardActionID) bool {
	for _, action := range actions {
		if action.ID == want {
			return true
		}
	}
	return false
}
