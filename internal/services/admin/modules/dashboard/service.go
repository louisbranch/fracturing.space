package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

// service implements dashboard module handlers using shared module dependencies.
type service struct {
	base             modulehandler.Base
	statisticsClient statev1.StatisticsServiceClient
	systemClient     statev1.SystemServiceClient
	authClient       authv1.AuthServiceClient
	campaignClient   statev1.CampaignServiceClient
	eventClient      statev1.EventServiceClient
}

// NewService builds the dashboard module service implementation.
func NewService(
	base modulehandler.Base,
	statisticsClient statev1.StatisticsServiceClient,
	systemClient statev1.SystemServiceClient,
	authClient authv1.AuthServiceClient,
	campaignClient statev1.CampaignServiceClient,
	eventClient statev1.EventServiceClient,
) Service {
	return service{
		base:             base,
		statisticsClient: statisticsClient,
		systemClient:     systemClient,
		authClient:       authClient,
		campaignClient:   campaignClient,
		eventClient:      eventClient,
	}
}

// HandleDashboard renders the dashboard page.
func (s service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
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
func (s service) HandleDashboardContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()
	loc, _ := s.base.Localizer(w, r)

	stats := templates.DashboardStats{
		TotalSystems:      "0",
		TotalCampaigns:    "0",
		TotalSessions:     "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
		TotalUsers:        "0",
	}

	var activities []templates.ActivityEvent

	resp, err := s.statisticsClient.GetGameStatistics(ctx, &statev1.GetGameStatisticsRequest{})
	if err == nil && resp != nil && resp.GetStats() != nil {
		stats.TotalCampaigns = strconv.FormatInt(resp.GetStats().GetCampaignCount(), 10)
		stats.TotalSessions = strconv.FormatInt(resp.GetStats().GetSessionCount(), 10)
		stats.TotalCharacters = strconv.FormatInt(resp.GetStats().GetCharacterCount(), 10)
		stats.TotalParticipants = strconv.FormatInt(resp.GetStats().GetParticipantCount(), 10)
	}

	systemsResp, err := s.systemClient.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err == nil && systemsResp != nil {
		stats.TotalSystems = strconv.FormatInt(int64(len(systemsResp.GetSystems())), 10)
	}

	var totalUsers int64
	pageToken := ""
	ok := true
	for {
		resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
			PageSize:  50,
			PageToken: pageToken,
		})
		if err != nil || resp == nil {
			adminerrors.LogError(r, "list users for dashboard: %v", err)
			ok = false
			break
		}
		totalUsers += int64(len(resp.GetUsers()))
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	if ok {
		stats.TotalUsers = strconv.FormatInt(totalUsers, 10)
	}

	activityService := newActivityService(s.campaignClient, s.eventClient)
	for _, record := range activityService.listRecent(ctx) {
		evt := record.event
		activities = append(activities, templates.ActivityEvent{
			CampaignID:   evt.GetCampaignId(),
			CampaignName: record.campaignName,
			EventType:    eventview.FormatEventType(evt.GetType(), loc),
			Timestamp:    eventview.FormatTimestamp(evt.GetTs()),
			Description:  eventview.FormatEventDescription(evt, loc),
		})
	}

	templ.Handler(templates.DashboardContent(stats, activities, loc)).ServeHTTP(w, r)
}
