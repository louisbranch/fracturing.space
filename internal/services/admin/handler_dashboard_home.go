package admin

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != routepath.Root {
		http.NotFound(w, r)
		return
	}
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.DashboardPage(loc),
		templates.DashboardFullPage(pageCtx),
		htmxLocalizedPageTitle(loc, "title.dashboard", templates.AppName()),
	)
}

// handleDashboardContent loads and renders the dashboard statistics and recent activity.
func (h *Handler) handleDashboardContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()
	loc, _ := h.localizer(w, r)

	stats := templates.DashboardStats{
		TotalSystems:      "0",
		TotalCampaigns:    "0",
		TotalSessions:     "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
		TotalUsers:        "0",
	}

	var activities []templates.ActivityEvent

	if statisticsClient := h.statisticsClient(); statisticsClient != nil {
		resp, err := statisticsClient.GetGameStatistics(ctx, &statev1.GetGameStatisticsRequest{})
		if err == nil && resp != nil && resp.GetStats() != nil {
			stats.TotalCampaigns = strconv.FormatInt(resp.GetStats().GetCampaignCount(), 10)
			stats.TotalSessions = strconv.FormatInt(resp.GetStats().GetSessionCount(), 10)
			stats.TotalCharacters = strconv.FormatInt(resp.GetStats().GetCharacterCount(), 10)
			stats.TotalParticipants = strconv.FormatInt(resp.GetStats().GetParticipantCount(), 10)
		}
	}

	if systemClient := h.systemClient(); systemClient != nil {
		systemsResp, err := systemClient.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
		if err == nil && systemsResp != nil {
			stats.TotalSystems = strconv.FormatInt(int64(len(systemsResp.GetSystems())), 10)
		}
	}

	if authClient := h.authClient(); authClient != nil {
		var totalUsers int64
		pageToken := ""
		ok := true
		for {
			resp, err := authClient.ListUsers(ctx, &authv1.ListUsersRequest{
				PageSize:  50,
				PageToken: pageToken,
			})
			if err != nil || resp == nil {
				log.Printf("list users for dashboard: %v", err)
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
	}

	// Fetch recent activity (last 15 events across all campaigns)
	activityService := newDashboardActivityService(h.campaignClient(), h.eventClient())
	for _, record := range activityService.listRecent(ctx) {
		evt := record.event
		activities = append(activities, templates.ActivityEvent{
			CampaignID:   evt.GetCampaignId(),
			CampaignName: record.campaignName,
			EventType:    formatEventType(evt.GetType(), loc),
			Timestamp:    formatTimestamp(evt.GetTs()),
			Description:  formatEventDescription(evt, loc),
		})
	}

	templ.Handler(templates.DashboardContent(stats, activities, loc)).ServeHTTP(w, r)
}

// formatEventType returns a display label for an event type string.
func formatEventType(eventType string, loc *message.Printer) string {
	switch eventType {
	// Campaign events
	case "campaign.created":
		return loc.Sprintf("event.campaign_created")
	case "campaign.forked":
		return loc.Sprintf("event.campaign_forked")
	case "campaign.updated":
		return loc.Sprintf("event.campaign_updated")
	// Participant events
	case "participant.joined":
		return loc.Sprintf("event.participant_joined")
	case "participant.left":
		return loc.Sprintf("event.participant_left")
	case "participant.updated":
		return loc.Sprintf("event.participant_updated")
	// Character events
	case "character.created":
		return loc.Sprintf("event.character_created")
	case "character.deleted":
		return loc.Sprintf("event.character_deleted")
	case "character.updated":
		return loc.Sprintf("event.character_updated")
	case "character.profile_updated":
		return loc.Sprintf("event.character_profile_updated")
	// Session events
	case "session.started":
		return loc.Sprintf("event.session_started")
	case "session.ended":
		return loc.Sprintf("event.session_ended")
	case "session.gate_opened":
		return loc.Sprintf("event.session_gate_opened")
	case "session.gate_resolved":
		return loc.Sprintf("event.session_gate_resolved")
	case "session.gate_abandoned":
		return loc.Sprintf("event.session_gate_abandoned")
	case "session.spotlight_set":
		return loc.Sprintf("event.session_spotlight_set")
	case "session.spotlight_cleared":
		return loc.Sprintf("event.session_spotlight_cleared")
	// Invite events
	case "invite.created":
		return loc.Sprintf("event.invite_created")
	case "invite.updated":
		return loc.Sprintf("event.invite_updated")
	// Action events
	case "action.roll_resolved":
		return loc.Sprintf("event.action_roll_resolved")
	case "action.outcome_applied":
		return loc.Sprintf("event.action_outcome_applied")
	case "action.outcome_rejected":
		return loc.Sprintf("event.action_outcome_rejected")
	case "action.note_added":
		return loc.Sprintf("event.action_note_added")
	case "action.character_state_patched":
		return loc.Sprintf("event.action_character_state_patched")
	case "action.gm_fear_changed":
		return loc.Sprintf("event.action_gm_fear_changed")
	case "action.death_move_resolved":
		return loc.Sprintf("event.action_death_move_resolved")
	case "action.blaze_of_glory_resolved":
		return loc.Sprintf("event.action_blaze_of_glory_resolved")
	case "action.attack_resolved":
		return loc.Sprintf("event.action_attack_resolved")
	case "action.reaction_resolved":
		return loc.Sprintf("event.action_reaction_resolved")
	case "action.damage_roll_resolved":
		return loc.Sprintf("event.action_damage_roll_resolved")
	case "action.adversary_action_resolved":
		return loc.Sprintf("event.action_adversary_action_resolved")
	default:
		// Fallback: capitalize and format unknown types
		parts := strings.Split(eventType, ".")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if len(last) > 0 {
				formatted := strings.ReplaceAll(last, "_", " ")
				return strings.ToUpper(formatted[:1]) + formatted[1:]
			}
		}
		return eventType
	}
}

// formatActorType returns a display label for an actor type string.
func formatActorType(actorType string, loc *message.Printer) string {
	if actorType == "" {
		return ""
	}
	switch actorType {
	case "system":
		return loc.Sprintf("filter.actor.system")
	case "participant":
		return loc.Sprintf("filter.actor.participant")
	case "gm":
		return loc.Sprintf("filter.actor.gm")
	default:
		return actorType
	}
}

// formatEventDescription generates a human-readable event description.
func formatEventDescription(event *statev1.Event, loc *message.Printer) string {
	if event == nil {
		return ""
	}
	return formatEventType(event.GetType(), loc)
}

func localeFromTag(tag string) commonv1.Locale {
	if locale, ok := platformi18n.ParseLocale(tag); ok {
		return locale
	}
	return platformi18n.DefaultLocale()
}

// handleCharactersList renders the characters list page.
