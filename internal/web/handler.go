package web

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	"github.com/louisbranch/fracturing.space/internal/web/templates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// campaignsRequestTimeout caps the gRPC request time for campaigns.
	campaignsRequestTimeout = 2 * time.Second
	// campaignThemePromptLimit caps the number of characters shown in the table.
	campaignThemePromptLimit = 80
	// sessionListPageSize caps the number of sessions shown in the UI.
	sessionListPageSize = 10
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
)

// GRPCClientProvider supplies gRPC clients for request handling.
type GRPCClientProvider interface {
	CampaignClient() statev1.CampaignServiceClient
	SessionClient() statev1.SessionServiceClient
	CharacterClient() statev1.CharacterServiceClient
	ParticipantClient() statev1.ParticipantServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	EventClient() statev1.EventServiceClient
}

// Handler routes web requests for the UI.
type Handler struct {
	clientProvider GRPCClientProvider
}

// NewHandler builds the HTTP handler for the web server.
func NewHandler(clientProvider GRPCClientProvider) http.Handler {
	handler := &Handler{clientProvider: clientProvider}
	return handler.routes()
}

// routes wires the HTTP routes for the web handler.
func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web/static"))))
	mux.Handle("/", http.HandlerFunc(h.handleDashboard))
	mux.Handle("/dashboard/content", http.HandlerFunc(h.handleDashboardContent))
	mux.Handle("/campaigns", http.HandlerFunc(h.handleCampaignsPage))
	mux.Handle("/campaigns/table", http.HandlerFunc(h.handleCampaignsTable))
	mux.Handle("/campaigns/", http.HandlerFunc(h.handleCampaignRoutes))
	return mux
}

// handleCampaignsTable returns the first page of campaign rows for HTMX.
func (h *Handler) handleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignTable(w, r, nil, "Campaign service unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		log.Printf("list campaigns: %v", err)
		h.renderCampaignTable(w, r, nil, "Campaigns unavailable.")
		return
	}

	campaigns := response.GetCampaigns()
	if len(campaigns) == 0 {
		h.renderCampaignTable(w, r, nil, "No campaigns yet.")
		return
	}

	rows := buildCampaignRows(campaigns)
	h.renderCampaignTable(w, r, rows, "")
}

// handleCampaignsPage renders the campaigns page fragment or full layout.
func (h *Handler) handleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	if isHTMXRequest(r) {
		templ.Handler(templates.CampaignsPage()).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.CampaignsFullPage()).ServeHTTP(w, r)
}

