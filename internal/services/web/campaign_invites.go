package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errRecipientUsernameRequired = errors.New("recipient username is required")
	errConnectionsUnavailable    = errors.New("connections service is not configured")
	errRecipientUsernameFormat   = errors.New("recipient username must start with @")
)

func (h *handler) handleAppCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInvites lists invites for a campaign and relies on
	// game-service policy to enforce manager/owner-only visibility.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := h.campaignReadContext(w, r, "Invites unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}

	var invites []*statev1.Invite
	if cachedInvites, ok := h.cachedCampaignInvites(readCtx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := h.inviteClient.ListInvites(readCtx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		h.setCampaignInvitesCache(readCtx, campaignID, userID, invites)
	}

	contactOptions := h.listInviteContactOptions(readCtx, campaignID, userID, invites)
	renderAppCampaignInvitesPageWithContextAndContacts(
		w,
		readReq,
		h.pageContextForCampaign(w, readReq, campaignID),
		campaignID,
		invites,
		contactOptions,
		true,
	)
}

func (h *handler) handleAppCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteCreate creates a player invitation and binds it to
	// the selected target participant.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	inviteActor := h.campaignInviteActorFromParticipant(actor)
	if inviteActor == nil || !inviteActor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite create form")
		return
	}
	targetParticipantID := strings.TrimSpace(r.FormValue("participant_id"))
	if targetParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "participant id is required")
		return
	}
	lookupCtx := grpcauthctx.WithUserID(r.Context(), strings.TrimSpace(actor.GetUserId()))
	if strings.EqualFold(strings.TrimSpace(r.FormValue("action")), "verify") {
		verification, err := h.lookupInviteRecipientVerification(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
		if err != nil {
			h.renderInviteRecipientLookupError(w, r, err)
			return
		}
		h.renderCampaignInvitesVerificationPage(w, r, campaignID, strings.TrimSpace(actor.GetUserId()), inviteActor.canManageInvites, verification)
		return
	}
	recipientUserID, err := h.resolveInviteRecipientUserID(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
	if err != nil {
		h.renderInviteRecipientLookupError(w, r, err)
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	_, err = h.inviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   targetParticipantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to create invite")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/invites", http.StatusFound)
}

func (h *handler) renderInviteRecipientLookupError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errRecipientUsernameRequired):
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is required")
	case errors.Is(err, errRecipientUsernameFormat):
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username must start with @")
	case errors.Is(err, errConnectionsUnavailable):
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "connections service is not configured")
	case status.Code(err) == codes.InvalidArgument:
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is invalid")
	case status.Code(err) == codes.NotFound:
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username was not found")
	default:
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to resolve invite recipient")
	}
}

func (h *handler) lookupInviteRecipientVerification(ctx context.Context, recipientUserID string) (webtemplates.CampaignInviteVerification, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameRequired
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameFormat
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameRequired
	}
	if h == nil || h.connectionsClient == nil {
		return webtemplates.CampaignInviteVerification{}, errConnectionsUnavailable
	}
	resp, err := h.connectionsClient.LookupPublicProfile(ctx, &connectionsv1.LookupPublicProfileRequest{
		Username: username,
	})
	if err != nil {
		return webtemplates.CampaignInviteVerification{}, err
	}
	usernameRecord := resp.GetUsernameRecord()
	if usernameRecord == nil {
		return webtemplates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(usernameRecord.GetUserId())
	if resolvedUserID == "" {
		return webtemplates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}
	verification := webtemplates.CampaignInviteVerification{
		HasResult: true,
		Username:  strings.TrimSpace(usernameRecord.GetUsername()),
		UserID:    resolvedUserID,
	}
	if verification.Username == "" {
		verification.Username = username
	}
	if profile := resp.GetPublicProfileRecord(); profile != nil {
		verification.Name = strings.TrimSpace(profile.GetName())
		verification.AvatarSetID = strings.TrimSpace(profile.GetAvatarSetId())
		verification.AvatarAssetID = strings.TrimSpace(profile.GetAvatarAssetId())
		verification.Bio = strings.TrimSpace(profile.GetBio())
	}
	return verification, nil
}

