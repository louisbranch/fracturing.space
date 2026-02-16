package web

import (
	"html"
	"io"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (h *handler) handleAppCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Characters unavailable", "character service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	canManageCharacters := false
	if participant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken); err == nil && participant != nil && strings.TrimSpace(participant.GetId()) != "" {
		canManageCharacters = canManageCampaignCharacters(participant.GetCampaignAccess())
	}
	controlParticipants := []*statev1.Participant(nil)
	if canManageCharacters {
		resp, err := h.participantClient.ListParticipants(r.Context(), &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, http.StatusBadGateway, "Characters unavailable", "failed to list participants")
			return
		}
		controlParticipants = resp.GetParticipants()
	}

	resp, err := h.characterClient.ListCharacters(r.Context(), &statev1.ListCharactersRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Characters unavailable", "failed to list characters")
		return
	}

	renderAppCampaignCharactersPage(w, campaignID, resp.GetCharacters(), canManageCharacters, controlParticipants)
}

func (h *handler) handleAppCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character unavailable", "character service client is not configured")
		return
	}

	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character unavailable", "character id is required")
		return
	}

	resp, err := h.characterClient.GetCharacterSheet(r.Context(), &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character unavailable", "failed to load character")
		return
	}
	if resp.GetCharacter() == nil {
		h.renderErrorPage(w, r, http.StatusNotFound, "Character unavailable", "character not found")
		return
	}

	renderAppCampaignCharacterDetailPage(w, campaignID, resp.GetCharacter())
}

func (h *handler) handleAppCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character create form")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character name is required")
		return
	}
	kind, ok := parseCharacterKindFormValue(r.FormValue("kind"))
	if !ok {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character kind value is invalid")
		return
	}

	actingParticipant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to resolve campaign participant")
		return
	}
	actingParticipantID := ""
	if actingParticipant != nil {
		actingParticipantID = strings.TrimSpace(actingParticipant.GetId())
	}
	if actingParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant identity required for character action")
		return
	}
	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), actingParticipantID)
	_, err = h.characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       kind,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to create character")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/characters", http.StatusFound)
}

func (h *handler) handleAppCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character update form")
		return
	}
	characterID := strings.TrimSpace(r.FormValue("character_id"))
	if characterID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character id is required")
		return
	}
	req := &statev1.UpdateCharacterRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	}
	hasFieldUpdate := false
	if name := strings.TrimSpace(r.FormValue("name")); name != "" {
		req.Name = wrapperspb.String(name)
		hasFieldUpdate = true
	}
	if rawKind := strings.TrimSpace(r.FormValue("kind")); rawKind != "" {
		kind, ok := parseCharacterKindFormValue(rawKind)
		if !ok {
			h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character kind value is invalid")
			return
		}
		req.Kind = kind
		hasFieldUpdate = true
	}
	if !hasFieldUpdate {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "at least one character field is required")
		return
	}

	actingParticipant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to resolve campaign participant")
		return
	}
	actingParticipantID := ""
	if actingParticipant != nil {
		actingParticipantID = strings.TrimSpace(actingParticipant.GetId())
	}
	if actingParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant identity required for character action")
		return
	}
	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), actingParticipantID)
	_, err = h.characterClient.UpdateCharacter(ctx, req)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to update character")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/characters", http.StatusFound)
}

func (h *handler) handleAppCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character controller form")
		return
	}
	characterID := strings.TrimSpace(r.FormValue("character_id"))
	if characterID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character id is required")
		return
	}
	targetParticipantID := strings.TrimSpace(r.FormValue("participant_id"))

	actingParticipant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to resolve campaign participant")
		return
	}
	actingParticipantID := ""
	if actingParticipant != nil {
		actingParticipantID = strings.TrimSpace(actingParticipant.GetId())
	}
	if actingParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant identity required for character action")
		return
	}
	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), actingParticipantID)
	_, err = h.characterClient.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(targetParticipantID),
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to set character controller")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/characters", http.StatusFound)
}

func canManageCampaignCharacters(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func parseCharacterKindFormValue(raw string) (statev1.CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pc", "character_kind_pc":
		return statev1.CharacterKind_PC, true
	case "npc", "character_kind_npc":
		return statev1.CharacterKind_NPC, true
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, false
	}
}

func characterKindFormValue(kind statev1.CharacterKind) string {
	if kind == statev1.CharacterKind_NPC {
		return "npc"
	}
	return "pc"
}

