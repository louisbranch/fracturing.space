package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
)

func (s *Server) resolvePlayUserID(ctx context.Context, r *http.Request) (string, error) {
	sessionID, ok := readPlaySessionCookie(r)
	if !ok {
		return "", errors.New("play session cookie is required")
	}
	return s.resolvePlayUserIDFromSessionID(ctx, sessionID)
}

func (s *Server) resolvePlayUserIDFromSessionID(ctx context.Context, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("play session cookie is required")
	}
	resp, err := s.auth.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return "", fmt.Errorf("lookup play session: %w", err)
	}
	if resp == nil || resp.GetSession() == nil {
		return "", errors.New("play session not found")
	}
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", errors.New("play session returned empty user id")
	}
	return userID, nil
}

// resolveCampaignShellAccess decides whether the browser request can render the
// play shell immediately, must redirect to the cleaned launch URL, or should
// fall back to the web app.
func (s *Server) resolveCampaignShellAccess(
	w http.ResponseWriter,
	r *http.Request,
	campaignID string,
	launchGrantCfg playlaunchgrant.Config,
) (shellAccess, bool) {
	grant := strings.TrimSpace(r.URL.Query().Get("launch"))
	if grant != "" {
		return s.exchangeLaunchGrantForPlaySession(w, r, campaignID, launchGrantCfg)
	}
	if userID, ok := s.resolveExistingPlaySessionUser(w, r); ok {
		return shellAccess{UserID: userID}, false
	}
	return shellAccess{RedirectToWeb: true}, false
}

// exchangeLaunchGrantForPlaySession validates one launch grant and performs the
// one-way cutover into a play-scoped session cookie.
func (s *Server) exchangeLaunchGrantForPlaySession(
	w http.ResponseWriter,
	r *http.Request,
	campaignID string,
	launchGrantCfg playlaunchgrant.Config,
) (shellAccess, bool) {
	claims, err := playlaunchgrant.Validate(launchGrantCfg, strings.TrimSpace(r.URL.Query().Get("launch")))
	if err != nil || strings.TrimSpace(claims.CampaignID) != strings.TrimSpace(campaignID) {
		clearPlaySessionCookie(w, r, s.requestSchemePolicy)
		return shellAccess{RedirectToWeb: true}, false
	}
	resp, err := s.auth.CreateWebSession(r.Context(), &authv1.CreateWebSessionRequest{UserId: claims.UserID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		writeJSONError(w, http.StatusBadGateway, "failed to create play session")
		return shellAccess{}, true
	}
	writePlaySessionCookie(w, r, resp.GetSession().GetId(), s.requestSchemePolicy)
	http.Redirect(w, r, stripLaunchGrant(r), http.StatusSeeOther)
	return shellAccess{}, true
}

// resolveExistingPlaySessionUser verifies an existing play cookie and clears it
// eagerly when it no longer resolves to a usable auth session.
func (s *Server) resolveExistingPlaySessionUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	sessionID, ok := readPlaySessionCookie(r)
	if !ok {
		return "", false
	}
	userID, err := s.resolvePlayUserIDFromSessionID(r.Context(), sessionID)
	if err == nil {
		return userID, true
	}
	clearPlaySessionCookie(w, r, s.requestSchemePolicy)
	return "", false
}

func stripLaunchGrant(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}
	cloned := new(url.URL)
	*cloned = *r.URL
	query := cloned.Query()
	query.Del("launch")
	cloned.RawQuery = query.Encode()
	return cloned.String()
}
