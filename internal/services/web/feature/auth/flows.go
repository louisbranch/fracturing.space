package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webcache "github.com/louisbranch/fracturing.space/internal/services/web/infra/cache"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

// AuthFlowDependencies captures the boundary seams for authentication flow handlers.
type AuthFlowDependencies struct {
	AuthClient func() authv1.AuthServiceClient

	AuthBaseURL   string
	AuthTokenURL  string
	OAuthClientID string
	CallbackURL   string
	ConfigDomain  string
	AuthLoginPath string

	ResolvedAppName     func() string
	BuildAuthConsentURL func(authBaseURL, pendingID string) string

	CreatePendingState  func(verifier string) string
	ConsumePendingState func(state string) (string, bool)
	CreateSession       func(accessToken, userDisplayName string, expiry time.Time) string
	DeleteSession       func(sessionID string)

	SessionIDFromRequest        func(r *http.Request) string
	TokenCookieDomainForRequest func(requestHost string) string

	GenerateCodeVerifier func() (string, error)
	ComputeS256Challenge func(verifier string) string

	Localizer        func(http.ResponseWriter, *http.Request) (*message.Printer, string)
	LocalizeError    func(http.ResponseWriter, *http.Request, int, string, ...any)
	RenderMagicPage  func(http.ResponseWriter, *http.Request, int, webtemplates.MagicParams)
	WriteJSON        func(http.ResponseWriter, int, any)
	WritePage        func(http.ResponseWriter, *http.Request, templ.Component, string) error
	ComposeHTMXTitle func(*message.Printer, string, ...any) string

	WriteSessionCookie func(http.ResponseWriter, string)
	WriteTokenCookie   func(http.ResponseWriter, string, string, int)
	ClearSessionCookie func(http.ResponseWriter)
	ClearTokenCookie   func(http.ResponseWriter, string)
}

// HandlePasskeyLoginStart builds and returns a passkey assertion challenge.
func HandlePasskeyLoginStart(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if d.AuthClient == nil || d.AuthClient() == nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}

	resp, err := d.AuthClient().BeginPasskeyLogin(r.Context(), &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_login")
		return
	}
	d.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": resp.GetSessionId(),
		"public_key": json.RawMessage(resp.GetCredentialRequestOptionsJson()),
	})
}

// HandlePasskeyLoginFinish validates assertion JSON and returns consent URL for pending flow.
func HandlePasskeyLoginFinish(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if d.AuthClient == nil || d.AuthClient() == nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	if _, err := d.AuthClient().FinishPasskeyLogin(r.Context(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
		PendingId:              payload.PendingID,
	}); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_login")
		return
	}

	d.WriteJSON(w, http.StatusOK, map[string]any{
		"redirect_url": webcache.BuildAuthConsentURL(d.AuthBaseURL, strings.TrimSpace(payload.PendingID)),
	})
}

