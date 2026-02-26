package campaign

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CampaignThemePromptLimit defines how many grapheme runes are shown in preview text.
const CampaignThemePromptLimit = 80

var campaignCoverManifest = catalog.CampaignCoverManifest()

// RenderCampaignsPage renders the campaigns index shell with canonical page chrome.
func RenderCampaignsPage(w http.ResponseWriter, r *http.Request, campaigns []*statev1.Campaign) {
	RenderCampaignsListPageWithConfig(w, r, webtemplates.PageContext{}, "", campaigns)
}

// RenderCampaignsListPage renders the campaigns list with the provided app context.
func RenderCampaignsListPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaigns []*statev1.Campaign, assetBaseURL string) {
	RenderCampaignsListPageWithConfig(w, r, page, assetBaseURL, campaigns)
}

// RenderCampaignsListPageWithConfig renders the campaigns list and applies cover resolution.
func RenderCampaignsListPageWithConfig(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, assetBaseURL string, campaigns []*statev1.Campaign) {
	sortedCampaigns := append([]*statev1.Campaign(nil), campaigns...)
	sort.SliceStable(sortedCampaigns, func(i, j int) bool {
		return CampaignCreatedAtUnixNano(sortedCampaigns[i]) > CampaignCreatedAtUnixNano(sortedCampaigns[j])
	})

	normalized := make([]webtemplates.CampaignListItem, 0, len(sortedCampaigns))
	for _, campaign := range sortedCampaigns {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		normalized = append(normalized, webtemplates.CampaignListItem{
			ID:               campaignID,
			Name:             name,
			Theme:            TruncateCampaignTheme(campaign.GetThemePrompt()),
			CoverImageURL:    CampaignCoverImageURL(assetBaseURL, campaignID, campaign.GetCoverSetId(), campaign.GetCoverAssetId()),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		})
	}
	if err := support.WritePage(w, r, webtemplates.CampaignsListPage(page, normalized), support.ComposeHTMXTitleForPage(page, "game.campaigns.title")); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_campaigns_list_page")
	}
}

// RenderCampaignCreatePage renders the campaign creation form.
func RenderCampaignCreatePage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext) {
	RenderCampaignCreatePageWithContext(w, r, page)
}

// RenderCampaignCreatePageWithContext renders the campaign creation form with a page context.
func RenderCampaignCreatePageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext) {
	if err := support.WritePage(w, r, webtemplates.CampaignCreatePage(page), support.ComposeHTMXTitleForPage(page, "game.create.title")); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_campaign_create_page")
	}
}

// RenderCampaignSessionsPage maps campaign sessions into a list view.
func RenderCampaignSessionsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	RenderCampaignSessionsPageWithContext(w, r, page, campaignID, sessions, canManageSessions)
}

// RenderCampaignSessionsPageWithContext maps campaign sessions into a list view with context.
func RenderCampaignSessionsPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	campaignID = strings.TrimSpace(campaignID)
	sessionItems := make([]webtemplates.SessionListItem, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		sessionID := strings.TrimSpace(session.GetId())
		name := strings.TrimSpace(session.GetName())
		if name == "" {
			name = sessionID
		}
		sessionItems = append(sessionItems, webtemplates.SessionListItem{
			ID:       sessionID,
			Name:     name,
			IsActive: session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE,
		})
	}
	if err := support.WritePage(
		w,
		r,
		webtemplates.SessionsListPage(page, campaignID, canManageSessions, sessionItems),
		support.ComposeHTMXTitleForPage(page, "game.sessions.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_sessions_page")
	}
}

// RenderCampaignSessionDetailPage renders one campaign session detail page.
func RenderCampaignSessionDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
	RenderCampaignSessionDetailPageWithContext(w, r, page, campaignID, session)
}

// RenderCampaignSessionDetailPageWithContext renders one campaign session detail page with context.
func RenderCampaignSessionDetailPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
	if session == nil {
		session = &statev1.Session{}
	}
	campaignID = strings.TrimSpace(campaignID)
	sessionID := strings.TrimSpace(session.GetId())
	sessionName := strings.TrimSpace(session.GetName())
	if sessionName == "" {
		sessionName = sessionID
	}
	detail := webtemplates.SessionDetail{
		CampaignID: campaignID,
		ID:         sessionID,
		Name:       sessionName,
		Status:     SessionStatusLabel(page.Loc, session.GetStatus()),
	}
	if err := support.WritePage(
		w,
		r,
		webtemplates.SessionDetailPage(page, detail),
		support.ComposeHTMXTitleForPage(page, "game.session_detail.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_session_detail_page")
	}
}

// RenderCampaignParticipantsPage maps participant rows for the campaign participants page.
func RenderCampaignParticipantsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	RenderCampaignParticipantsPageWithContext(w, r, page, campaignID, participants, canManageParticipants)
}