// handleCampaignRoutes dispatches detail and session subroutes.
func (h *Handler) handleCampaignRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		canonical := strings.TrimRight(r.URL.Path, "/")
		if canonical == "" {
			canonical = "/"
		}
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}
	campaignPath := strings.TrimPrefix(r.URL.Path, "/campaigns/")
	parts := splitPathParts(campaignPath)

	// /campaigns/{id}/characters
	if len(parts) == 2 && parts[1] == "characters" {
		h.handleCharactersList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/table
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "table" {
		h.handleCharactersTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/{characterId}
	if len(parts) == 3 && parts[1] == "characters" {
		h.handleCharacterSheet(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/participants
	if len(parts) == 2 && parts[1] == "participants" {
		h.handleParticipantsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/participants/table
	if len(parts) == 3 && parts[1] == "participants" && parts[2] == "table" {
		h.handleParticipantsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "sessions" {
		h.handleSessionsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/table
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "table" {
		h.handleSessionsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}
	if len(parts) == 3 && parts[1] == "sessions" {
		h.handleSessionDetail(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}/events
	if len(parts) == 4 && parts[1] == "sessions" && parts[3] == "events" {
		h.handleSessionEvents(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/events
	if len(parts) == 2 && parts[1] == "events" {
		h.handleEventLog(w, r, parts[0])
		return
	}
	// /campaigns/{id}/events/table (HTMX fragment)
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "table" {
		h.handleEventLogTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		h.handleCampaignDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

// handleCampaignDetail renders the single-campaign detail content.
func (h *Handler) handleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, "Campaign service unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		log.Printf("get campaign: %v", err)
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, "Campaign unavailable.")
		return
	}

	campaign := response.GetCampaign()
	if campaign == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, "Campaign not found.")
		return
	}

	detail := buildCampaignDetail(campaign)
	h.renderCampaignDetail(w, r, detail, "")
}

// handleSessionsList renders the sessions list page.
func (h *Handler) handleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	campaignName := getCampaignName(h, r, campaignID)

	if isHTMXRequest(r) {
		templ.Handler(templates.SessionsListPage(campaignID, campaignName)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.SessionsListFullPage(campaignID, campaignName)).ServeHTTP(w, r)
}

// handleSessionsTable renders the sessions table via HTMX.
func (h *Handler) handleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		h.renderCampaignSessions(w, r, nil, "Session service unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   sessionListPageSize,
	})
	if err != nil {
		log.Printf("list sessions: %v", err)
		h.renderCampaignSessions(w, r, nil, "Sessions unavailable.")
		return
	}

	sessions := response.GetSessions()
	if len(sessions) == 0 {
		h.renderCampaignSessions(w, r, nil, "No sessions yet.")
		return
	}

	rows := buildCampaignSessionRows(sessions)
	h.renderCampaignSessions(w, r, rows, "")
}

// renderCampaignTable renders a campaign table with optional rows and message.
func (h *Handler) renderCampaignTable(w http.ResponseWriter, r *http.Request, rows []templates.CampaignRow, message string) {
	templ.Handler(templates.CampaignsTable(rows, message)).ServeHTTP(w, r)
}

// renderCampaignDetail renders the campaign detail fragment or full layout.
func (h *Handler) renderCampaignDetail(w http.ResponseWriter, r *http.Request, detail templates.CampaignDetail, message string) {
	if isHTMXRequest(r) {
		templ.Handler(templates.CampaignDetailPage(detail, message)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.CampaignDetailFullPage(detail, message)).ServeHTTP(w, r)
}

// renderCampaignSessions renders the session list fragment.
func (h *Handler) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string) {
	templ.Handler(templates.CampaignSessionsList(rows, message)).ServeHTTP(w, r)
}

// campaignClient returns the currently configured campaign client.
func (h *Handler) campaignClient() statev1.CampaignServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CampaignClient()
}

// sessionClient returns the currently configured session client.
func (h *Handler) sessionClient() statev1.SessionServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SessionClient()
}

// characterClient returns the currently configured character client.
func (h *Handler) characterClient() statev1.CharacterServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CharacterClient()
}

// participantClient returns the currently configured participant client.
func (h *Handler) participantClient() statev1.ParticipantServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.ParticipantClient()
}

// snapshotClient returns the currently configured snapshot client.
func (h *Handler) snapshotClient() statev1.SnapshotServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SnapshotClient()
}

// eventClient returns the currently configured event client.
func (h *Handler) eventClient() statev1.EventServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.EventClient()
}

// isHTMXRequest reports whether the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

// splitPathParts returns non-empty path segments.
func splitPathParts(path string) []string {
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return parts
}

// buildCampaignRows formats campaign rows for the table.
func buildCampaignRows(campaigns []*statev1.Campaign) []templates.CampaignRow {
	rows := make([]templates.CampaignRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		rows = append(rows, templates.CampaignRow{
			ID:               campaign.GetId(),
			Name:             campaign.GetName(),
			System:           formatGameSystem(campaign.GetSystem()),
			GMMode:           formatGmMode(campaign.GetGmMode()),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			ThemePrompt:      truncateText(campaign.GetThemePrompt(), campaignThemePromptLimit),
			CreatedDate:      formatCreatedDate(campaign.GetCreatedAt()),
		})
	}
	return rows
}

