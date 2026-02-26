package web

import (
	"context"
	"net/http"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	featureinvites "github.com/louisbranch/fracturing.space/internal/services/web/feature/invites"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) appInvitesRouteDependencies(w http.ResponseWriter, r *http.Request) featureinvites.AppInvitesHandlers {
	sess := sessionFromRequest(r, h.sessions)
	var listPendingInvites func(context.Context, *statev1.ListPendingInvitesForUserRequest) (*statev1.ListPendingInvitesForUserResponse, error)
	if h.inviteClient != nil {
		listPendingInvites = func(ctx context.Context, req *statev1.ListPendingInvitesForUserRequest) (*statev1.ListPendingInvitesForUserResponse, error) {
			return h.inviteClient.ListPendingInvitesForUser(ctx, req)
		}
	}
	var issueJoinGrant func(context.Context, *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error)
	if h.authClient != nil {
		issueJoinGrant = func(ctx context.Context, req *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error) {
			return h.authClient.IssueJoinGrant(ctx, req)
		}
	}
	var claimInvite func(context.Context, *statev1.ClaimInviteRequest) error
	if h.inviteClient != nil {
		claimInvite = func(ctx context.Context, req *statev1.ClaimInviteRequest) error {
			_, err := h.inviteClient.ClaimInvite(ctx, req)
			return err
		}
	}
	return featureinvites.AppInvitesHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		HasInviteClient: func() bool {
			return h.inviteClient != nil
		},
		HasAuthClient: func() bool {
			return h.authClient != nil
		},
		ResolveProfileUserID: func(ctx context.Context) (string, error) {
			return h.resolveProfileUserID(ctx, sess)
		},
		ListPendingInvites: listPendingInvites,
		IssueJoinGrant:     issueJoinGrant,
		ClaimInvite:        claimInvite,
		RenderErrorPage:    h.renderErrorPage,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
	}
}

func (h *handler) appInviteClaimRouteDependencies(w http.ResponseWriter, r *http.Request) featureinvites.AppInvitesHandlers {
	sess := sessionFromRequest(r, h.sessions)
	var issueJoinGrant func(context.Context, *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error)
	if h.authClient != nil {
		issueJoinGrant = func(ctx context.Context, req *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error) {
			return h.authClient.IssueJoinGrant(ctx, req)
		}
	}
	var claimInvite func(context.Context, *statev1.ClaimInviteRequest) error
	if h.inviteClient != nil {
		claimInvite = func(ctx context.Context, req *statev1.ClaimInviteRequest) error {
			_, err := h.inviteClient.ClaimInvite(ctx, req)
			return err
		}
	}
	return featureinvites.AppInvitesHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		HasInviteClient: func() bool {
			return h.inviteClient != nil
		},
		HasAuthClient: func() bool {
			return h.authClient != nil
		},
		ResolveProfileUserID: func(ctx context.Context) (string, error) {
			return h.resolveProfileUserID(ctx, sess)
		},
		IssueJoinGrant:  issueJoinGrant,
		ClaimInvite:     claimInvite,
		RenderErrorPage: h.renderErrorPage,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
	}
}