// RenderCampaignParticipantsPageWithContext maps participant rows for the campaign participants page.
func RenderCampaignParticipantsPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	campaignID = strings.TrimSpace(campaignID)
	participantItems := make([]webtemplates.ParticipantListItem, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		name := strings.TrimSpace(participant.GetName())
		if name == "" {
			name = strings.TrimSpace(participant.GetUserId())
		}
		if name == "" {
			name = strings.TrimSpace(participant.GetId())
		}
		accessValue := CampaignAccessFormValue(participant.GetCampaignAccess())
		roleValue := ParticipantRoleFormValue(participant.GetRole())
		controllerValue := ParticipantControllerFormValue(participant.GetController())
		participantItems = append(participantItems, webtemplates.ParticipantListItem{
			ID:              strings.TrimSpace(participant.GetId()),
			Name:            name,
			MemberSelected:  accessValue == "member",
			ManagerSelected: accessValue == "manager",
			OwnerSelected:   accessValue == "owner",
			GMSelected:      roleValue == "gm",
			PlayerSelected:  roleValue == "player",
			HumanSelected:   controllerValue == "human",
			AISelected:      controllerValue == "ai",
		})
	}
	if err := support.WritePage(
		w,
		r,
		webtemplates.CampaignParticipantsPage(page, campaignID, canManageParticipants, participantItems),
		support.ComposeHTMXTitleForPage(page, "game.participants.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_participants_page")
	}
}

// RenderCampaignCharactersPage maps campaign characters into list view rows.
func RenderCampaignCharactersPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
	RenderCampaignCharactersPageWithContext(w, r, page, campaignID, characters, canManageCharacters, controlParticipants)
}

// RenderCampaignCharactersPageWithContext maps campaign characters into list view rows.
func RenderCampaignCharactersPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
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
		selectedKind := CharacterKindFormValue(character.GetKind())
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
			label := ParticipantControlFormLabel(participant)
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
			ID:             characterID,
			DisplayName:    displayName,
			EditableName:   editName,
			Kind:           selectedKind,
			PCSelected:     selectedKind == "pc",
			NPCSelected:    selectedKind == "npc",
			ControlOptions: controlOptions,
		})
	}
	if err := support.WritePage(
		w,
		r,
		webtemplates.CampaignCharactersPage(page, campaignID, canManageCharacters, characterItems),
		support.ComposeHTMXTitleForPage(page, "game.characters.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_characters_page")
	}
}

// RenderCampaignCharacterDetailPage renders one character details page.
func RenderCampaignCharacterDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
	RenderCampaignCharacterDetailPageWithContext(w, r, page, campaignID, character)
}

// RenderCampaignCharacterDetailPageWithContext renders one character detail page with context.
func RenderCampaignCharacterDetailPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
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
		Kind:       CharacterKindLabel(page.Loc, character.GetKind()),
	}
	if err := support.WritePage(
		w,
		r,
		webtemplates.CharacterDetailPage(page, detail),
		support.ComposeHTMXTitleForPage(page, "game.character_detail.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_character_detail_page")
	}
}

// RenderCampaignInvitesPage maps campaign invites into list content.
func RenderCampaignInvitesPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool) {
	RenderCampaignInvitesPageWithContext(w, r, page, campaignID, invites, contacts, canManageInvites, webtemplates.CampaignInviteVerification{})
}

// RenderCampaignInvitesPageWithContext maps campaign invites into list content.
func RenderCampaignInvitesPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
	campaignID = strings.TrimSpace(campaignID)
	inviteItems := make([]webtemplates.CampaignInviteItem, 0, len(invites))
	unknownInviteID := "unknown-invite"
	unknownRecipient := "unknown-recipient"
	if page.Loc != nil {
		unknownInviteID = webtemplates.T(page.Loc, "game.campaign_invite.unknown_id")
		unknownRecipient = webtemplates.T(page.Loc, "game.campaign_invite.unknown_recipient")
	}
	for _, invite := range invites {
		if invite == nil {
			continue
		}
		inviteID := strings.TrimSpace(invite.GetId())
		displayInviteID := inviteID
		if displayInviteID == "" {
			displayInviteID = unknownInviteID
		}
		recipient := strings.TrimSpace(invite.GetRecipientUserId())
		if recipient == "" {
			recipient = unknownRecipient
		}
		inviteItems = append(inviteItems, webtemplates.CampaignInviteItem{
			ID:    inviteID,
			Label: displayInviteID + " - " + recipient,
		})
	}
	if err := support.WritePage(w, r, webtemplates.CampaignInvitesPage(page, campaignID, canManageInvites, inviteItems, contacts, verification), support.ComposeHTMXTitleForPage(page, "game.campaign_invites.title")); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_campaign_invites_page")
	}
}

