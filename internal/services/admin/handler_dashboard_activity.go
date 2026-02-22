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
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
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
						EventType:    formatEventType(evt.GetType(), loc),
						Timestamp:    formatTimestamp(evt.GetTs()),
						Description:  formatEventDescription(evt, loc),
					})
				}
			}
		}
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
func (h *Handler) handleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.CharactersListPage(campaignID, campaignName, loc),
		templates.CharactersListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.characters", templates.AppName()),
	)
}

// handleCharactersTable renders the characters table.
func (h *Handler) handleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	characterClient := h.characterClient()
	if characterClient == nil {
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.character_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	// Get characters
	response, err := characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list characters: %v", err)
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.characters_unavailable"), loc)
		return
	}

	characters := response.GetCharacters()
	if len(characters) == 0 {
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.no_characters"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := h.participantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for character table: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetName()
				}
			}
		}
	}

	rows := buildCharacterRows(characters, participantNames, loc)
	h.renderCharactersTable(w, r, rows, "", loc)
}

// handleCharacterSheet renders the character sheet page.
func (h *Handler) handleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.renderCharacterSheet(w, r, campaignID, characterID, "info")
}

// handleCharacterActivity renders the character activity tab.
func (h *Handler) handleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.renderCharacterSheet(w, r, campaignID, characterID, "activity")
}

func (h *Handler) renderCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string, activePage string) {
	loc, lang := h.localizer(w, r)
	characterClient := h.characterClient()
	if characterClient == nil {
		http.Error(w, loc.Sprintf("error.character_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		log.Printf("get character sheet: %v", err)
		http.Error(w, loc.Sprintf("error.character_unavailable"), http.StatusNotFound)
		return
	}

	character := response.GetCharacter()
	if character == nil {
		http.Error(w, loc.Sprintf("error.character_not_found"), http.StatusNotFound)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)

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
						Type:        formatEventType(event.GetType(), loc),
						Timestamp:   formatTimestamp(event.GetTs()),
						Description: formatEventDescription(event, loc),
						PayloadJSON: string(event.GetPayloadJson()),
					})
				}
			}
		}
	}

	controller := loc.Sprintf("label.unassigned")
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID != "" {
		if participantClient := h.participantClient(); participantClient != nil {
			participantResp, err := participantClient.GetParticipant(ctx, &statev1.GetParticipantRequest{
				CampaignId:    campaignID,
				ParticipantId: participantID,
			})
			if err != nil {
				log.Printf("get participant for character sheet: %v", err)
				controller = loc.Sprintf("label.unknown")
			} else if participant := participantResp.GetParticipant(); participant != nil {
				controller = participant.GetName()
			} else {
				controller = loc.Sprintf("label.unknown")
			}
		} else {
			controller = loc.Sprintf("label.unknown")
		}
	}

	sheet := buildCharacterSheet(campaignID, campaignName, character, recentEvents, controller, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.CharacterSheetPage(sheet, activePage, loc),
		templates.CharacterSheetFullPage(sheet, activePage, pageCtx),
		htmxLocalizedPageTitle(loc, "title.character_sheet", sheet.Character.GetName(), templates.AppName()),
	)
}

// renderCharactersTable renders the characters table component.
func (h *Handler) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string, loc *message.Printer) {
	templ.Handler(templates.CharactersTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildCharacterRows formats character rows for the table.
func buildCharacterRows(characters []*statev1.Character, participantNames map[string]string, loc *message.Printer) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		controller := formatCharacterController(character, participantNames, loc)

		rows = append(rows, templates.CharacterRow{
			ID:         character.GetId(),
			CampaignID: character.GetCampaignId(),
			Name:       character.GetName(),
			Kind:       formatCharacterKind(character.GetKind(), loc),
			Controller: controller,
		})
	}
	return rows
}

// buildCharacterSheet formats character sheet data.
func buildCharacterSheet(campaignID, campaignName string, character *statev1.Character, recentEvents []templates.EventRow, controller string, loc *message.Printer) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   controller,
		CreatedAt:    formatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    formatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
}