// HandlePasskeyRegisterStart creates registration options and optionally resolves locale.
func HandlePasskeyRegisterStart(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if d.AuthClient == nil || d.AuthClient() == nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		Email     string `json:"email"`
		PendingID string `json:"pending_id"`
		Locale    string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.Email) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.email_is_required")
		return
	}

	requestLocale := strings.TrimSpace(payload.Locale)
	resolvedLocale := platformi18n.LocaleForTag(webi18n.Default())
	if requestLocale != "" {
		parsedLocale, ok := platformi18n.ParseLocale(requestLocale)
		if !ok {
			d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_locale")
			return
		}
		resolvedLocale = parsedLocale
	} else {
		if printer, _ := d.Localizer(w, r); printer != nil {
			if tag, ok := webi18n.ResolveTag(r); ok {
				resolvedLocale = platformi18n.LocaleForTag(tag)
			}
		}
	}

	createResp, err := d.AuthClient().CreateUser(r.Context(), &authv1.CreateUserRequest{
		Email:  payload.Email,
		Locale: resolvedLocale,
	})
	if err != nil || createResp.GetUser() == nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.failed_to_create_user")
		return
	}

	beginResp, err := d.AuthClient().BeginPasskeyRegistration(r.Context(), &authv1.BeginPasskeyRegistrationRequest{UserId: createResp.GetUser().GetId()})
	if err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_registration")
		return
	}

	d.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": beginResp.GetSessionId(),
		"public_key": json.RawMessage(beginResp.GetCredentialCreationOptionsJson()),
		"user_id":    createResp.GetUser().GetId(),
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// HandlePasskeyRegisterFinish validates registration response and returns flow identifiers.
func HandlePasskeyRegisterFinish(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if d.AuthClient == nil || d.AuthClient() == nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		UserID     string          `json:"user_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if strings.TrimSpace(payload.UserID) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.user_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	if _, err := d.AuthClient().FinishPasskeyRegistration(r.Context(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
	}); err != nil {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_registration")
		return
	}

	d.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id":    payload.UserID,
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// HandleMagicLink renders magic link success/failure pages and follows consent redirects.
func HandleMagicLink(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	printer, lang := d.Localizer(w, r)
	if d.AuthClient == nil || d.AuthClient() == nil {
		RenderMagicPage(d, w, r, http.StatusInternalServerError, webtemplates.MagicParams{
			AppName: d.ResolvedAppName(),
			Title:   webtemplates.T(printer, "magic.unavailable.title"),
			Message: webtemplates.T(printer, "magic.unavailable.message"),
			Detail:  webtemplates.T(printer, "magic.unavailable.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		RenderMagicPage(d, w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: d.ResolvedAppName(),
			Title:   webtemplates.T(printer, "magic.missing.title"),
			Message: webtemplates.T(printer, "magic.missing.message"),
			Detail:  webtemplates.T(printer, "magic.missing.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	resp, err := d.AuthClient().ConsumeMagicLink(r.Context(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		RenderMagicPage(d, w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: d.ResolvedAppName(),
			Title:   webtemplates.T(printer, "magic.invalid.title"),
			Message: webtemplates.T(printer, "magic.invalid.message"),
			Detail:  webtemplates.T(printer, "magic.invalid.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	if pendingID := strings.TrimSpace(resp.GetPendingId()); pendingID != "" {
		http.Redirect(w, r, d.BuildAuthConsentURL(d.AuthBaseURL, pendingID), http.StatusFound)
		return
	}

	RenderMagicPage(d, w, r, http.StatusOK, webtemplates.MagicParams{
		AppName:   d.ResolvedAppName(),
		Title:     webtemplates.T(printer, "magic.verified.title"),
		Message:   webtemplates.T(printer, "magic.verified.message"),
		Detail:    webtemplates.T(printer, "magic.verified.detail"),
		Loc:       printer,
		Success:   true,
		LinkURL:   "/",
		LinkLabel: webtemplates.T(printer, "magic.verified.link"),
		Lang:      lang,
	})
}

// HandleAuthLogin starts OAuth PKCE flow and redirects to auth authorization endpoint.
func HandleAuthLogin(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if strings.TrimSpace(d.OAuthClientID) == "" || strings.TrimSpace(d.CallbackURL) == "" {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	verifier, err := d.GenerateCodeVerifier()
	if err != nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.failed_to_generate_pkce_verifier")
		return
	}
	challenge := d.ComputeS256Challenge(verifier)
	state := ""
	if d.CreatePendingState != nil {
		state = d.CreatePendingState(verifier)
	}

	authorizeURL := strings.TrimRight(strings.TrimSpace(d.AuthBaseURL), "/") + "/authorize"
	redirectURL, err := url.Parse(authorizeURL)
	if err != nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.invalid_auth_base_url")
		return
	}

	q := redirectURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", d.OAuthClientID)
	q.Set("redirect_uri", d.CallbackURL)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// HandleAuthCallback exchanges the authorization code for token and persists local session state.
func HandleAuthCallback(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || state == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.missing_code_or_state")
		return
	}
	flowCodeVerifier := ""
	ok := false
	if d.ConsumePendingState != nil {
		flowCodeVerifier, ok = d.ConsumePendingState(state)
	}
	if !ok || strings.TrimSpace(flowCodeVerifier) == "" {
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.invalid_or_expired_state")
		return
	}

	tokenURL := strings.TrimSpace(d.AuthTokenURL)
	if tokenURL == "" {
		tokenURL = strings.TrimRight(strings.TrimSpace(d.AuthBaseURL), "/") + "/token"
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {d.CallbackURL},
		"code_verifier": {flowCodeVerifier},
		"client_id":     {d.OAuthClientID},
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		d.LocalizeError(w, r, http.StatusBadGateway, "error.http.token_exchange_failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d.LocalizeError(w, r, http.StatusBadGateway, "error.http.token_exchange_returned", resp.Status)
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		d.LocalizeError(w, r, http.StatusBadGateway, "error.http.failed_to_decode_token_response")
		return
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		d.LocalizeError(w, r, http.StatusBadGateway, "error.http.empty_access_token")
		return
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	sessionID := ""
	if d.CreateSession != nil {
		sessionID = d.CreateSession(tokenResp.AccessToken, "", expiry)
	}
	if d.WriteSessionCookie != nil {
		d.WriteSessionCookie(w, sessionID)
	}

	tokenCookieDomain := ""
	if d.TokenCookieDomainForRequest != nil {
		tokenCookieDomain = d.TokenCookieDomainForRequest(r.Host)
	}
	if d.WriteTokenCookie != nil {
		d.WriteTokenCookie(w, tokenResp.AccessToken, tokenCookieDomain, int(tokenResp.ExpiresIn))
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleAuthLogout clears session and token cookies and removes local session record.
func HandleAuthLogout(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	if d.SessionIDFromRequest != nil {
		sid := d.SessionIDFromRequest(r)
		if sid != "" && d.DeleteSession != nil {
			d.DeleteSession(sid)
		}
	}
	if d.ClearSessionCookie != nil {
		d.ClearSessionCookie(w)
	}
	tokenCookieDomain := ""
	if d.TokenCookieDomainForRequest != nil {
		tokenCookieDomain = d.TokenCookieDomainForRequest(r.Host)
	}
	if d.ClearTokenCookie != nil {
		d.ClearTokenCookie(w, tokenCookieDomain)
		if strings.TrimSpace(tokenCookieDomain) != strings.TrimSpace(d.ConfigDomain) {
			d.ClearTokenCookie(w, strings.TrimSpace(d.ConfigDomain))
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleAuthLoginPage renders the first-party login page used by /login route.
func HandleAuthLoginPage(d AuthFlowDependencies, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	printer, lang := d.Localizer(w, r)

	pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
	if pendingID == "" {
		if strings.TrimSpace(d.OAuthClientID) != "" {
			http.Redirect(w, r, d.AuthLoginPath, http.StatusFound)
			return
		}
		d.LocalizeError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}

	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	clientName := strings.TrimSpace(r.URL.Query().Get("client_name"))
	errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))
	if clientName == "" {
		if clientID != "" {
			clientName = clientID
		} else if printer != nil {
			clientName = webtemplates.T(printer, "web.login.unknown_client")
		}
	}

	page := webtemplates.LoginParams{
		AppName:      d.ResolvedAppName(),
		PendingID:    pendingID,
		ClientName:   clientName,
		Error:        errorMessage,
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
	}

	if err := d.WritePage(w, r, webtemplates.LoginPage(page), d.ComposeHTMXTitle(printer, "title.login")); err != nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// WriteJSON writes a JSON response with normalized headers and status.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	websupport.WriteJSON(w, status, payload)
}

// RenderMagicPage renders the shared magic-link status page.
func RenderMagicPage(_ AuthFlowDependencies, w http.ResponseWriter, r *http.Request, status int, params webtemplates.MagicParams) {
	if r != nil {
		if strings.TrimSpace(params.CurrentPath) == "" {
			params.CurrentPath = r.URL.Path
		}
		if strings.TrimSpace(params.CurrentQuery) == "" {
			params.CurrentQuery = r.URL.RawQuery
		}
	}
	w.WriteHeader(status)
	if err := websupport.WritePage(
		w,
		r,
		webtemplates.MagicPage(params),
		websupport.ComposeHTMXTitle(params.Loc, params.Title),
	); err != nil {
		log.Printf("web: failed to render magic page: %v", err)
	}
}
