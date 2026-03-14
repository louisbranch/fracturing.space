package render

import (
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// campaignCharacterDetailURL centralizes character detail links for render-owned cards.
func campaignCharacterDetailURL(view DetailView, character CharacterView) string {
	campaignID := strings.TrimSpace(view.CampaignID)
	characterID := strings.TrimSpace(character.ID)
	if campaignID == "" || characterID == "" {
		return ""
	}
	return routepath.AppCampaignCharacter(campaignID, characterID)
}

// campaignParticipantEditURL keeps participant edit links consistent with routepath.
func campaignParticipantEditURL(view DetailView, participant ParticipantView) string {
	campaignID := strings.TrimSpace(view.CampaignID)
	participantID := strings.TrimSpace(participant.ID)
	if campaignID == "" || participantID == "" {
		return ""
	}
	return routepath.AppCampaignParticipantEdit(campaignID, participantID)
}

// campaignCharacterEditURL centralizes character edit links for detail pages.
func campaignCharacterEditURL(view DetailView) string {
	campaignID := strings.TrimSpace(view.CampaignID)
	characterID := strings.TrimSpace(view.CharacterID)
	if campaignID == "" || characterID == "" {
		return ""
	}
	return routepath.AppCampaignCharacterEdit(campaignID, characterID)
}

// campaignCharacterSheetTitle derives the character-creation panel title from the campaign system.
func campaignCharacterSheetTitle(loc Localizer, view DetailView) string {
	system := strings.TrimSpace(campaignOverviewSystem(loc, view))
	if system == "" {
		system = T(loc, "game.campaign.system_unspecified")
	}
	return system + " " + T(loc, "game.character_detail.character_sheet_suffix")
}

// campaignCharacterAliases renders aliases in the same display shape as the old shared fragment.
func campaignCharacterAliases(value []string) string {
	if len(value) == 0 {
		return ""
	}
	return strings.Join(value, ", ")
}

// campaignCharacterHasDaggerheartSummary guards Daggerheart-only metadata sections.
func campaignCharacterHasDaggerheartSummary(character CharacterView) bool {
	if character.Daggerheart == nil {
		return false
	}
	return strings.TrimSpace(character.Daggerheart.ClassName) != "" &&
		strings.TrimSpace(character.Daggerheart.SubclassName) != "" &&
		strings.TrimSpace(character.Daggerheart.AncestryName) != "" &&
		strings.TrimSpace(character.Daggerheart.CommunityName) != "" &&
		character.Daggerheart.Level > 0
}

// campaignCharacterDaggerheartLevelAttr exposes level as a stable data attribute.
func campaignCharacterDaggerheartLevelAttr(character CharacterView) string {
	if !campaignCharacterHasDaggerheartSummary(character) {
		return ""
	}
	return strconv.FormatInt(int64(character.Daggerheart.Level), 10)
}

// campaignCharacterControlOptionLabel keeps controller reassignment labels stable.
func campaignCharacterControlOptionLabel(loc Localizer, option CharacterControlOptionView) string {
	if strings.TrimSpace(option.ParticipantID) == "" {
		return T(loc, "game.participants.value_unassigned")
	}
	label := strings.TrimSpace(option.Label)
	if label == "" {
		return strings.TrimSpace(option.ParticipantID)
	}
	return label
}

// campaignOverviewTheme preserves the explicit empty-theme fallback copy.
func campaignOverviewTheme(loc Localizer, view DetailView) string {
	if strings.TrimSpace(view.Theme) == "" {
		return T(loc, "game.campaign.overview.theme_empty")
	}
	return strings.TrimSpace(view.Theme)
}

// campaignSystemLabel maps persisted system identifiers to contributor-facing copy.
func campaignSystemLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "daggerheart":
		return T(loc, "game.campaigns.system_daggerheart")
	default:
		return raw
	}
}

// campaignGMModeLabel maps GM mode values to localized overview labels.
func campaignGMModeLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "", "unspecified":
		return T(loc, "game.campaign.gm_mode_unspecified")
	case "human":
		return T(loc, "game.create.field_gm_mode_human")
	case "ai":
		return T(loc, "game.create.field_gm_mode_ai")
	case "hybrid":
		return T(loc, "game.create.field_gm_mode_hybrid")
	default:
		return raw
	}
}

// campaignOverviewSystem adapts the detail view to the shared system-label helper.
func campaignOverviewSystem(loc Localizer, view DetailView) string {
	return campaignSystemLabel(loc, view.System)
}

