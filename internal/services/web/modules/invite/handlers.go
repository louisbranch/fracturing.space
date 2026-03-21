package invite

import (
	"net/http"
	"net/url"
	"strings"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers groups public invite HTTP actions around one service dependency set.
type handlers struct {
	publichandler.Base
	service     inviteapp.Service
	requestMeta requestmeta.SchemePolicy
	sync        DashboardSync
}

// newHandlers assembles the invite transport surface from explicit dependencies.
func newHandlers(
	s inviteapp.Service,
	requestPrincipal principal.PrincipalResolver,
	policy requestmeta.SchemePolicy,
	sync DashboardSync,
) handlers {
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return handlers{
		Base:        publichandler.NewBaseFromPrincipal(requestPrincipal),
		service:     s,
		requestMeta: policy,
		sync:        sync,
	}
}

// withInviteID extracts the invite route parameter before delegating to handlers.
func (h handlers) withInviteID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return routeparam.WithRequired("inviteID", h.handleNotFound, fn)
}

// handleInvite renders the invite landing page for anonymous or signed-in viewers.
func (h handlers) handleInvite(w http.ResponseWriter, r *http.Request, inviteID string) {
	page, err := h.service.LoadInvite(r.Context(), h.userID(r), inviteID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderInvitePage(w, r, page)
}

// handleAccept claims an invite for the signed-in viewer and returns to the
// claimed campaign overview.
func (h handlers) handleAccept(w http.ResponseWriter, r *http.Request, inviteID string) {
	if !h.allowMutation(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	userID := h.userID(r)
	if strings.TrimSpace(userID) == "" {
		http.Redirect(w, r, loginRedirectForInvite(inviteID), http.StatusSeeOther)
		return
	}
	result, err := h.service.AcceptInvite(r.Context(), userID, inviteID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.sync.InviteChanged(r.Context(), result.UserIDs, result.CampaignID)
	flash.Write(w, r, flash.NoticeSuccess("web.invite.notice_joined_campaign"))
	http.Redirect(w, r, routepath.AppCampaign(result.CampaignID), http.StatusSeeOther)
}

// handleDecline declines a targeted invite for the signed-in viewer and returns to dashboard.
func (h handlers) handleDecline(w http.ResponseWriter, r *http.Request, inviteID string) {
	if !h.allowMutation(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	userID := h.userID(r)
	if strings.TrimSpace(userID) == "" {
		http.Redirect(w, r, loginRedirectForInvite(inviteID), http.StatusSeeOther)
		return
	}
	result, err := h.service.DeclineInvite(r.Context(), userID, inviteID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.sync.InviteChanged(r.Context(), result.UserIDs, result.CampaignID)
	http.Redirect(w, r, routepath.AppDashboard, http.StatusSeeOther)
}

// handleNotFound gives the public module a single invite-specific not-found response.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, "invite not found"))
}

// renderInvitePage applies the public shell layout around the invite landing view.
func (h handlers) renderInvitePage(w http.ResponseWriter, r *http.Request, page inviteapp.InvitePage) {
	loc, lang := h.PageLocalizer(w, r)
	h.WritePublicPage(
		w,
		r,
		page.Invite.CampaignName,
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		PublicInvitePage(mapPublicInviteView(page, loc), loc),
	)
}

// userID resolves the signed-in viewer when the invite transport is mounted in web.
func (h handlers) userID(r *http.Request) string {
	return h.RequestUserID(r)
}

// allowMutation applies the standard cookie-backed same-origin rule to the
// public invite mutation endpoints without blocking anonymous auth redirects.
func (h handlers) allowMutation(r *http.Request) bool {
	return sessioncookie.AllowsMutationWithPolicy(r, h.requestMeta)
}

// loginRedirectForInvite keeps invite URLs sticky across anonymous auth entrypoints.
func loginRedirectForInvite(inviteID string) string {
	values := url.Values{}
	values.Set("next", routepath.PublicInvite(inviteID))
	return routepath.Login + "?" + values.Encode()
}