func formatCharacterController(character *statev1.Character, participantNames map[string]string, loc *message.Printer) string {
	if character == nil {
		return loc.Sprintf("label.unassigned")
	}
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID == "" {
		return loc.Sprintf("label.unassigned")
	}
	if name, ok := participantNames[participantID]; ok {
		return name
	}
	return loc.Sprintf("label.unknown")
}

// formatCharacterKind returns a display label for a character kind.
func formatCharacterKind(kind statev1.CharacterKind, loc *message.Printer) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return loc.Sprintf("label.pc")
	case statev1.CharacterKind_NPC:
		return loc.Sprintf("label.npc")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

// getCampaignName fetches the campaign name by ID.
func getCampaignName(h *Handler, r *http.Request, campaignID string, loc *message.Printer) string {
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		return loc.Sprintf("label.campaign")
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}

	return response.GetCampaign().GetName()
}

// handleParticipantsList renders the participants list page.
func (h *Handler) handleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.ParticipantsListPage(campaignID, campaignName, loc),
		templates.ParticipantsListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.participants", templates.AppName()),
	)
}

// handleParticipantsTable renders the participants table.
func (h *Handler) handleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	participantClient := h.participantClient()
	if participantClient == nil {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participant_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list participants: %v", err)
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participants_unavailable"), loc)
		return
	}

	participants := response.GetParticipants()
	if len(participants) == 0 {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.no_participants"), loc)
		return
	}

	rows := buildParticipantRows(participants, loc)
	h.renderParticipantsTable(w, r, rows, "", loc)
}

// handleInvitesList renders the invites list page.
func (h *Handler) handleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.InvitesListPage(campaignID, campaignName, loc),
		templates.InvitesListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.invites", templates.AppName()),
	)
}

// handleInvitesTable renders the invites table.
func (h *Handler) handleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	inviteClient := h.inviteClient()
	if inviteClient == nil {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invite_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   inviteListPageSize,
	})
	if err != nil {
		log.Printf("list invites: %v", err)
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invites_unavailable"), loc)
		return
	}

	invites := response.GetInvites()
	if len(invites) == 0 {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.no_invites"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := h.participantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for invites: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetName()
				}
			}
		}
	}

	recipientNames := map[string]string{}
	if authClient := h.authClient(); authClient != nil {
		for _, inv := range invites {
			if inv == nil {
				continue
			}
			recipientID := strings.TrimSpace(inv.GetRecipientUserId())
			if recipientID == "" {
				continue
			}
			if _, ok := recipientNames[recipientID]; ok {
				continue
			}
			userResp, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientID})
			if err != nil {
				log.Printf("get invite recipient: %v", err)
				recipientNames[recipientID] = ""
				continue
			}
			if user := userResp.GetUser(); user != nil {
				recipientNames[recipientID] = user.GetEmail()
			}
		}
	}

	rows := buildInviteRows(invites, participantNames, recipientNames, loc)
	h.renderInvitesTable(w, r, rows, "", loc)
}