func (h *handler) resolveInviteRecipientUserID(ctx context.Context, recipientUserID string) (string, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return "", nil
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return recipientUserID, nil
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return "", errRecipientUsernameRequired
	}
	if h == nil || h.connectionsClient == nil {
		return "", errConnectionsUnavailable
	}
	resp, err := h.connectionsClient.LookupUsername(ctx, &connectionsv1.LookupUsernameRequest{
		Username: username,
	})
	if err != nil {
		return "", err
	}
	record := resp.GetUsernameRecord()
	if record == nil {
		return "", status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(record.GetUserId())
	if resolvedUserID == "" {
		return "", status.Error(codes.NotFound, "username not found")
	}
	return resolvedUserID, nil
}

func (h *handler) handleAppCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteRevoke removes an invite resource to terminate a
	// pending membership path for the campaign.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	inviteActor := h.campaignInviteActorFromParticipant(actor)
	if inviteActor == nil || !inviteActor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite revoke form")
		return
	}
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	if inviteID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "invite id is required")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	_, err := h.inviteClient.RevokeInvite(ctx, &statev1.RevokeInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to revoke invite")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/invites", http.StatusFound)
}

type campaignInviteActor struct {
	participantID    string
	canManageInvites bool
}

func (h *handler) campaignInviteActorFromParticipant(participant *statev1.Participant) *campaignInviteActor {
	if participant == nil {
		return nil
	}
	participantID := strings.TrimSpace(participant.GetId())
	if participantID == "" {
		return nil
	}
	return &campaignInviteActor{
		participantID:    participantID,
		canManageInvites: canManageCampaignInvites(participant.GetCampaignAccess()),
	}
}

func (h *handler) campaignParticipant(ctx context.Context, campaignID string, sess *session) (*statev1.Participant, error) {
	// campaignParticipant maps an access token to the participant record in the
	// campaign, with pagination across participant pages if needed.
	if h == nil || h.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	userID, err := h.sessionUserIDForSession(ctx, sess)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	return h.campaignParticipantByUserID(grpcauthctx.WithUserID(ctx, userID), campaignID, userID)
}

func (h *handler) campaignParticipantByUserID(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
	if h == nil || h.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, nil
	}

	pageToken := ""
	for {
		resp, err := h.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	return nil, nil
}

func canManageCampaignInvites(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func renderAppCampaignInvitesPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContacts(w, r, page, campaignID, invites, nil, canManageInvites)
}

func renderAppCampaignInvitesPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContacts(w, r, page, campaignID, invites, nil, canManageInvites)
}

func renderAppCampaignInvitesPageWithContextAndContacts(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool) {
	renderAppCampaignInvitesPageWithContextAndContactsAndVerification(
		w,
		r,
		page,
		campaignID,
		invites,
		contacts,
		canManageInvites,
		webtemplates.CampaignInviteVerification{},
	)
}

func renderAppCampaignInvitesPageWithContextAndContactsAndVerification(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
	// renderAppCampaignInvitesPage exposes write controls only to managed roles.
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
	if err := writePage(w, r, webtemplates.CampaignInvitesPage(page, campaignID, canManageInvites, inviteItems, contacts, verification), composeHTMXTitleForPage(page, "game.campaign_invites.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_campaign_invites_page")
	}
}

func (h *handler) renderCampaignInvitesVerificationPage(w http.ResponseWriter, r *http.Request, campaignID string, userID string, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
	if h == nil || h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}
	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	var invites []*statev1.Invite
	if cachedInvites, ok := h.cachedCampaignInvites(ctx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := h.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		h.setCampaignInvitesCache(ctx, campaignID, userID, invites)
	}
	contactOptions := h.listInviteContactOptions(ctx, campaignID, userID, invites)
	renderReq := r.WithContext(ctx)
	renderAppCampaignInvitesPageWithContextAndContactsAndVerification(
		w,
		renderReq,
		h.pageContextForCampaign(w, renderReq, campaignID),
		campaignID,
		invites,
		contactOptions,
		canManageInvites,
		verification,
	)
}

