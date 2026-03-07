package dashboard

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

// handlers implements the dashboard Handlers contract.
type handlers struct {
	base             modulehandler.Base
	statisticsClient statev1.StatisticsServiceClient
	systemClient     statev1.SystemServiceClient
	authClient       authv1.AuthServiceClient
	campaignClient   statev1.CampaignServiceClient
	eventClient      statev1.EventServiceClient
}

// NewHandlers builds the dashboard handler implementation.
func NewHandlers(
	base modulehandler.Base,
	statisticsClient statev1.StatisticsServiceClient,
	systemClient statev1.SystemServiceClient,
	authClient authv1.AuthServiceClient,
	campaignClient statev1.CampaignServiceClient,
	eventClient statev1.EventServiceClient,
) Handlers {
	return handlers{
		base:             base,
		statisticsClient: statisticsClient,
		systemClient:     systemClient,
		authClient:       authClient,
		campaignClient:   campaignClient,
		eventClient:      eventClient,
	}
}

// HandleDashboard renders the dashboard page.
func (s handlers) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.DashboardPage(loc),
		templates.DashboardFullPage(pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.dashboard", templates.AppName()),
	)
}

// HandleDashboardContent renders dashboard statistics and recent activity.
func (s handlers) HandleDashboardContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()
	loc, _ := s.base.Localizer(w, r)

	var gameStats *statev1.GameStatistics
	resp, err := s.statisticsClient.GetGameStatistics(ctx, &statev1.GetGameStatisticsRequest{})
	if err == nil && resp != nil {
		gameStats = resp.GetStats()
	}

	var systemCount int64
	systemsResp, err := s.systemClient.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err == nil && systemsResp != nil {
		systemCount = int64(len(systemsResp.GetSystems()))
	}

	userCount := s.countUsers(r, ctx)

	stats := buildDashboardStats(gameStats, systemCount, userCount)

	activitySvc := newActivityService(s.campaignClient, s.eventClient)
	activities := buildActivityEvents(activitySvc.listRecent(ctx), loc)

	templ.Handler(templates.DashboardContent(stats, activities, loc)).ServeHTTP(w, r)
}

// countUsers paginates through all users to produce a total count.
// Returns -1 on error so the caller can display a zero fallback.
func (s handlers) countUsers(r *http.Request, ctx context.Context) int64 {
	var total int64
	pageToken := ""
	for {
		resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
			PageSize:  50,
			PageToken: pageToken,
		})
		if err != nil || resp == nil {
			adminerrors.LogError(r, "list users for dashboard: %v", err)
			return -1
		}
		total += int64(len(resp.GetUsers()))
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			return total
		}
	}
}