// renderInvitesTable renders the invites table component.
func (h *Handler) renderInvitesTable(w http.ResponseWriter, r *http.Request, rows []templates.InviteRow, message string, loc *message.Printer) {
	templ.Handler(templates.InvitesTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildInviteRows formats invite rows for the table.
func buildInviteRows(invites []*statev1.Invite, participantNames map[string]string, recipientNames map[string]string, loc *message.Printer) []templates.InviteRow {
	rows := make([]templates.InviteRow, 0, len(invites))
	for _, inv := range invites {
		if inv == nil {
			continue
		}

		participantLabel := participantNames[inv.GetParticipantId()]
		if participantLabel == "" {
			participantLabel = loc.Sprintf("label.unknown")
		}

		recipientLabel := loc.Sprintf("label.unassigned")
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID != "" {
			recipientLabel = recipientNames[recipientID]
			if recipientLabel == "" {
				recipientLabel = recipientID
			}
		}

		statusLabel, statusVariant := formatInviteStatus(inv.GetStatus(), loc)

		rows = append(rows, templates.InviteRow{
			ID:            inv.GetId(),
			CampaignID:    inv.GetCampaignId(),
			Participant:   participantLabel,
			Recipient:     recipientLabel,
			Status:        statusLabel,
			StatusVariant: statusVariant,
			CreatedAt:     formatTimestamp(inv.GetCreatedAt()),
			UpdatedAt:     formatTimestamp(inv.GetUpdatedAt()),
		})
	}
	return rows
}

// renderParticipantsTable renders the participants table component.
func (h *Handler) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string, loc *message.Printer) {
	templ.Handler(templates.ParticipantsTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildParticipantRows formats participant rows for the table.
func buildParticipantRows(participants []*statev1.Participant, loc *message.Printer) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole(), loc)
		access, accessVariant := formatParticipantAccess(participant.GetCampaignAccess(), loc)
		controller, controllerVariant := formatParticipantController(participant.GetController(), loc)

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			Name:              participant.GetName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Access:            access,
			AccessVariant:     accessVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

// formatParticipantRole returns a display label and variant for a participant role.
func formatParticipantRole(role statev1.ParticipantRole, loc *message.Printer) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return loc.Sprintf("label.gm"), "info"
	case statev1.ParticipantRole_PLAYER:
		return loc.Sprintf("label.player"), "success"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantController returns a display label and variant for a controller type.
func formatParticipantController(controller statev1.Controller, loc *message.Printer) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return loc.Sprintf("label.human"), "success"
	case statev1.Controller_CONTROLLER_AI:
		return loc.Sprintf("label.ai"), "info"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantAccess returns a display label and variant for campaign access.
func formatParticipantAccess(access statev1.CampaignAccess, loc *message.Printer) (string, string) {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return loc.Sprintf("label.member"), "secondary"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return loc.Sprintf("label.manager"), "info"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return loc.Sprintf("label.owner"), "warning"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// handleSessionDetail renders the session detail page.
func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, lang := h.localizer(w, r)
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		http.Error(w, loc.Sprintf("error.session_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	// Get session details
	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		log.Printf("get session: %v", err)
		http.Error(w, loc.Sprintf("error.session_unavailable"), http.StatusNotFound)
		return
	}

	session := response.GetSession()
	if session == nil {
		http.Error(w, loc.Sprintf("error.session_not_found"), http.StatusNotFound)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)

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

	detail := buildSessionDetail(campaignID, campaignName, session, eventCount, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.SessionDetailPage(detail, loc),
		templates.SessionDetailFullPage(detail, pageCtx),
		htmxLocalizedPageTitle(loc, "title.session", detail.Name, templates.AppName()),
	)
}

// handleSessionEvents renders the session events via HTMX.
func (h *Handler) handleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
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
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)
	sessionName := getSessionName(h, r, campaignID, sessionID, loc)

	events := buildEventRows(eventsResp.GetEvents(), loc)
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

	templ.Handler(templates.SessionEventsContent(detail, loc)).ServeHTTP(w, r)
}

// handleEventLog renders the event log page.
func (h *Handler) handleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	filters := parseEventFilters(r)

	// Fetch events for initial load
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := h.gameGRPCCallContext(r.Context())
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)

		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err == nil && eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents(), loc)
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
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.EventLogPage(view, loc),
		templates.EventLogFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.events", templates.AppName()),
	)
}

// handleEventLogTable renders the event log table via HTMX.
func (h *Handler) handleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
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
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)
	events := buildEventRows(eventsResp.GetEvents(), loc)

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
		TotalCount:   eventsResp.GetTotalSize(),
	}

	if pushURL := eventFilterPushURL("/campaigns/"+campaignID+"/events", filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.EventLogTableContent(view, loc)).ServeHTTP(w, r)
}