// buildCampaignDetail formats a campaign into detail view data.
func buildCampaignDetail(campaign *statev1.Campaign) templates.CampaignDetail {
	if campaign == nil {
		return templates.CampaignDetail{}
	}
	return templates.CampaignDetail{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		System:           formatGameSystem(campaign.GetSystem()),
		GMMode:           formatGmMode(campaign.GetGmMode()),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		ThemePrompt:      campaign.GetThemePrompt(),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
	}
}

// buildCampaignSessionRows formats session rows for the detail view.
func buildCampaignSessionRows(sessions []*statev1.Session) []templates.CampaignSessionRow {
	rows := make([]templates.CampaignSessionRow, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		statusBadge := "secondary"
		if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			statusBadge = "success"
		}
		row := templates.CampaignSessionRow{
			ID:          session.GetId(),
			CampaignID:  session.GetCampaignId(),
			Name:        session.GetName(),
			Status:      formatSessionStatus(session.GetStatus()),
			StatusBadge: statusBadge,
			StartedAt:   formatTimestamp(session.GetStartedAt()),
		}
		if session.GetEndedAt() != nil {
			row.EndedAt = formatTimestamp(session.GetEndedAt())
		}
		rows = append(rows, row)
	}
	return rows
}

// formatGmMode returns a display label for a GM mode enum.
func formatGmMode(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "Human"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "Hybrid"
	default:
		return "Unspecified"
	}
}

func formatGameSystem(system commonv1.GameSystem) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "Daggerheart"
	default:
		return "Unspecified"
	}
}

// formatSessionStatus returns a display label for a session status.
func formatSessionStatus(status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return "Active"
	case statev1.SessionStatus_SESSION_ENDED:
		return "Ended"
	default:
		return "Unspecified"
	}
}

// formatCreatedDate returns a YYYY-MM-DD string for a timestamp.
func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

// formatTimestamp returns a YYYY-MM-DD HH:MM:SS string for a timestamp.
func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

// truncateText shortens text to a maximum length with an ellipsis.
func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

// handleDashboard renders the dashboard page.
func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if isHTMXRequest(r) {
		templ.Handler(templates.DashboardPage()).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.DashboardFullPage()).ServeHTTP(w, r)
}

