package admin

import (
	"log"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

func (h *Handler) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UsersPageView{}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	if userID := strings.TrimSpace(r.URL.Query().Get("user_id")); userID != "" {
		h.redirectToUserDetail(w, r, userID)
		return
	}

	renderPage(
		w,
		r,
		templates.UsersPage(view, loc),
		templates.UsersFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.users", templates.AppName()),
	)
}

// handleUserLookup redirects the get-user form to the detail page.
func (h *Handler) handleUserLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UsersPageView{}

	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		view.Message = loc.Sprintf("error.user_id_required")
		renderPage(
			w,
			r,
			templates.UsersPage(view, loc),
			templates.UsersFullPage(view, pageCtx),
			htmxLocalizedPageTitle(loc, "title.users", templates.AppName()),
		)
		return
	}

	h.redirectToUserDetail(w, r, userID)
}

// handleUserDetail renders the single-user detail page.
func (h *Handler) handleUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleUserDetailTab(w, r, userID, "details")
}

// handleUserInvites renders the user pending invites tab.
func (h *Handler) handleUserInvites(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleUserDetailTab(w, r, userID, "invites")
}

func (h *Handler) handleUserDetailTab(w http.ResponseWriter, r *http.Request, userID, tab string) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	detail, message := h.loadUserDetail(ctx, userID, loc)
	view.Detail = detail
	if message != "" && view.Message == "" {
		view.Message = message
	}

	h.populateUserInvites(ctx, view.Detail, loc)

	h.renderUserDetail(w, r, view, pageCtx, loc, tab)
}

// handleMagicLink generates a magic link for a user email.
func (h *Handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{}
	if !requireSameOrigin(w, r, loc) {
		return
	}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.magic_link_invalid")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	email := strings.TrimSpace(r.FormValue("email"))
	if userID == "" || email == "" {
		view.Message = loc.Sprintf("error.magic_link_invalid")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	client := h.authClient()
	if client == nil {
		view.Message = loc.Sprintf("error.user_service_unavailable")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()
	response, err := client.GenerateMagicLink(ctx, &authv1.GenerateMagicLinkRequest{
		UserId: userID,
		Email:  email,
	})
	if err != nil || response.GetMagicLinkUrl() == "" {
		log.Printf("generate magic link: %v", err)
		view.Message = loc.Sprintf("error.magic_link_failed")
		view.Detail, _ = h.loadUserDetail(ctx, userID, loc)
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	detail, message := h.loadUserDetail(ctx, userID, loc)
	view.Detail = detail
	if message != "" && view.Message == "" {
		view.Message = message
	}
	view.MagicLinkURL = response.GetMagicLinkUrl()
	view.MagicLinkEmail = email
	if response.GetExpiresAt() != nil {
		view.MagicLinkExpiresAt = formatTimestamp(response.GetExpiresAt())
	}

	view.Message = loc.Sprintf("users.magic.success")
	h.renderUserDetail(w, r, view, pageCtx, loc, "details")
}

// handleUsersTable renders the users table via HTMX.
func (h *Handler) handleUsersTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	client := h.authClient()
	if client == nil {
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.user_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := client.ListUsers(ctx, &authv1.ListUsersRequest{PageSize: 50})
	if err != nil {
		log.Printf("list users: %v", err)
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.users_unavailable"), loc)
		return
	}

	users := response.GetUsers()
	if len(users) == 0 {
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.no_users"), loc)
		return
	}

	rows := buildUserRows(users)
	h.renderUsersTable(w, r, rows, "", loc)
}
