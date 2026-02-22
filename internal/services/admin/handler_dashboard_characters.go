package admin

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

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
