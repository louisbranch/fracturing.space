package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (h *handler) handleAppCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignCharacters renders all characters for a campaign once
	// membership and capability checks pass.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actingParticipant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Characters unavailable", "character service client is not configured")
		return
	}
	canManageCharacters := canManageCampaignCharacters(actingParticipant.GetCampaignAccess())
	controlParticipants := []*statev1.Participant(nil)
	if canManageCharacters {
		if h.participantClient == nil {
			h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Characters unavailable", "participant service client is not configured")
			return
		}
		if cachedParticipants, ok := h.cachedCampaignParticipants(r.Context(), campaignID); ok {
			controlParticipants = cachedParticipants
		} else {
			resp, err := h.participantClient.ListParticipants(r.Context(), &statev1.ListParticipantsRequest{
				CampaignId: campaignID,
				PageSize:   10,
			})
			if err != nil {
				h.renderErrorPage(w, r, http.StatusBadGateway, "Characters unavailable", "failed to list participants")
				return
			}
			controlParticipants = resp.GetParticipants()
			h.setCampaignParticipantsCache(r.Context(), campaignID, controlParticipants)
		}
	}

	characters := []*statev1.Character(nil)
	if cachedCharacters, ok := h.cachedCampaignCharacters(r.Context(), campaignID); ok {
		characters = cachedCharacters
	} else {
		resp, err := h.characterClient.ListCharacters(r.Context(), &statev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, http.StatusBadGateway, "Characters unavailable", "failed to list characters")
			return
		}
		characters = resp.GetCharacters()
		h.setCampaignCharactersCache(r.Context(), campaignID, characters)
	}

	renderAppCampaignCharactersPage(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, characters, canManageCharacters, controlParticipants)
}

func (h *handler) handleAppCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	// handleAppCampaignCharacterDetail loads a single character sheet to support
	// detailed editing from the campaign context.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if _, ok := h.requireCampaignActor(w, r, campaignID); !ok {
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

	renderAppCampaignCharacterDetailPage(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, resp.GetCharacter())
}

func (h *handler) handleAppCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignCharacterCreate validates owner/manager intent from the
	// caller before creating a new character aggregate.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actingParticipant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
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

	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actingParticipant.GetId()))
	_, err := h.characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       kind,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to create character")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/characters", http.StatusFound)
}

func (h *handler) handleAppCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignCharacterUpdate applies partial character updates while
	// preserving idempotent form-to-domain conversions for name and kind.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actingParticipant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
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

	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actingParticipant.GetId()))
	_, err := h.characterClient.UpdateCharacter(ctx, req)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to update character")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/characters", http.StatusFound)
}

func (h *handler) handleAppCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignCharacterControl assigns character control ownership so game
	// actions can be routed to the correct participant at the session layer.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actingParticipant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.characterClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
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

	if !canManageCampaignCharacters(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actingParticipant.GetId()))
	_, err := h.characterClient.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(targetParticipantID),
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to set character controller")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/characters", http.StatusFound)
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
	label := strings.TrimSpace(participant.GetName())
	if label == "" {
		label = strings.TrimSpace(participant.GetUserId())
	}
	if label == "" {
		label = strings.TrimSpace(participant.GetId())
	}
	return label
}

func renderAppCampaignCharactersPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
	renderAppCampaignCharactersPageWithContext(w, r, page, campaignID, characters, canManageCharacters, controlParticipants)
}

func renderAppCampaignCharactersPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
	// renderAppCampaignCharactersPage keeps the write controls tied to current
	// campaign access level so members cannot reach management operations.
	campaignID = strings.TrimSpace(campaignID)
	characterItems := make([]webtemplates.CharacterListItem, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}
		characterID := strings.TrimSpace(character.GetId())
		editName := strings.TrimSpace(character.GetName())
		displayName := editName
		if displayName == "" {
			displayName = characterID
		}
		selectedKind := characterKindFormValue(character.GetKind())
		currentParticipantID := ""
		if character.GetParticipantId() != nil {
			currentParticipantID = strings.TrimSpace(character.GetParticipantId().GetValue())
		}

		controlOptions := make([]webtemplates.CharacterControlOption, 0, len(controlParticipants)+1)
		controlOptions = append(controlOptions, webtemplates.CharacterControlOption{
			ID:       "",
			Label:    webtemplates.T(page.Loc, "game.participants.value_unassigned"),
			Selected: currentParticipantID == "",
		})
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
			controlOptions = append(controlOptions, webtemplates.CharacterControlOption{
				ID:       participantID,
				Label:    label,
				Selected: participantID == currentParticipantID,
			})
		}

		characterItems = append(characterItems, webtemplates.CharacterListItem{
			ID:           characterID,
			DisplayName:  displayName,
			EditableName: editName,
			Kind:         selectedKind,
			PCSelected:   selectedKind == "pc",
			NPCSelected:  selectedKind == "npc",
			// canManageCharacters controls form visibility at the page level.
			// This item intentionally does not include per-character permissions.
			ControlOptions: controlOptions,
		})
	}
	if err := writePage(w, r, webtemplates.CampaignCharactersPage(page, campaignID, canManageCharacters, characterItems), composeHTMXTitleForPage(page, "game.characters.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_characters_page")
	}
}

func renderAppCampaignCharacterDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
	renderAppCampaignCharacterDetailPageWithContext(w, r, page, campaignID, character)
}

func renderAppCampaignCharacterDetailPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
	// renderAppCampaignCharacterDetailPage provides the stable read surface for a
	// single character without mutating state.
	if character == nil {
		character = &statev1.Character{}
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID := strings.TrimSpace(character.GetId())
	characterName := strings.TrimSpace(character.GetName())
	if characterName == "" {
		characterName = characterID
	}
	detail := webtemplates.CharacterDetail{
		CampaignID: campaignID,
		ID:         characterID,
		Name:       characterName,
		Kind:       characterKindLabel(page.Loc, character.GetKind()),
	}
	if err := writePage(w, r, webtemplates.CharacterDetailPage(page, detail), composeHTMXTitleForPage(page, "game.character_detail.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_character_detail_page")
	}
}

func characterKindLabel(loc webtemplates.Localizer, kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return webtemplates.T(loc, "game.character_kind.pc")
	case statev1.CharacterKind_NPC:
		return webtemplates.T(loc, "game.character_kind.npc")
	default:
		return webtemplates.T(loc, "game.character_detail.kind_unspecified")
	}
}
