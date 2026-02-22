package web

import (
	"log"
	"net/http"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// renderErrorPage converts internal transport and auth errors to a localized web
// error template, so failure states stay in one shared UX surface.
func (h *handler) renderErrorPage(w http.ResponseWriter, r *http.Request, status int, title string, message string) {
	page := h.pageContext(w, r)
	localizedTitle := localizeErrorPageText(page.Loc, title, errorPageTitleTextKeys)
	localizedMessage := localizeErrorPageText(page.Loc, message, errorPageMessageTextKeys)
	writeGameContentType(w)
	w.WriteHeader(status)
	if err := h.writePage(
		w,
		r,
		webtemplates.ErrorPage(page, localizedTitle, localizedMessage),
		composeHTMXTitleForPage(page, localizedTitle),
	); err != nil {
		log.Printf("web: failed to render error page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func localizeErrorPageText(loc webtemplates.Localizer, raw string, keyMap map[string]string) string {
	key, ok := keyMap[raw]
	if !ok {
		return webtemplates.T(loc, raw)
	}
	return webtemplates.T(loc, key)
}

func localizeHTTPError(w http.ResponseWriter, r *http.Request, status int, key string, args ...any) {
	printer, _ := localizer(w, r)
	http.Error(w, webtemplates.T(printer, key, args...), status)
}

var errorPageTitleTextKeys = map[string]string{
	"Access denied":                     "error.web.title.access_denied",
	"Authentication required":           "error.web.title.authentication_required",
	"Profile unavailable":               "error.web.title.profile_unavailable",
	"Campaign create unavailable":       "error.web.title.campaign_create_unavailable",
	"Campaign unavailable":              "error.web.title.campaign_unavailable",
	"Campaigns unavailable":             "error.web.title.campaigns_unavailable",
	"Character action unavailable":      "error.web.title.character_action_unavailable",
	"Character unavailable":             "error.web.title.character_unavailable",
	"Characters unavailable":            "error.web.title.characters_unavailable",
	"Profile update failed":             "error.web.title.profile_update_failed",
	"Invalid claim request":             "error.web.title.invalid_claim_request",
	"Invite action unavailable":         "error.web.title.invite_action_unavailable",
	"Invite claim unavailable":          "error.web.title.invite_claim_unavailable",
	"Invites unavailable":               "error.web.title.invites_unavailable",
	"Participant action unavailable":    "error.web.title.participant_action_unavailable",
	"Participants unavailable":          "error.web.title.participants_unavailable",
	"Session action unavailable":        "error.web.title.session_action_unavailable",
	"Session unavailable":               "error.web.title.session_unavailable",
	"Sessions unavailable":              "error.web.title.sessions_unavailable",
	"AI keys unavailable":               "error.web.title.ai_keys_unavailable",
	"AI key action unavailable":         "error.web.title.ai_key_action_unavailable",
	"User profile settings unavailable": "error.web.title.user_profile_settings_unavailable",
	"Notification unavailable":          "error.web.title.notification_unavailable",
	"Notifications unavailable":         "error.web.title.notifications_unavailable",
}

var errorPageMessageTextKeys = map[string]string{
	"at least one character field is required":                "error.web.message.at_least_one_character_field_is_required",
	"at least one participant field is required":              "error.web.message.at_least_one_participant_field_is_required",
	"campaign access checker is not configured":               "error.web.message.campaign_access_checker_is_not_configured",
	"campaign access value is invalid":                        "error.web.message.campaign_access_value_is_invalid",
	"campaign gm mode is invalid":                             "error.web.message.campaign_gm_mode_is_invalid",
	"campaign invite service is not configured":               "error.web.message.campaign_invite_service_is_not_configured",
	"campaign name is required":                               "error.web.message.campaign_name_is_required",
	"campaign service client is not configured":               "error.web.message.campaign_service_client_is_not_configured",
	"campaign system is invalid":                              "error.web.message.campaign_system_is_invalid",
	"campaign, invite, and participant ids are required":      "error.web.message.campaign_invite_and_participant_ids_are_required",
	"character id is required":                                "error.web.message.character_id_is_required",
	"character kind value is invalid":                         "error.web.message.character_kind_value_is_invalid",
	"character name is required":                              "error.web.message.character_name_is_required",
	"character not found":                                     "error.web.message.character_not_found",
	"character service client is not configured":              "error.web.message.character_service_client_is_not_configured",
	"created campaign id was empty":                           "error.web.message.created_campaign_id_was_empty",
	"failed to claim invite":                                  "error.web.message.failed_to_claim_invite",
	"failed to create campaign":                               "error.web.message.failed_to_create_campaign",
	"failed to create invite":                                 "error.web.message.failed_to_create_invite",
	"failed to create character":                              "error.web.message.failed_to_create_character",
	"failed to issue join grant":                              "error.web.message.failed_to_issue_join_grant",
	"failed to list campaign invites":                         "error.web.message.failed_to_list_campaign_invites",
	"failed to list campaigns":                                "error.web.message.failed_to_list_campaigns",
	"failed to list characters":                               "error.web.message.failed_to_list_characters",
	"failed to list participants":                             "error.web.message.failed_to_list_participants",
	"failed to list pending invites":                          "error.web.message.failed_to_list_pending_invites",
	"failed to list sessions":                                 "error.web.message.failed_to_list_sessions",
	"failed to revoke invite":                                 "error.web.message.failed_to_revoke_invite",
	"failed to load character":                                "error.web.message.failed_to_load_character",
	"failed to load session":                                  "error.web.message.failed_to_load_session",
	"failed to parse campaign create form":                    "error.web.message.failed_to_parse_campaign_create_form",
	"failed to parse character controller form":               "error.web.message.failed_to_parse_character_controller_form",
	"failed to parse character create form":                   "error.web.message.failed_to_parse_character_create_form",
	"failed to parse character update form":                   "error.web.message.failed_to_parse_character_update_form",
	"failed to parse claim form":                              "error.web.message.failed_to_parse_claim_form",
	"failed to parse profile form":                            "error.web.message.failed_to_parse_profile_form",
	"failed to parse profile locale":                          "error.web.message.failed_to_parse_profile_locale",
	"failed to parse invite create form":                      "error.web.message.failed_to_parse_invite_create_form",
	"failed to parse invite revoke form":                      "error.web.message.failed_to_parse_invite_revoke_form",
	"failed to parse user profile form":                       "error.web.message.failed_to_parse_user_profile_form",
	"failed to parse participant update form":                 "error.web.message.failed_to_parse_participant_update_form",
	"failed to parse session end form":                        "error.web.message.failed_to_parse_session_end_form",
	"failed to parse session start form":                      "error.web.message.failed_to_parse_session_start_form",
	"failed to resolve current user":                          "error.web.message.failed_to_resolve_current_user",
	"failed to load profile":                                  "error.web.message.failed_to_load_profile",
	"failed to load user profile":                             "error.web.message.failed_to_load_user_profile",
	"account service client is not configured":                "error.web.message.account_service_client_is_not_configured",
	"failed to set character controller":                      "error.web.message.failed_to_set_character_controller",
	"failed to update profile":                                "error.web.message.failed_to_update_profile",
	"failed to update character":                              "error.web.message.failed_to_update_character",
	"failed to update participant":                            "error.web.message.failed_to_update_participant",
	"failed to verify campaign access":                        "error.web.message.failed_to_verify_campaign_access",
	"invite claim dependencies are not configured":            "error.web.message.invite_claim_dependencies_are_not_configured",
	"invite id is required":                                   "error.web.message.invite_id_is_required",
	"invite service client is not configured":                 "error.web.message.invite_service_client_is_not_configured",
	"join grant was empty":                                    "error.web.message.join_grant_was_empty",
	"manager or owner access required for character action":   "error.web.message.manager_or_owner_access_required_for_character_action",
	"manager or owner access required for invite access":      "error.web.message.manager_or_owner_access_required_for_invite_access",
	"manager or owner access required for invite action":      "error.web.message.manager_or_owner_access_required_for_invite_action",
	"manager or owner access required for participant action": "error.web.message.manager_or_owner_access_required_for_participant_action",
	"manager or owner access required for session action":     "error.web.message.manager_or_owner_access_required_for_session_action",
	"no user identity was resolved for this session":          "error.web.message.no_user_identity_was_resolved_for_this_session",
	"participant access required":                             "error.web.message.participant_access_required",
	"participant controller value is invalid":                 "error.web.message.participant_controller_value_is_invalid",
	"participant id is required":                              "error.web.message.participant_id_is_required",
	"participant role value is invalid":                       "error.web.message.participant_role_value_is_invalid",
	"participant service client is not configured":            "error.web.message.participant_service_client_is_not_configured",
	"recipient username is required":                          "error.web.message.recipient_username_is_required",
	"recipient username must start with @":                    "error.web.message.recipient_username_must_start_with_at",
	"recipient username is invalid":                           "error.web.message.recipient_username_is_invalid",
	"recipient username was not found":                        "error.web.message.recipient_username_was_not_found",
	"failed to resolve invite recipient":                      "error.web.message.failed_to_resolve_invite_recipient",
	"connections service is not configured":                   "error.web.message.connections_service_is_not_configured",
	"session id is required":                                  "error.web.message.session_id_is_required",
	"session name is required":                                "error.web.message.session_name_is_required",
	"session not found":                                       "error.web.message.session_not_found",
	"session service client is not configured":                "error.web.message.session_service_client_is_not_configured",
	"failed to end session":                                   "error.web.message.failed_to_end_session",
	"failed to start session":                                 "error.web.message.failed_to_start_session",
	"credential service client is not configured":             "error.web.message.credential_service_client_is_not_configured",
	"failed to list ai keys":                                  "error.web.message.failed_to_list_ai_keys",
	"failed to parse ai key form":                             "error.web.message.failed_to_parse_ai_key_form",
	"failed to revoke ai key":                                 "error.web.message.failed_to_revoke_ai_key",
	"ai key id is required":                                   "error.web.message.ai_key_id_is_required",
	"notification service client is not configured":           "error.web.message.notification_service_client_is_not_configured",
	"failed to list notifications":                            "error.web.message.failed_to_list_notifications",
	"failed to mark notification read":                        "error.web.message.failed_to_mark_notification_read",
}
