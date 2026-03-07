package users

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

const (
	// inviteListPageSize caps the number of invites shown per page.
	inviteListPageSize = 50
)

// handlers implements the users Handlers contract.
type handlers struct {
	base         modulehandler.Base
	authClient   authv1.AuthServiceClient
	inviteClient statev1.InviteServiceClient
}

// NewHandlers returns the users handler implementation.
func NewHandlers(base modulehandler.Base, authClient authv1.AuthServiceClient, inviteClient statev1.InviteServiceClient) Handlers {
	return &handlers{base: base, authClient: authClient, inviteClient: inviteClient}
}

// HandleUsersPage renders the users page shell.
func (s *handlers) HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	view := templates.UsersPageView{}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	if userID := strings.TrimSpace(r.URL.Query().Get("user_id")); userID != "" {
		s.redirectToUserDetail(w, r, userID)
		return
	}

	s.base.RenderPage(
		w,
		r,
		templates.UsersPage(view, loc),
		templates.UsersFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.users", templates.AppName()),
	)
}

// HandleUsersTable renders the users table via HTMX.
func (s *handlers) HandleUsersTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{PageSize: 50})
	if err != nil {
		adminerrors.LogError(r, "list users: %v", err)
		s.renderUsersTable(w, r, nil, loc.Sprintf("error.users_unavailable"), loc)
		return
	}

	users := response.GetUsers()
	if len(users) == 0 {
		s.renderUsersTable(w, r, nil, loc.Sprintf("error.no_users"), loc)
		return
	}

	rows := buildUserRows(users)
	s.renderUsersTable(w, r, rows, "", loc)
}

// HandleUserLookup redirects the lookup form to a concrete user detail route.
func (s *handlers) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	view := templates.UsersPageView{}

	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		view.Message = loc.Sprintf("error.user_id_required")
		s.base.RenderPage(
			w,
			r,
			templates.UsersPage(view, loc),
			templates.UsersFullPage(view, pageCtx),
			s.base.HTMXLocalizedPageTitle(loc, "title.users", templates.AppName()),
		)
		return
	}

	s.redirectToUserDetail(w, r, userID)
}

// HandleUserDetail renders the user detail tab.
func (s *handlers) HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	s.handleUserDetailTab(w, r, userID, "details")
}

// HandleUserInvites renders the pending invites tab for a user.
func (s *handlers) HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string) {
	s.handleUserDetailTab(w, r, userID, "invites")
}

func (s *handlers) handleUserDetailTab(w http.ResponseWriter, r *http.Request, userID, tab string) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	view := templates.UserDetailPageView{}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	detail, loadMessage := s.loadUserDetail(r, ctx, userID, loc)
	view.Detail = detail
	if loadMessage != "" && view.Message == "" {
		view.Message = loadMessage
	}

	s.populateUserInvites(r, ctx, view.Detail, loc)
	s.renderUserDetail(w, r, view, pageCtx, loc, tab)
}

func (s *handlers) renderUserDetail(w http.ResponseWriter, r *http.Request, view templates.UserDetailPageView, pageCtx templates.PageContext, loc *message.Printer, activePage string) {
	s.base.RenderPage(
		w,
		r,
		templates.UserDetailPage(view, activePage, loc),
		templates.UserDetailFullPage(view, activePage, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.user", templates.AppName()),
	)
}

func (s *handlers) redirectToUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		http.NotFound(w, r)
		return
	}
	redirectURL := routepath.UserDetail(userID)
	if s.base.IsHTMXRequest(r) {
		w.Header().Set("Location", redirectURL)
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *handlers) loadUserDetail(r *http.Request, ctx context.Context, userID string, loc *message.Printer) (*templates.UserDetail, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("error.user_id_required")
	}
	response, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || response.GetUser() == nil {
		adminerrors.LogError(r, "get user: %v", err)
		return nil, loc.Sprintf("error.user_not_found")
	}
	detail := buildUserDetail(response.GetUser())
	if detail != nil {
		emails, err := s.authClient.ListUserEmails(ctx, &authv1.ListUserEmailsRequest{UserId: userID})
		if err != nil {
			adminerrors.LogError(r, "list user emails: %v", err)
		} else {
			detail.Emails = buildUserEmailRows(emails.GetEmails(), loc)
		}
	}
	return detail, ""
}

func (s *handlers) populateUserInvites(r *http.Request, ctx context.Context, detail *templates.UserDetail, loc *message.Printer) {
	if detail == nil {
		return
	}
	rows, message := s.listPendingInvitesForUser(r, ctx, detail.ID, loc)
	detail.PendingInvites = rows
	detail.PendingInvitesMessage = message
}

func (s *handlers) listPendingInvitesForUser(r *http.Request, ctx context.Context, userID string, loc *message.Printer) ([]templates.InviteRow, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("users.invites.empty")
	}
	rows := make([]templates.InviteRow, 0)
	pageToken := ""
	for {
		resp, err := s.inviteClient.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{
			PageSize:  inviteListPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			adminerrors.LogError(r, "list pending invites for user: %v", err)
			return nil, loc.Sprintf("error.pending_invites_unavailable")
		}

		for _, pending := range resp.GetInvites() {
			if pending == nil {
				continue
			}
			inv := pending.GetInvite()
			campaign := pending.GetCampaign()
			participant := pending.GetParticipant()

			campaignID := strings.TrimSpace(campaign.GetId())
			if campaignID == "" && inv != nil {
				campaignID = strings.TrimSpace(inv.GetCampaignId())
			}
			campaignName := strings.TrimSpace(campaign.GetName())
			if campaignName == "" {
				if campaignID != "" {
					campaignName = campaignID
				} else {
					campaignName = loc.Sprintf("label.unknown")
				}
			}

			participantLabel := strings.TrimSpace(participant.GetName())
			if participantLabel == "" {
				participantLabel = loc.Sprintf("label.unknown")
			}

			inviteID := ""
			status := statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED
			createdAt := ""
			if inv != nil {
				inviteID = inv.GetId()
				status = inv.GetStatus()
				createdAt = eventview.FormatTimestamp(inv.GetCreatedAt())
			}
			statusLabel, statusVariant := formatInviteStatus(status, loc)

			rows = append(rows, templates.InviteRow{
				ID:            inviteID,
				CampaignID:    campaignID,
				CampaignName:  campaignName,
				Participant:   participantLabel,
				Status:        statusLabel,
				StatusVariant: statusVariant,
				CreatedAt:     createdAt,
			})
		}

		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	if len(rows) == 0 {
		return nil, loc.Sprintf("users.invites.empty")
	}
	return rows, ""
}

func (s *handlers) renderUsersTable(w http.ResponseWriter, r *http.Request, rows []templates.UserRow, message string, loc *message.Printer) {
	templ.Handler(templates.UsersTable(rows, message, loc)).ServeHTTP(w, r)
}
