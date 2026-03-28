package sessions

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playorigin"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// ServiceConfig groups session lifecycle app config.
type ServiceConfig struct {
	Characters    campaignapp.CharacterReadServiceConfig
	Mutation      campaignapp.SessionMutationServiceConfig
	Participants  campaignapp.ParticipantReadServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// HandlerServices groups session lifecycle behavior.
type HandlerServices struct {
	characters   campaignapp.CampaignCharacterReadService
	mutation     campaignapp.CampaignSessionMutationService
	participants campaignapp.CampaignParticipantReadService
}

// NewHandlerServices keeps session transport dependencies owned by the session
// surface instead of the campaigns root constructor.
func NewHandlerServices(config ServiceConfig) (HandlerServices, error) {
	characters, err := campaignapp.NewCharacterReadService(config.Characters, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("session-character-reads: %w", err)
	}
	mutation, err := campaignapp.NewSessionMutationService(config.Mutation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("session-mutation: %w", err)
	}
	participants, err := campaignapp.NewParticipantReadService(config.Participants, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("session-participant-reads: %w", err)
	}
	return HandlerServices{
		characters:   characters,
		mutation:     mutation,
		participants: participants,
	}, nil
}

// Handler owns session detail/lifecycle plus play-launch routes.
type Handler struct {
	campaigndetail.Handler
	sessions         HandlerServices
	playFallbackPort string
	playLaunchGrant  playlaunchgrant.Config
}

// NewHandler assembles the session route-owner handler.
func NewHandler(detail campaigndetail.Handler, services HandlerServices, playFallbackPort string, playLaunchGrant playlaunchgrant.Config) Handler {
	return Handler{
		Handler:          detail,
		sessions:         services,
		playFallbackPort: playFallbackPort,
		playLaunchGrant:  playLaunchGrant,
	}
}

// HandleSessions renders the sessions detail page.
func (h Handler) HandleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	_, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := sessionsView(page, campaignID)
	h.WriteCampaignDetailPage(w, r, page, campaignID, campaignrender.SessionsFragment(view, page.Loc), sessionsBreadcrumbs(page)...)
}

// HandleSessionCreatePage renders the session create page.
func (h Handler) HandleSessionCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.Pages.Authorization.RequireManageSession(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	readiness, err := h.Pages.SessionReads.CampaignSessionReadiness(ctx, campaignID, page.Locale)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	readContext := campaignapp.CharacterReadContext{
		System:       page.Workspace.System,
		Locale:       page.Locale,
		ViewerUserID: h.RequestUserID(r),
	}
	characters, err := h.sessions.characters.CampaignCharacters(ctx, campaignID, readContext)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	participants, err := h.sessions.participants.CampaignParticipants(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := sessionCreateView(page, campaignID, readiness, characters, participants)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.SessionCreateFragment(view, page.Loc),
		sessionCreateBreadcrumbs(page, campaignID)...,
	)
}

// HandleSessionCreate starts a campaign session from the dedicated create page.
func (h Handler) HandleSessionCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	redirectURL := routepath.AppCampaignSessionCreate(campaignID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_session_start_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.sessions.mutation.StartSession(ctx, campaignID, parseStartSessionInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_start_session", redirectURL)
		return
	}
	h.Sync().SessionStarted(ctx, userID, campaignID)
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_session_started", routepath.AppCampaignSessions(campaignID))
}

// HandleSessionDetail renders the selected session page.
func (h Handler) HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	_, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if !campaignHasSession(page.Sessions, sessionID) {
		h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, "session not found"))
		return
	}
	view := sessionDetailView(page, campaignID, sessionID)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.SessionDetailFragment(view, page.Loc),
		sessionDetailBreadcrumbs(page, campaignID, view)...,
	)
}

// HandleSessionEnd ends the active session.
func (h Handler) HandleSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_session_end_form", routepath.AppCampaignSessions(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.sessions.mutation.EndSession(ctx, campaignID, parseEndSessionInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_end_session", routepath.AppCampaignSessions(campaignID))
		return
	}
	h.Sync().SessionEnded(ctx, userID, campaignID)
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_session_ended", routepath.AppCampaignSessions(campaignID))
}

// HandleGame redirects the campaign game route into the dedicated play
// surface.
func (h Handler) HandleGame(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, err := h.LoadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	userID := strings.TrimSpace(h.RequestUserID(r))
	if userID == "" {
		h.WriteError(w, r, http.ErrNoCookie)
		return
	}
	grant, _, err := playlaunchgrant.Issue(h.playLaunchGrant, playlaunchgrant.IssueInput{
		GrantID:    strconv.FormatInt(h.Now().UnixNano(), 10),
		CampaignID: strings.TrimSpace(campaignID),
		UserID:     userID,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	_ = ctx
	_ = page
	target := playorigin.PlayURL(
		r,
		h.RequestMeta(),
		h.playFallbackPort,
		"/campaigns/"+url.PathEscape(campaignID)+"?launch="+url.QueryEscape(grant),
	)
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, target)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// campaignHasSession reports whether the selected session exists in page state.
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