// buildSessionDetail formats a session into detail view data.
func buildSessionDetail(campaignID, campaignName string, session *statev1.Session, eventCount int32, loc *message.Printer) templates.SessionDetail {
	if session == nil {
		return templates.SessionDetail{}
	}

	status := formatSessionStatus(session.GetStatus(), loc)
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
func buildEventRows(events []*statev1.Event, loc *message.Printer) []templates.EventRow {
	rows := make([]templates.EventRow, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		rows = append(rows, templates.EventRow{
			CampaignID:       event.GetCampaignId(),
			Seq:              event.GetSeq(),
			Hash:             event.GetHash(),
			Type:             event.GetType(),
			TypeDisplay:      formatEventType(event.GetType(), loc),
			Timestamp:        formatTimestamp(event.GetTs()),
			SessionID:        event.GetSessionId(),
			ActorType:        event.GetActorType(),
			ActorTypeDisplay: formatActorType(event.GetActorType(), loc),
			ActorName:        "",
			EntityType:       event.GetEntityType(),
			EntityID:         event.GetEntityId(),
			EntityName:       event.GetEntityId(),
			Description:      formatEventDescription(event, loc),
			PayloadJSON:      string(event.GetPayloadJson()),
		})
	}
	return rows
}

func buildScenarioTimelineEntries(entries []*statev1.TimelineEntry, loc *message.Printer) []templates.ScenarioTimelineEntry {
	rows := make([]templates.ScenarioTimelineEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		projection := entry.GetProjection()
		title := strings.TrimSpace(projection.GetTitle())
		eventTypeDisplay := formatEventType(entry.GetEventType(), loc)
		if title == "" {
			title = eventTypeDisplay
		}
		subtitle := strings.TrimSpace(projection.GetSubtitle())
		status := strings.TrimSpace(projection.GetStatus())
		iconID := entry.GetIconId()
		if iconID == commonv1.IconId_ICON_ID_UNSPECIFIED {
			iconID = commonv1.IconId_ICON_ID_GENERIC
		}
		rows = append(rows, templates.ScenarioTimelineEntry{
			Seq:              entry.GetSeq(),
			EventType:        entry.GetEventType(),
			EventTypeDisplay: eventTypeDisplay,
			EventTime:        formatTimestamp(entry.GetEventTime()),
			IconID:           iconID,
			Title:            title,
			Subtitle:         subtitle,
			Status:           status,
			StatusBadge:      timelineStatusBadgeVariant(status),
			Fields:           buildScenarioTimelineFields(projection.GetFields()),
			PayloadJSON:      strings.TrimSpace(entry.GetEventPayloadJson()),
		})
	}
	return rows
}

func buildScenarioTimelineFields(fields []*statev1.ProjectionField) []templates.ScenarioTimelineField {
	if len(fields) == 0 {
		return nil
	}
	result := make([]templates.ScenarioTimelineField, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		label := strings.TrimSpace(field.GetLabel())
		value := strings.TrimSpace(field.GetValue())
		if label == "" && value == "" {
			continue
		}
		result = append(result, templates.ScenarioTimelineField{
			Label: label,
			Value: value,
		})
	}
	return result
}

func timelineStatusBadgeVariant(status string) string {
	if status == "" {
		return "secondary"
	}
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "ACTIVE":
		return "success"
	case "DRAFT":
		return "warning"
	case "COMPLETED":
		return "success"
	case "ARCHIVED", "ENDED":
		return "neutral"
	default:
		return "secondary"
	}
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

func eventFilterPushURL(basePath string, filters templates.EventFilterOptions, pageToken string) string {
	pushURL := templates.EventFilterBaseURL(basePath, filters)
	if pageToken != "" {
		return templates.AppendQueryParam(pushURL, "page_token", pageToken)
	}
	return pushURL
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
func getSessionName(h *Handler, r *http.Request, campaignID, sessionID string, loc *message.Printer) string {
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		return loc.Sprintf("label.session")
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil || response == nil || response.GetSession() == nil {
		return loc.Sprintf("label.session")
	}

	return response.GetSession().GetName()
}