// handleDashboardContent loads and renders the dashboard statistics and recent activity.
func (h *Handler) handleDashboardContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	stats := templates.DashboardStats{
		TotalCampaigns:    "0",
		ActiveSessions:    "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
	}

	var activities []templates.ActivityEvent

	// Aggregate campaign count
	if campaignClient := h.campaignClient(); campaignClient != nil {
		resp, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
		if err == nil && resp != nil {
			stats.TotalCampaigns = strconv.FormatInt(int64(len(resp.GetCampaigns())), 10)

			// Count active sessions and aggregate participants/characters
			var totalChars, totalParts int32
			for _, campaign := range resp.GetCampaigns() {
				if campaign != nil {
					totalChars += campaign.GetCharacterCount()
					totalParts += campaign.GetParticipantCount()
				}
			}
			stats.TotalCharacters = strconv.FormatInt(int64(totalChars), 10)
			stats.TotalParticipants = strconv.FormatInt(int64(totalParts), 10)
		}
	}

	// Count active sessions across all campaigns
	if sessionClient := h.sessionClient(); sessionClient != nil {
		if campaignClient := h.campaignClient(); campaignClient != nil {
			campaignsResp, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
			if err == nil && campaignsResp != nil {
				var activeSessions int32
				for _, campaign := range campaignsResp.GetCampaigns() {
					if campaign == nil {
						continue
					}
					sessionsResp, err := sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
						CampaignId: campaign.GetId(),
					})
					if err == nil && sessionsResp != nil {
						for _, session := range sessionsResp.GetSessions() {
							if session != nil && session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
								activeSessions++
							}
						}
					}
				}
				stats.ActiveSessions = strconv.FormatInt(int64(activeSessions), 10)
			}
		}
	}

	// Fetch recent activity (last 15 events across all campaigns)
	if eventClient := h.eventClient(); eventClient != nil {
		if campaignClient := h.campaignClient(); campaignClient != nil {
			campaignsResp, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
			if err == nil && campaignsResp != nil {
				// Get events from each campaign and merge
				allEvents := make([]struct {
					event        *statev1.Event
					campaignName string
				}, 0)

				for _, campaign := range campaignsResp.GetCampaigns() {
					if campaign == nil {
						continue
					}
					eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
						CampaignId: campaign.GetId(),
						PageSize:   5, // Get top 5 from each campaign
						OrderBy:    "seq desc",
					})
					if err == nil && eventsResp != nil {
						for _, event := range eventsResp.GetEvents() {
							if event != nil {
								allEvents = append(allEvents, struct {
									event        *statev1.Event
									campaignName string
								}{event, campaign.GetName()})
							}
						}
					}
				}

				// Sort by timestamp descending and take top 15
				// Simple bubble sort for small datasets
				for i := 0; i < len(allEvents); i++ {
					for j := i + 1; j < len(allEvents); j++ {
						iTs := allEvents[i].event.GetTs()
						jTs := allEvents[j].event.GetTs()
						if iTs != nil && jTs != nil && iTs.AsTime().Before(jTs.AsTime()) {
							allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
						}
					}
				}

				maxEvents := 15
				if len(allEvents) < maxEvents {
					maxEvents = len(allEvents)
				}

				for i := 0; i < maxEvents; i++ {
					evt := allEvents[i].event
					activities = append(activities, templates.ActivityEvent{
						CampaignID:   evt.GetCampaignId(),
						CampaignName: allEvents[i].campaignName,
						EventType:    formatEventType(evt.GetType()),
						Timestamp:    formatTimestamp(evt.GetTs()),
						Description:  formatEventDescription(evt),
					})
				}
			}
		}
	}

	templ.Handler(templates.DashboardContent(stats, activities)).ServeHTTP(w, r)
}

