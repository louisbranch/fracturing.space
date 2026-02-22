package web

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlePasskeyLoginStart begins a passkey authentication round trip and returns
// the credential challenge expected by browser/WebAuth clients.
func (h *handler) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}

	resp, err := h.authClient.BeginPasskeyLogin(r.Context(), &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_login")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": resp.GetSessionId(),
		"public_key": json.RawMessage(resp.GetCredentialRequestOptionsJson()),
	})
}

// handlePasskeyLoginFinish finalizes passkey authentication and hands control back
// to the consent flow via the shared pending transaction state.
func (h *handler) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	_, err := h.authClient.FinishPasskeyLogin(r.Context(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
		PendingId:              payload.PendingID,
	})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_login")
		return
	}

	redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, payload.PendingID)
	writeJSON(w, http.StatusOK, map[string]any{"redirect_url": redirectURL})
}

// handlePasskeyRegisterStart creates a new passkey credential request so users can
// onboard a new WebAuth identity without leaving the current auth flow.
func (h *handler) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		Email     string `json:"email"`
		PendingID string `json:"pending_id"`
		Locale    string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.Email) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.email_is_required")
		return
	}
	requestedLocale := strings.TrimSpace(payload.Locale)
	resolvedLocale := platformi18n.LocaleForTag(webi18n.Default())
	if requestedLocale == "" {
		tag, _ := webi18n.ResolveTag(r)
		resolvedLocale = platformi18n.LocaleForTag(tag)
	} else {
		parsedLocale, ok := platformi18n.ParseLocale(requestedLocale)
		if !ok {
			localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_locale")
			return
		}
		resolvedLocale = parsedLocale
	}

	createResp, err := h.authClient.CreateUser(r.Context(), &authv1.CreateUserRequest{
		Email:  payload.Email,
		Locale: resolvedLocale,
	})
	if err != nil || createResp.GetUser() == nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_create_user")
		return
	}

	beginResp, err := h.authClient.BeginPasskeyRegistration(r.Context(), &authv1.BeginPasskeyRegistrationRequest{UserId: createResp.GetUser().GetId()})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_registration")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": beginResp.GetSessionId(),
		"public_key": json.RawMessage(beginResp.GetCredentialCreationOptionsJson()),
		"user_id":    createResp.GetUser().GetId(),
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// handlePasskeyRegisterFinish completes the registration ceremony and returns the
// newly created participant binding identifiers for client continuation.
func (h *handler) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		UserID     string          `json:"user_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if strings.TrimSpace(payload.UserID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.user_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	_, err := h.authClient.FinishPasskeyRegistration(r.Context(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
	})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_registration")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":    payload.UserID,
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// handleMagicLink validates one-time login tokens and moves valid sessions into the
// normal consent redirect path.
func (h *handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	printer, lang := localizer(w, r)
	if h == nil || h.authClient == nil {
		h.renderMagicPage(w, r, http.StatusInternalServerError, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.unavailable.title"),
			Message: printer.Sprintf("magic.unavailable.message"),
			Detail:  printer.Sprintf("magic.unavailable.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		h.renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.missing.title"),
			Message: printer.Sprintf("magic.missing.message"),
			Detail:  printer.Sprintf("magic.missing.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	resp, err := h.authClient.ConsumeMagicLink(r.Context(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		h.renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.invalid.title"),
			Message: printer.Sprintf("magic.invalid.message"),
			Detail:  printer.Sprintf("magic.invalid.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}
	if pendingID := strings.TrimSpace(resp.GetPendingId()); pendingID != "" {
		redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, pendingID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	h.renderMagicPage(w, r, http.StatusOK, webtemplates.MagicParams{
		AppName:   h.resolvedAppName(),
		Title:     printer.Sprintf("magic.verified.title"),
		Message:   printer.Sprintf("magic.verified.message"),
		Detail:    printer.Sprintf("magic.verified.detail"),
		Loc:       printer,
		Success:   true,
		LinkURL:   "/",
		LinkLabel: printer.Sprintf("magic.verified.link"),
		Lang:      lang,
	})
}

// renderMagicPage writes the status code and renders the magic-link templ page.
func (h *handler) renderMagicPage(w http.ResponseWriter, r *http.Request, status int, params webtemplates.MagicParams) {
	if r != nil {
		if strings.TrimSpace(params.CurrentPath) == "" {
			params.CurrentPath = r.URL.Path
		}
		if strings.TrimSpace(params.CurrentQuery) == "" {
			params.CurrentQuery = r.URL.RawQuery
		}
	}
	writeGameContentType(w)
	w.WriteHeader(status)
	if err := h.writePage(
		w,
		r,
		webtemplates.MagicPage(params),
		composeHTMXTitleForPage(webtemplates.PageContext{
			Lang:    params.Lang,
			Loc:     params.Loc,
			AppName: h.resolvedAppName(),
		}, params.Title),
	); err != nil {
		log.Printf("web: failed to render magic page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// handleAuthLogin initiates the OAuth PKCE flow by redirecting to the auth server
// with state and challenge that ties browser and token exchange together.
func (h *handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_generate_pkce_verifier")
		return
	}
	challenge := computeS256Challenge(verifier)
	state := h.pendingFlows.create(verifier)

	authorizeURL := strings.TrimRight(strings.TrimSpace(h.config.AuthBaseURL), "/") + "/authorize"
	redirectURL, err := url.Parse(authorizeURL)
	if err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.invalid_auth_base_url")
		return
	}
	q := redirectURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", h.config.OAuthClientID)
	q.Set("redirect_uri", h.config.CallbackURL)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// handleAuthCallback exchanges the authorization code for a token and creates a
// web session that subsequent web handlers can reuse for campaign membership checks.
func (h *handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	if code == "" || state == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.missing_code_or_state")
		return
	}

	flow := h.pendingFlows.consume(state)
	if flow == nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_or_expired_state")
		return
	}

	tokenURL := strings.TrimSpace(h.config.AuthTokenURL)
	if tokenURL == "" {
		tokenURL = strings.TrimRight(strings.TrimSpace(h.config.AuthBaseURL), "/") + "/token"
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {h.config.CallbackURL},
		"code_verifier": {flow.codeVerifier},
		"client_id":     {h.config.OAuthClientID},
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.token_exchange_failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.token_exchange_returned", resp.Status)
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.failed_to_decode_token_response")
		return
	}

	if tokenResp.AccessToken == "" {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.empty_access_token")
		return
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	sessionID := h.sessions.create(tokenResp.AccessToken, "", expiry)
	setSessionCookie(w, sessionID)
	tokenCookieDomain := tokenCookieDomainForRequest(h.config.Domain, r.Host)
	setTokenCookie(w, tokenResp.AccessToken, tokenCookieDomain, int(tokenResp.ExpiresIn))
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleAuthLogout clears both local and cross-subdomain session/token artifacts to
// avoid mixed-session conditions across web and auth-aware siblings.
func (h *handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		h.sessions.delete(cookie.Value)
	}
	clearSessionCookie(w)
	tokenCookieDomain := tokenCookieDomainForRequest(h.config.Domain, r.Host)
	clearTokenCookie(w, tokenCookieDomain)
	if tokenCookieDomain != h.config.Domain {
		clearTokenCookie(w, h.config.Domain)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// writeJSON writes JSON responses with a consistent content type for auth flows.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(payload)
}
