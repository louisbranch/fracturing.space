package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"golang.org/x/text/language"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	readGateway    Gateway
	logger         *slog.Logger
	healthProvider HealthProvider
}

// NewService constructs a dashboard service with fail-closed gateway defaults.
func NewService(gateway Gateway, logger *slog.Logger, health HealthProvider) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return service{readGateway: gateway, logger: logger, healthProvider: health}
}

// loadHealth resolves optional service-health entries for dashboard rendering.
func (s service) loadHealth(ctx context.Context) []ServiceHealthEntry {
	if s.healthProvider == nil {
		return nil
	}
	return s.healthProvider(ctx)
}

// LoadDashboard loads the package state needed for this request path.
func (s service) LoadDashboard(ctx context.Context, userID string, locale language.Tag) (DashboardView, error) {
	userID = userid.Normalize(userID)
	if userID == "" {
		return DashboardView{DataStatus: DashboardDataStatusAnonymous}, nil
	}
	snapshot, err := s.readGateway.LoadDashboard(ctx, userID, locale)
	if err != nil {
		s.logger.Warn("dashboard unavailable", "user_id", userID, "error", err)
		return DashboardView{
			DataStatus:    DashboardDataStatusUnavailable,
			ServiceHealth: s.loadHealth(ctx),
		}, nil
	}
	if snapshot.Freshness != DashboardFreshnessUnspecified || snapshot.CacheHit || !snapshot.GeneratedAt.IsZero() {
		s.logger.Info(
			"dashboard freshness",
			"freshness", snapshot.Freshness,
			"cache_hit", snapshot.CacheHit,
			"generated_at", snapshot.GeneratedAt.UTC().Format(time.RFC3339),
			"user_id", userID,
		)
	}
	activeSessions := []ActiveSessionItem(nil)
	if snapshot.ActiveSessionsAvailable && !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencyGameSessions) {
		activeSessions = append(activeSessions, snapshot.ActiveSessions...)
	}
	pendingInvites := []PendingInviteItem(nil)
	if snapshot.InvitesAvailable && !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencyGameInvites) {
		pendingInvites = append(pendingInvites, snapshot.PendingInvites...)
	}
	campaignStartNudges := []CampaignStartNudgeItem(nil)
	campaignStartNudgesMore := false
	if snapshot.CampaignStartNudgesAvailable && !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencyGameReadiness) {
		campaignStartNudges = append(campaignStartNudges, snapshot.CampaignStartNudges...)
		campaignStartNudgesMore = snapshot.CampaignStartNudgesHasMore
	}
	if HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencySocialProfile) {
		s.logger.Warn("dashboard degraded dependency", "dependency", DegradedDependencySocialProfile, "user_id", userID)
	}
	health := s.loadHealth(ctx)
	showAdventureBlock := false
	if len(activeSessions) == 0 && !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencyGameCampaigns) {
		showAdventureBlock = !snapshot.HasDraftOrActiveCampaign && !snapshot.CampaignsHasMore
	}
	status := DashboardDataStatusReady
	if len(snapshot.DegradedDependencies) > 0 {
		status = DashboardDataStatusDegraded
	}
	return DashboardView{
		DataStatus:              status,
		DegradedDependencies:    snapshot.DegradedDependencies,
		ShowPendingProfileBlock: snapshot.NeedsProfileCompletion && !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencySocialProfile),
		PendingInvites:          pendingInvites,
		ShowAdventureBlock:      showAdventureBlock,
		CampaignStartNudges:     campaignStartNudges,
		CampaignStartNudgesMore: campaignStartNudgesMore,
		ActiveSessions:          activeSessions,
		ServiceHealth:           health,
	}, nil
}

// HasDegradedDependency reports whether a degraded dependency marker is present.
func HasDegradedDependency(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