// formatEventType returns a display label for an event type string.
func formatEventType(eventType string) string {
	switch eventType {
	// Campaign events
	case "campaign.created":
		return "Campaign Created"
	case "campaign.forked":
		return "Campaign Forked"
	case "campaign.status_changed":
		return "Campaign Status Changed"
	case "campaign.updated":
		return "Campaign Updated"
	// Participant events
	case "participant.joined":
		return "Participant Joined"
	case "participant.left":
		return "Participant Left"
	case "participant.updated":
		return "Participant Updated"
	// Character events
	case "character.created":
		return "Character Created"
	case "character.deleted":
		return "Character Deleted"
	case "character.updated":
		return "Character Updated"
	case "character.profile_updated":
		return "Profile Updated"
	case "character.controller_assigned":
		return "Controller Assigned"
	// Snapshot events
	case "chronicle.character_state_changed":
		return "Character State Changed"
	case "chronicle.gm_fear_changed":
		return "GM Fear Changed"
	// Session events
	case "session.started":
		return "Session Started"
	case "session.ended":
		return "Session Ended"
	// Action events
	case "action.roll_resolved":
		return "Roll Resolved"
	case "action.outcome_applied":
		return "Outcome Applied"
	case "action.outcome_rejected":
		return "Outcome Rejected"
	case "action.note_added":
		return "Note Added"
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

// formatEventDescription generates a human-readable event description.
func formatEventDescription(event *statev1.Event) string {
	if event == nil {
		return ""
	}
	return formatEventType(event.GetType())
}

// handleCharactersList renders the characters list page.
func (h *Handler) handleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	campaignName := getCampaignName(h, r, campaignID)

	if isHTMXRequest(r) {
		templ.Handler(templates.CharactersListPage(campaignID, campaignName)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.CharactersListFullPage(campaignID, campaignName)).ServeHTTP(w, r)
}

// handleCharactersTable renders the characters table.
func (h *Handler) handleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	characterClient := h.characterClient()
	if characterClient == nil {
		h.renderCharactersTable(w, r, nil, "Character service unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	// Get characters
	response, err := characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list characters: %v", err)
		h.renderCharactersTable(w, r, nil, "Characters unavailable.")
		return
	}

	characters := response.GetCharacters()
	if len(characters) == 0 {
		h.renderCharactersTable(w, r, nil, "No characters yet.")
		return
	}

	rows := buildCharacterRows(characters)
	h.renderCharactersTable(w, r, rows, "")
}

// handleCharacterSheet renders the character sheet page.
func (h *Handler) handleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	characterClient := h.characterClient()
	if characterClient == nil {
		http.Error(w, "Character service unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	// Get character sheet
	response, err := characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		log.Printf("get character sheet: %v", err)
		http.Error(w, "Character unavailable", http.StatusNotFound)
		return
	}

	character := response.GetCharacter()
	if character == nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	// Get campaign name
	campaignName := getCampaignName(h, r, campaignID)

	// Get recent events for this character
	var recentEvents []templates.EventRow
	if eventClient := h.eventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   20,
			OrderBy:    "seq desc",
			Filter:     "entity_id = \"" + characterID + "\"",
		})
		if err == nil && eventsResp != nil {
			for _, event := range eventsResp.GetEvents() {
				if event != nil {
					recentEvents = append(recentEvents, templates.EventRow{
						Seq:         event.GetSeq(),
						Type:        formatEventType(event.GetType()),
						Timestamp:   formatTimestamp(event.GetTs()),
						Description: formatEventDescription(event),
						PayloadJSON: string(event.GetPayloadJson()),
					})
				}
			}
		}
	}

	sheet := buildCharacterSheet(campaignID, campaignName, character, recentEvents)

	if isHTMXRequest(r) {
		templ.Handler(templates.CharacterSheetPage(sheet)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.CharacterSheetFullPage(sheet)).ServeHTTP(w, r)
}

// renderCharactersTable renders the characters table component.
func (h *Handler) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string) {
	templ.Handler(templates.CharactersTable(rows, message)).ServeHTTP(w, r)
}

// buildCharacterRows formats character rows for the table.
func buildCharacterRows(characters []*statev1.Character) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		// Format controller
		controller := "Unknown"
		// TODO: Get controller information (requires join with participant data)

		rows = append(rows, templates.CharacterRow{
			ID:         character.GetId(),
			CampaignID: character.GetCampaignId(),
			Name:       character.GetName(),
			Kind:       formatCharacterKind(character.GetKind()),
			Controller: controller,
		})
	}
	return rows
}

// buildCharacterSheet formats character sheet data.
func buildCharacterSheet(campaignID, campaignName string, character *statev1.Character, recentEvents []templates.EventRow) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   "Unknown",
		CreatedAt:    formatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    formatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
}

// formatCharacterKind returns a display label for a character kind.
func formatCharacterKind(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "Unspecified"
	}
}

// getCampaignName fetches the campaign name by ID.
func getCampaignName(h *Handler, r *http.Request, campaignID string) string {
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		return "Campaign"
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return "Campaign"
	}

	return response.GetCampaign().GetName()
}

// handleParticipantsList renders the participants list page.
func (h *Handler) handleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	campaignName := getCampaignName(h, r, campaignID)

	if isHTMXRequest(r) {
		templ.Handler(templates.ParticipantsListPage(campaignID, campaignName)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.ParticipantsListFullPage(campaignID, campaignName)).ServeHTTP(w, r)
}