func (h *handler) listInviteContactOptions(ctx context.Context, campaignID string, ownerUserID string, invites []*statev1.Invite) []webtemplates.CampaignInviteContactOption {
	if h == nil || h.connectionsClient == nil || h.participantClient == nil {
		return nil
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil
	}

	contacts, err := h.listAllContacts(ctx, ownerUserID)
	if err != nil {
		log.Printf("web: list invite contacts failed: %v", err)
		return nil
	}
	if len(contacts) == 0 {
		return nil
	}

	participants, err := h.listAllCampaignParticipants(ctx, campaignID)
	if err != nil {
		log.Printf("web: list campaign participants for contact options failed: %v", err)
		return nil
	}
	return buildInviteContactOptions(contacts, participants, invites)
}

func (h *handler) listAllContacts(ctx context.Context, ownerUserID string) ([]*connectionsv1.Contact, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, nil
	}
	pageToken := ""
	seenTokens := make(map[string]struct{})
	contacts := make([]*connectionsv1.Contact, 0)
	for {
		resp, err := h.connectionsClient.ListContacts(ctx, &connectionsv1.ListContactsRequest{
			OwnerUserId: ownerUserID,
			PageSize:    50,
			PageToken:   pageToken,
		})
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, resp.GetContacts()...)
		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		if _, ok := seenTokens[nextToken]; ok {
			return nil, fmt.Errorf("list contacts: repeated page token %q", nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}
	return contacts, nil
}

func (h *handler) listAllCampaignParticipants(ctx context.Context, campaignID string) ([]*statev1.Participant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, nil
	}
	if cached, ok := h.cachedCampaignParticipants(ctx, campaignID); ok {
		return cached, nil
	}
	pageToken := ""
	seenTokens := make(map[string]struct{})
	participants := make([]*statev1.Participant, 0)
	for {
		resp, err := h.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		participants = append(participants, resp.GetParticipants()...)
		nextToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextToken == "" {
			break
		}
		if _, ok := seenTokens[nextToken]; ok {
			return nil, fmt.Errorf("list participants: repeated page token %q", nextToken)
		}
		seenTokens[nextToken] = struct{}{}
		pageToken = nextToken
	}
	h.setCampaignParticipantsCache(ctx, campaignID, participants)
	return participants, nil
}

func buildInviteContactOptions(contacts []*connectionsv1.Contact, participants []*statev1.Participant, invites []*statev1.Invite) []webtemplates.CampaignInviteContactOption {
	participantUsers := make(map[string]struct{})
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		userID := strings.TrimSpace(participant.GetUserId())
		if userID == "" {
			continue
		}
		participantUsers[userID] = struct{}{}
	}

	pendingInviteRecipients := make(map[string]struct{})
	for _, invite := range invites {
		if invite == nil || invite.GetStatus() != statev1.InviteStatus_PENDING {
			continue
		}
		recipientUserID := strings.TrimSpace(invite.GetRecipientUserId())
		if recipientUserID == "" {
			continue
		}
		pendingInviteRecipients[recipientUserID] = struct{}{}
	}

	options := make([]webtemplates.CampaignInviteContactOption, 0, len(contacts))
	seen := make(map[string]struct{})
	for _, contact := range contacts {
		if contact == nil {
			continue
		}
		userID := strings.TrimSpace(contact.GetContactUserId())
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		if _, ok := participantUsers[userID]; ok {
			continue
		}
		if _, ok := pendingInviteRecipients[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		options = append(options, webtemplates.CampaignInviteContactOption{
			UserID: userID,
			Label:  userID,
		})
	}
	sort.Slice(options, func(i int, j int) bool {
		return options[i].UserID < options[j].UserID
	})
	return options
}