func participantControlFormLabel(participant *statev1.Participant) string {
	if participant == nil {
		return ""
	}
	label := strings.TrimSpace(participant.GetDisplayName())
	if label == "" {
		label = strings.TrimSpace(participant.GetUserId())
	}
	if label == "" {
		label = strings.TrimSpace(participant.GetId())
	}
	return label
}

func renderAppCampaignCharactersPage(w http.ResponseWriter, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Characters</title></head><body><h1>Characters</h1>")
	if canManageCharacters {
		_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/characters/create\"><input type=\"text\" name=\"name\" placeholder=\"character name\" required><select name=\"kind\"><option value=\"pc\">pc</option><option value=\"npc\">npc</option></select><button type=\"submit\">Create Character</button></form>")
	}
	_, _ = io.WriteString(w, "<ul>")
	for _, character := range characters {
		if character == nil {
			continue
		}
		characterID := strings.TrimSpace(character.GetId())
		name := strings.TrimSpace(character.GetName())
		if name == "" {
			name = characterID
		}
		_, _ = io.WriteString(w, "<li>")
		if characterID != "" {
			_, _ = io.WriteString(w, "<a href=\"/app/campaigns/"+escapedCampaignID+"/characters/"+html.EscapeString(characterID)+"\">"+html.EscapeString(name)+"</a>")
		} else {
			_, _ = io.WriteString(w, html.EscapeString(name))
		}
		if canManageCharacters {
			if characterID != "" {
				selectedKind := characterKindFormValue(character.GetKind())
				pcSelected := ""
				npcSelected := ""
				if selectedKind == "npc" {
					npcSelected = " selected"
				} else {
					pcSelected = " selected"
				}
				_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/characters/update\"><input type=\"hidden\" name=\"character_id\" value=\""+html.EscapeString(characterID)+"\"><input type=\"text\" name=\"name\" value=\""+html.EscapeString(strings.TrimSpace(character.GetName()))+"\"><select name=\"kind\"><option value=\"pc\""+pcSelected+">pc</option><option value=\"npc\""+npcSelected+">npc</option></select><button type=\"submit\">Update Character</button></form>")

				currentParticipantID := ""
				if character.GetParticipantId() != nil {
					currentParticipantID = strings.TrimSpace(character.GetParticipantId().GetValue())
				}
				_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/characters/control\"><input type=\"hidden\" name=\"character_id\" value=\""+html.EscapeString(characterID)+"\"><select name=\"participant_id\">")
				unassignedSelected := ""
				if currentParticipantID == "" {
					unassignedSelected = " selected"
				}
				_, _ = io.WriteString(w, "<option value=\"\""+unassignedSelected+">unassigned</option>")
				for _, participant := range controlParticipants {
					if participant == nil {
						continue
					}
					participantID := strings.TrimSpace(participant.GetId())
					if participantID == "" {
						continue
					}
					label := participantControlFormLabel(participant)
					if label == "" {
						continue
					}
					selected := ""
					if participantID == currentParticipantID {
						selected = " selected"
					}
					_, _ = io.WriteString(w, "<option value=\""+html.EscapeString(participantID)+"\""+selected+">"+html.EscapeString(label)+"</option>")
				}
				_, _ = io.WriteString(w, "</select><button type=\"submit\">Set Controller</button></form>")
			}
		}
		_, _ = io.WriteString(w, "</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}

func renderAppCampaignCharacterDetailPage(w http.ResponseWriter, campaignID string, character *statev1.Character) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	characterID := strings.TrimSpace(character.GetId())
	characterName := strings.TrimSpace(character.GetName())
	if characterName == "" {
		characterName = characterID
	}

	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Character</title></head><body><h1>"+html.EscapeString(characterName)+"</h1>")
	if characterID != "" {
		_, _ = io.WriteString(w, "<p>Character ID: "+html.EscapeString(characterID)+"</p>")
	}
	_, _ = io.WriteString(w, "<p>Kind: "+html.EscapeString(characterKindLabel(character.GetKind()))+"</p>")
	_, _ = io.WriteString(w, "<p><a href=\"/app/campaigns/"+escapedCampaignID+"/characters\">Back to Characters</a></p>")
	_, _ = io.WriteString(w, "</body></html>")
}

func characterKindLabel(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "pc"
	case statev1.CharacterKind_NPC:
		return "npc"
	default:
		return "unspecified"
	}
}