// handleParticipantsTable renders the participants table.
func (h *Handler) handleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	participantClient := h.participantClient()
	if participantClient == nil {
		h.renderParticipantsTable(w, r, nil, "Participant service unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list participants: %v", err)
		h.renderParticipantsTable(w, r, nil, "Participants unavailable.")
		return
	}

	participants := response.GetParticipants()
	if len(participants) == 0 {
		h.renderParticipantsTable(w, r, nil, "No participants yet.")
		return
	}

	rows := buildParticipantRows(participants)
	h.renderParticipantsTable(w, r, rows, "")
}

// renderParticipantsTable renders the participants table component.
func (h *Handler) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string) {
	templ.Handler(templates.ParticipantsTable(rows, message)).ServeHTTP(w, r)
}

// buildParticipantRows formats participant rows for the table.
func buildParticipantRows(participants []*statev1.Participant) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole())
		controller, controllerVariant := formatParticipantController(participant.GetController())

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			DisplayName:       participant.GetDisplayName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

// formatParticipantRole returns a display label and variant for a participant role.
func formatParticipantRole(role statev1.ParticipantRole) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM", "info"
	case statev1.ParticipantRole_PLAYER:
		return "Player", "success"
	default:
		return "Unspecified", "secondary"
	}
}

// formatParticipantController returns a display label and variant for a controller type.
func formatParticipantController(controller statev1.Controller) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "Human", "success"
	case statev1.Controller_CONTROLLER_AI:
		return "AI", "info"
	default:
		return "Unspecified", "secondary"
	}
}

// handleSessionDetail renders the session detail page.
func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		http.Error(w, "Session service unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	// Get session details
	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		log.Printf("get session: %v", err)
		http.Error(w, "Session unavailable", http.StatusNotFound)
		return
	}

	session := response.GetSession()
	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	campaignName := getCampaignName(h, r, campaignID)

	// Get event count for this session
	var eventCount int32
	if eventClient := h.eventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   1,
			Filter:     "session_id = \"" + sessionID + "\"",
		})
		if err == nil && eventsResp != nil {
			eventCount = eventsResp.GetTotalSize()
		}
	}

	detail := buildSessionDetail(campaignID, campaignName, session, eventCount)

	if isHTMXRequest(r) {
		templ.Handler(templates.SessionDetailPage(detail)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.SessionDetailFullPage(detail)).ServeHTTP(w, r)
}

// handleSessionEvents renders the session events via HTMX.
func (h *Handler) handleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState("Event service unavailable")).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	pageToken := r.URL.Query().Get("page_token")

	// Get events for this session
	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\"",
	})
	if err != nil {
		log.Printf("list session events: %v", err)
		templ.Handler(templates.EmptyState("Events unavailable")).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID)
	sessionName := getSessionName(h, r, campaignID, sessionID)

	events := buildEventRows(eventsResp.GetEvents())
	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           sessionID,
		Name:         sessionName,
		Events:       events,
		EventCount:   eventsResp.GetTotalSize(),
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
	}

	templ.Handler(templates.SessionEventsContent(detail)).ServeHTTP(w, r)
}

// handleEventLog renders the event log page.
func (h *Handler) handleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	campaignName := getCampaignName(h, r, campaignID)
	filters := parseEventFilters(r)

	// Fetch events for initial load
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)

		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err == nil && eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents())
			totalCount = eventsResp.GetTotalSize()
			nextToken = eventsResp.GetNextPageToken()
			prevToken = eventsResp.GetPreviousPageToken()
		}
	}

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		TotalCount:   totalCount,
		NextToken:    nextToken,
		PrevToken:    prevToken,
	}

	if isHTMXRequest(r) {
		templ.Handler(templates.EventLogPage(view)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.EventLogFullPage(view)).ServeHTTP(w, r)
}

// handleEventLogTable renders the event log table via HTMX.
func (h *Handler) handleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState("Event service unavailable")).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	filters := parseEventFilters(r)
	filterExpr := buildEventFilterExpression(filters)
	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		log.Printf("list events: %v", err)
		templ.Handler(templates.EmptyState("Events unavailable")).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID)
	events := buildEventRows(eventsResp.GetEvents())

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
		TotalCount:   eventsResp.GetTotalSize(),
	}

	templ.Handler(templates.EventLogTableContent(view)).ServeHTTP(w, r)
}