// TruncateCampaignTheme normalizes and shortens a campaign theme prompt.
func TruncateCampaignTheme(themePrompt string) string {
	runes := []rune(strings.TrimSpace(themePrompt))
	if CampaignThemePromptLimit <= 0 || len(runes) == 0 {
		return ""
	}
	if len(runes) <= CampaignThemePromptLimit {
		return string(runes)
	}
	return string(runes[:CampaignThemePromptLimit]) + "..."
}

// CampaignCreatedAtUnixNano returns campaign creation time for deterministic sorting.
func CampaignCreatedAtUnixNano(campaign *statev1.Campaign) int64 {
	if campaign == nil || campaign.GetCreatedAt() == nil {
		return 0
	}
	return campaign.GetCreatedAt().AsTime().UTC().UnixNano()
}

// CampaignCoverImageURL selects the resolved campaign cover asset URL.
func CampaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string) string {
	_, resolvedCoverAssetID := ResolveWebCampaignCoverSelection(campaignID, coverSetID, coverAssetID)
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   resolvedCoverAssetID,
		Extension: ".png",
	})
	if err == nil {
		return resolvedAssetURL
	}
	return "/static/campaign-covers/" + url.PathEscape(resolvedCoverAssetID) + ".png"
}

// NormalizeCampaignCoverAssetID validates a requested campaign cover asset identifier.
func NormalizeCampaignCoverAssetID(raw string) (string, bool) {
	normalizedCoverAssetID := campaignCoverManifest.NormalizeAssetID(raw)
	if normalizedCoverAssetID == "" {
		return "", false
	}
	if !campaignCoverManifest.ValidateAssetInSet(catalog.CampaignCoverSetV1, normalizedCoverAssetID) {
		return "", false
	}
	return normalizedCoverAssetID, true
}

func defaultCampaignCoverAssetID() string {
	coverSet, ok := campaignCoverManifest.Sets[catalog.CampaignCoverSetV1]
	if !ok || len(coverSet.AssetIDs) == 0 {
		return ""
	}
	return coverSet.AssetIDs[0]
}

// ResolveWebCampaignCoverSelection resolves the campaign cover set and asset tuple.
func ResolveWebCampaignCoverSelection(campaignID, coverSetID, coverAssetID string) (string, string) {
	resolvedCoverSetID, resolvedCoverAssetID, err := campaignCoverManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: "campaign",
		EntityID:   strings.TrimSpace(campaignID),
		SetID:      coverSetID,
		AssetID:    coverAssetID,
	})
	if err == nil {
		return resolvedCoverSetID, resolvedCoverAssetID
	}

	fallbackCoverSetID, fallbackCoverAssetID, fallbackErr := campaignCoverManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: "campaign",
		EntityID:   strings.TrimSpace(campaignID),
		SetID:      catalog.CampaignCoverSetV1,
		AssetID:    "",
	})
	if fallbackErr == nil {
		return fallbackCoverSetID, fallbackCoverAssetID
	}
	return catalog.CampaignCoverSetV1, defaultCampaignCoverAssetID()
}

// CampaignAccessFormValue serializes campaign access for form rendering.
func CampaignAccessFormValue(access statev1.CampaignAccess) string {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return "manager"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return "owner"
	default:
		return "member"
	}
}

// CharacterKindFormValue serializes character kind for form rendering.
func CharacterKindFormValue(kind statev1.CharacterKind) string {
	if kind == statev1.CharacterKind_NPC {
		return "npc"
	}
	return "pc"
}

// CharacterKindLabel resolves character kind into localized display text.
func CharacterKindLabel(loc webtemplates.Localizer, kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return webtemplates.T(loc, "game.character_kind.pc")
	case statev1.CharacterKind_NPC:
		return webtemplates.T(loc, "game.character_kind.npc")
	default:
		return webtemplates.T(loc, "game.character_detail.kind_unspecified")
	}
}

// ParticipantControllerFormValue serializes character controller for form rendering.
func ParticipantControllerFormValue(controller statev1.Controller) string {
	if controller == statev1.Controller_CONTROLLER_AI {
		return "ai"
	}
	return "human"
}

// ParticipantRoleFormValue serializes participant role for form rendering.
func ParticipantRoleFormValue(role statev1.ParticipantRole) string {
	if role == statev1.ParticipantRole_GM {
		return "gm"
	}
	return "player"
}

// ParticipantControlFormLabel resolves the UI label for a participant control target.
func ParticipantControlFormLabel(participant *statev1.Participant) string {
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

// SessionStatusLabel resolves a localized session status label.
func SessionStatusLabel(loc webtemplates.Localizer, status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return webtemplates.T(loc, "game.session_status.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return webtemplates.T(loc, "game.session_status.ended")
	default:
		return webtemplates.T(loc, "game.session_status.unspecified")
	}
}
