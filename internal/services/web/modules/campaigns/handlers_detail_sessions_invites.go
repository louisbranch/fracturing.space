package campaigns

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// sessionHandlerServices groups session lifecycle behavior.
type sessionHandlerServices struct {
	mutation campaignapp.CampaignSessionMutationService
}

// inviteHandlerServices groups invite reads, mutations, and recipient lookup
// behavior.
type inviteHandlerServices struct {
	reads            campaignapp.CampaignInviteReadService
	mutation         campaignapp.CampaignInviteMutationService
	participantReads campaignapp.CampaignParticipantReadService
}

// sessionHandlers owns session detail/lifecycle plus play-launch routes.
type sessionHandlers struct {
	campaignDetailHandlers
	sessions         sessionHandlerServices
	playFallbackPort string
	playLaunchGrant  playlaunchgrant.Config
}

// inviteHandlers owns invite read, search, and mutation routes.
type inviteHandlers struct {
	campaignDetailHandlers
	invites inviteHandlerServices
}

// newSessionHandlerServices keeps session transport dependencies owned by the
// session surface instead of the root constructor.
func newSessionHandlerServices(config sessionServiceConfig, authorization campaignapp.AuthorizationGateway) sessionHandlerServices {
	return sessionHandlerServices{
		mutation: campaignapp.NewSessionMutationService(config.Mutation, authorization),
	}
}

// newInviteHandlerServices keeps invite transport dependencies owned by the
// invite surface instead of the root constructor.
func newInviteHandlerServices(config inviteServiceConfig) inviteHandlerServices {
	return inviteHandlerServices{
		reads:            campaignapp.NewInviteReadService(config.Read, config.Authorization),
		mutation:         campaignapp.NewInviteMutationService(config.Mutation, config.Authorization),
		participantReads: campaignapp.NewParticipantReadService(config.ParticipantRead, config.Authorization),
	}
}

// newSessionHandlers assembles the session route-owner handler.
func newSessionHandlers(detail campaignDetailHandlers, services sessionHandlerServices, playFallbackPort string, playLaunchGrant playlaunchgrant.Config) sessionHandlers {
	return sessionHandlers{
		campaignDetailHandlers: detail,
		sessions:               services,
		playFallbackPort:       playFallbackPort,
		playLaunchGrant:        playLaunchGrant,
	}
}

// newInviteHandlers assembles the invite route-owner handler.
func newInviteHandlers(detail campaignDetailHandlers, services inviteHandlerServices) inviteHandlers {
	return inviteHandlers{
		campaignDetailHandlers: detail,
		invites:                services,
	}
}

// missingSessionHandlerServices reports which session controls are absent
// before session mutation routes are mounted.
func missingSessionHandlerServices(services sessionHandlerServices) []string {
	if services.mutation == nil {
		return []string{"session-mutation"}
	}
	return nil
}

// missingInviteHandlerServices reports which invite capabilities are absent
// before invite routes are mounted.
func missingInviteHandlerServices(services inviteHandlerServices) []string {
	missing := []string{}
	if services.reads == nil {
		missing = append(missing, "invite-reads")
	}
	if services.mutation == nil {
		missing = append(missing, "invite-mutation")
	}
	if services.participantReads == nil {
		missing = append(missing, "invite-participant-reads")
	}
	return missing
}

// handleSessions handles this route in the module transport layer.
func (h sessionHandlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	_, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.sessionsView(campaignID)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.SessionsFragment(view, page.loc), page.sessionsBreadcrumbs()...)
}

// handleSessionCreatePage handles this route in the module transport layer.
func (h sessionHandlers) handleSessionCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.pages.authorization.RequireManageSession(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	readiness, err := h.pages.sessionReads.CampaignSessionReadiness(ctx, campaignID, page.locale)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.sessionCreateView(campaignID, readiness)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.SessionCreateFragment(view, page.loc),
		page.sessionCreateBreadcrumbs(campaignID)...,
	)
}

// handleSessionDetail handles this route in the module transport layer.
func (h sessionHandlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	_, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if !campaignHasSession(page.sessions, sessionID) {
		h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, "session not found"))
		return
	}
	view := page.sessionDetailView(campaignID, sessionID)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.SessionDetailFragment(view, page.loc),
		page.sessionDetailBreadcrumbs(campaignID, view)...,
	)
}

// handleInvites handles this route in the module transport layer.
func (h inviteHandlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.invites.reads.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	var participants []campaignapp.CampaignParticipant
	if page.canManageInvites {
		var err error
		participants, err = h.invites.participantReads.CampaignParticipants(ctx, campaignID)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
	}
	view := page.invitesView(campaignID, participants, items, r)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.InvitesFragment(view, page.loc), page.invitesBreadcrumbs()...)
}

// campaignHasSession ensures detail routes render only stored campaign sessions,
// never synthetic labels derived from route parameters.
func campaignHasSession(sessions []campaignapp.CampaignSession, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	for _, session := range sessions {
		if strings.TrimSpace(session.ID) == sessionID {
			return true
		}
	}
	return false
}