// buildSessionDetail formats a session into detail view data.
func buildSessionDetail(campaignID, campaignName string, session *statev1.Session, eventCount int32) templates.SessionDetail {
	if session == nil {
		return templates.SessionDetail{}
	}

	status := formatSessionStatus(session.GetStatus())
	statusBadge := "secondary"
	if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
		statusBadge = "success"
	}

	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           session.GetId(),
		Name:         session.GetName(),
		Status:       status,
		StatusBadge:  statusBadge,
		StartedAt:    formatTimestamp(session.GetStartedAt()),
		EventCount:   eventCount,
	}

	if session.GetEndedAt() != nil {
		detail.EndedAt = formatTimestamp(session.GetEndedAt())
	}

	return detail
}

// buildEventRows formats events for display.
func buildEventRows(events []*statev1.Event) []templates.EventRow {
	rows := make([]templates.EventRow, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		rows = append(rows, templates.EventRow{
			CampaignID:  event.GetCampaignId(),
			Seq:         event.GetSeq(),
			Hash:        event.GetHash(),
			Type:        event.GetType(),
			TypeDisplay: formatEventType(event.GetType()),
			Timestamp:   formatTimestamp(event.GetTs()),
			SessionID:   event.GetSessionId(),
			ActorType:   event.GetActorType(),
			ActorName:   event.GetActorId(),
			EntityType:  event.GetEntityType(),
			EntityID:    event.GetEntityId(),
			EntityName:  event.GetEntityId(),
			Description: formatEventDescription(event),
			PayloadJSON: string(event.GetPayloadJson()),
		})
	}
	return rows
}

// parseEventFilters extracts filter parameters from the request.
func parseEventFilters(r *http.Request) templates.EventFilterOptions {
	return templates.EventFilterOptions{
		SessionID:  r.URL.Query().Get("session_id"),
		EventType:  r.URL.Query().Get("event_type"),
		ActorType:  r.URL.Query().Get("actor_type"),
		EntityType: r.URL.Query().Get("entity_type"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
}

// escapeAIP160StringLiteral escapes special characters for AIP-160 string literals.
// Backslashes and double quotes must be escaped to prevent injection.
func escapeAIP160StringLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// buildEventFilterExpression creates an AIP-160 filter expression from options.
func buildEventFilterExpression(filters templates.EventFilterOptions) string {
	var parts []string

	if filters.SessionID != "" {
		parts = append(parts, "session_id = \""+escapeAIP160StringLiteral(filters.SessionID)+"\"")
	}
	if filters.EventType != "" {
		parts = append(parts, "type = \""+escapeAIP160StringLiteral(filters.EventType)+"\"")
	}
	if filters.ActorType != "" {
		parts = append(parts, "actor_type = \""+escapeAIP160StringLiteral(filters.ActorType)+"\"")
	}
	if filters.EntityType != "" {
		parts = append(parts, "entity_type = \""+escapeAIP160StringLiteral(filters.EntityType)+"\"")
	}
	if filters.StartDate != "" {
		parts = append(parts, "ts >= timestamp(\""+escapeAIP160StringLiteral(filters.StartDate)+"T00:00:00Z\")")
	}
	if filters.EndDate != "" {
		parts = append(parts, "ts <= timestamp(\""+escapeAIP160StringLiteral(filters.EndDate)+"T23:59:59Z\")")
	}

	return strings.Join(parts, " AND ")
}

// getSessionName fetches the session name by ID.
func getSessionName(h *Handler, r *http.Request, campaignID, sessionID string) string {
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		return "Session"
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil || response == nil || response.GetSession() == nil {
		return "Session"
	}

	return response.GetSession().GetName()
}