// campaignOverviewGMMode adapts the detail view to the shared GM-mode label helper.
func campaignOverviewGMMode(loc Localizer, view DetailView) string {
	return campaignGMModeLabel(loc, view.GMMode)
}

// campaignOverviewStatus keeps campaign status labels consistent across detail screens.
func campaignOverviewStatus(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "draft":
		return T(loc, "game.campaign.overview.value_draft")
	case "active":
		return T(loc, "game.campaign.overview.value_active")
	case "completed":
		return T(loc, "game.campaign.overview.value_completed")
	case "archived":
		return T(loc, "game.campaign.overview.value_archived")
	default:
		return raw
	}
}

// campaignOverviewLocale maps persisted locale labels to the user-facing copy.
func campaignOverviewLocale(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "english (us)":
		return T(loc, "game.campaign.overview.value_locale_en_us")
	case "portuguese (brazil)":
		return T(loc, "game.campaign.overview.value_locale_pt_br")
	default:
		return raw
	}
}

// campaignOverviewIntent maps campaign intent values to localized detail-page text.
func campaignOverviewIntent(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "standard":
		return T(loc, "game.campaign.overview.value_standard")
	case "starter":
		return T(loc, "game.campaign.overview.value_starter")
	case "sandbox":
		return T(loc, "game.campaign.overview.value_sandbox")
	default:
		return raw
	}
}

// campaignOverviewAccessPolicy maps access-policy values to localized detail-page text.
func campaignOverviewAccessPolicy(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	case "private":
		return T(loc, "game.campaign.overview.value_private")
	case "restricted":
		return T(loc, "game.campaign.overview.value_restricted")
	case "public":
		return T(loc, "game.campaign.overview.value_public")
	default:
		return raw
	}
}

// campaignSessionStatusLabel keeps session tables and detail pages on the same status copy.
func campaignSessionStatusLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.session_status_unspecified")
	case "active":
		return T(loc, "game.campaign.session_status_active")
	case "ended":
		return T(loc, "game.campaign.session_status_ended")
	default:
		return raw
	}
}

// campaignInviteStatusLabel keeps invite tables and detail pages on the same status copy.
func campaignInviteStatusLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign_invites.value_unspecified")
	case "pending":
		return T(loc, "game.campaign_invites.value_pending")
	case "claimed":
		return T(loc, "game.campaign_invites.value_claimed")
	case "revoked":
		return T(loc, "game.campaign_invites.value_revoked")
	default:
		return raw
	}
}

// campaignSessionCanEnd gates end-session affordances to active sessions only.
func campaignSessionCanEnd(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "active")
}

// campaignSessionStartReady exposes the session-readiness contract to templates.
func campaignSessionStartReady(view DetailView) bool {
	return view.SessionReadiness.Ready
}

// campaignActionsLocked exposes the detail-page mutation lock state to templates.
func campaignActionsLocked(view DetailView) bool {
	return view.ActionsLocked
}

// campaignInviteCanRevoke limits revoke affordances to pending invites.
func campaignInviteCanRevoke(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "pending")
}

// campaignParticipantRoleLabel maps participant roles to localized card and form copy.
func campaignParticipantRoleLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "gm":
		return T(loc, "game.participants.value.gm")
	case "player":
		return T(loc, "game.participants.value.player")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// campaignParticipantAccessLabel maps participant access values to localized labels.
func campaignParticipantAccessLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "member":
		return T(loc, "game.participants.value.member")
	case "manager":
		return T(loc, "game.participants.value.manager")
	case "owner":
		return T(loc, "game.participants.value.owner")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// campaignParticipantControllerLabel maps controller values to localized participant copy.
func campaignParticipantControllerLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "human":
		return T(loc, "game.participants.value.human")
	case "ai":
		return T(loc, "game.participants.value_ai")
	case "unassigned":
		return T(loc, "game.participants.value_unassigned")
	case "", "unspecified":
		return T(loc, "game.campaign.system_unspecified")
	default:
		return raw
	}
}

// participantPronounsLabel preserves the display mapping used across participant and character cards.
func participantPronounsLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return raw
	case "she/her":
		return T(loc, "game.participants.value_she_her")
	case "he/him":
		return T(loc, "game.participants.value_he_him")
	case "they/them":
		return T(loc, "game.participants.value_they_them")
	case "it/its":
		return T(loc, "game.participants.value_it_its")
	default:
		return raw
	}
}

// campaignParticipantPronounPresets keeps participant-edit suggestions in the render seam.
func campaignParticipantPronounPresets(loc Localizer, editor ParticipantEditorView) []string {
	presets := []string{
		T(loc, "game.participants.value_she_her"),
		T(loc, "game.participants.value_he_him"),
		T(loc, "game.participants.value_they_them"),
	}
	if campaignParticipantControllerCanonical(editor.Controller) == "ai" {
		return append(presets, T(loc, "game.participants.value_it_its"))
	}
	return presets
}

// campaignCharacterPronounPresets keeps character-edit suggestions in the render seam.
func campaignCharacterPronounPresets(loc Localizer) []string {
	return []string{
		T(loc, "game.participants.value_they_them"),
		T(loc, "game.participants.value_he_him"),
		T(loc, "game.participants.value_she_her"),
		T(loc, "game.participants.value_it_its"),
	}
}

// campaignParticipantControllerCanonical normalizes controller values from backend and form inputs.
func campaignParticipantControllerCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ai", "controller_ai":
		return "ai"
	case "human", "controller_human":
		return "human"
	case "unassigned", "controller_unassigned":
		return "unassigned"
	default:
		return ""
	}
}

// campaignParticipantRoleFormValue ensures participant edit forms always submit a concrete role.
func campaignParticipantRoleFormValue(editor ParticipantEditorView) string {
	if value := campaignParticipantRoleCanonical(editor.Role); value != "" {
		return value
	}
	return "gm"
}

// campaignParticipantRoleCanonical normalizes participant role values for comparisons and forms.
func campaignParticipantRoleCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gm", "participant_role_gm", "role_gm":
		return "gm"
	case "player", "participant_role_player", "role_player":
		return "player"
	default:
		return ""
	}
}

// campaignParticipantAccessCanonical normalizes participant access values for comparisons and forms.
func campaignParticipantAccessCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "member", "campaign_access_member":
		return "member"
	case "manager", "campaign_access_manager":
		return "manager"
	case "owner", "campaign_access_owner":
		return "owner"
	default:
		return ""
	}
}

// campaignParticipantAccessFormValue ensures participant edit forms always submit a concrete access value.
func campaignParticipantAccessFormValue(editor ParticipantEditorView) string {
	if value := campaignParticipantAccessCanonical(editor.CampaignAccess); value != "" {
		return value
	}
	return "member"
}

// campaignParticipantEditLayout exposes whether AI-binding UI shares the edit screen.
func campaignParticipantEditLayout(view DetailView) string {
	if view.AIBindingEditor.Visible {
		return "ai"
	}
	return "standard"
}

// campaignAIBindingCurrentUnbound drives the unbound-state copy for AI binding editors.
func campaignAIBindingCurrentUnbound(editor AIBindingEditorView) bool {
	return strings.TrimSpace(editor.CurrentID) == ""
}

// campaignCharacterKindLabel maps character kind values to localized detail copy.
func campaignCharacterKindLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	value = strings.ToLower(raw)
	switch value {
	case "pc":
		return T(loc, "game.characters.value_pc")
	case "npc":
		return T(loc, "game.characters.value_npc")
	case "", "unspecified":
		return T(loc, "game.character_detail.kind_unspecified")
	default:
		return raw
	}
}

// campaignSessionByID resolves the selected session without forcing handlers to pre-split the view.
func campaignSessionByID(loc Localizer, view DetailView) SessionView {
	sessionID := strings.TrimSpace(view.SessionID)
	if sessionID == "" {
		return SessionView{}
	}
	for _, session := range view.Sessions {
		if strings.TrimSpace(session.ID) == sessionID {
			return session
		}
	}
	return SessionView{
		ID:        sessionID,
		Name:      sessionID,
		Status:    campaignSessionStatusLabel(loc, "Unspecified"),
		StartedAt: "",
		UpdatedAt: "",
		EndedAt:   "",
	}
}

// campaignCharacterByID resolves the selected character without duplicating handler-only lookup code.
func campaignCharacterByID(view DetailView) CharacterView {
	characterID := strings.TrimSpace(view.CharacterID)
	if characterID == "" {
		return CharacterView{}
	}
	for _, character := range view.Characters {
		if strings.TrimSpace(character.ID) == characterID {
			return character
		}
	}
	return CharacterView{ID: characterID, Kind: "Unspecified", Controller: "Unassigned"}
}

// campaignCharacterDisplayName preserves the detail-page fallback title for unnamed characters.
func campaignCharacterDisplayName(loc Localizer, character CharacterView) string {
	name := strings.TrimSpace(character.Name)
	if name != "" {
		return name
	}
	return T(loc, "game.character_detail.title")
}
