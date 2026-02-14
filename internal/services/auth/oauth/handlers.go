package oauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"golang.org/x/crypto/bcrypt"
)

type loginView struct {
	AppName    string
	PendingID  string
	ClientID   string
	ClientName string
	Error      string
}

type consentView struct {
	AppName    string
	PendingID  string
	ClientID   string
	ClientName string
	Username   string
	Scopes     []string
}

type errorView struct {
	AppName          string
	Error            string
	ErrorDescription string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type introspectResponse struct {
	Active   bool   `json:"active"`
	Scope    string `json:"scope,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Exp      int64  `json:"exp,omitempty"`
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()
	request := AuthorizationRequest{
		ResponseType:        params.Get("response_type"),
		ClientID:            params.Get("client_id"),
		RedirectURI:         params.Get("redirect_uri"),
		Scope:               params.Get("scope"),
		State:               params.Get("state"),
		CodeChallenge:       params.Get("code_challenge"),
		CodeChallengeMethod: params.Get("code_challenge_method"),
	}

	if request.ResponseType != "code" {
		s.renderError(w, "unsupported_response_type", "Only 'code' response type is supported", http.StatusBadRequest)
		return
	}

	client := s.clientForID(request.ClientID)
	if client == nil {
		s.renderError(w, "invalid_request", "Unknown client_id", http.StatusBadRequest)
		return
	}

	if request.RedirectURI == "" {
		s.renderError(w, "invalid_request", "redirect_uri is required", http.StatusBadRequest)
		return
	}
	if !redirectURIAllowed(request.RedirectURI, client.RedirectURIs) {
		s.renderError(w, "invalid_request", "redirect_uri is not registered", http.StatusBadRequest)
		return
	}

	if request.CodeChallenge == "" {
		s.redirectError(w, r, request, "invalid_request", "code_challenge is required")
		return
	}
	if request.CodeChallengeMethod != "S256" {
		s.redirectError(w, r, request, "invalid_request", "code_challenge_method must be S256")
		return
	}
	if !ValidateCodeChallenge(request.CodeChallenge) {
		s.redirectError(w, r, request, "invalid_request", "invalid code_challenge format")
		return
	}

	pendingID, err := s.store.CreatePendingAuthorization(request, s.config.PendingAuthorizationTTL)
	if err != nil {
		s.redirectError(w, r, request, "server_error", "failed to create authorization request")
		return
	}

	loginUIURL := strings.TrimSpace(s.config.LoginUIURL)
	if loginUIURL != "" {
		redirectURL, err := url.Parse(loginUIURL)
		if err != nil {
			s.renderError(w, "server_error", "invalid login ui url", http.StatusInternalServerError)
			return
		}
		query := redirectURL.Query()
		query.Set("pending_id", pendingID)
		query.Set("client_id", client.ID)
		query.Set("client_name", clientDisplayName(client))
		redirectURL.RawQuery = query.Encode()
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}

	view := loginView{
		AppName:    branding.AppName,
		PendingID:  pendingID,
		ClientID:   client.ID,
		ClientName: clientDisplayName(client),
	}
	if err := templates.ExecuteTemplate(w, "login.html", view); err != nil {
		http.Error(w, "failed to render login", http.StatusInternalServerError)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	pendingID := strings.TrimSpace(r.FormValue("pending_id"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	pending, err := s.store.GetPendingAuthorization(pendingID)
	if err != nil || pending == nil {
		s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
		return
	}
	if pending.ExpiresAt.Before(s.clock().UTC()) {
		s.store.DeletePendingAuthorization(pendingID)
		s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetOAuthUserByUsername(username)
	if err != nil || user == nil {
		s.renderLoginError(w, r, pending, "invalid username or password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.renderLoginError(w, r, pending, "invalid username or password")
		return
	}

	if err := s.store.UpdatePendingAuthorizationUserID(pendingID, user.UserID); err != nil {
		s.renderError(w, "server_error", "failed to update authorization", http.StatusInternalServerError)
		return
	}

	view := consentView{
		AppName:    branding.AppName,
		PendingID:  pendingID,
		ClientID:   pending.Request.ClientID,
		ClientName: clientDisplayName(s.clientForID(pending.Request.ClientID)),
		Username:   user.DisplayName,
		Scopes:     formatScopes(pending.Request.Scope),
	}
	if err := templates.ExecuteTemplate(w, "consent.html", view); err != nil {
		http.Error(w, "failed to render consent", http.StatusInternalServerError)
	}
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
		pending, err := s.store.GetPendingAuthorization(pendingID)
		if err != nil || pending == nil {
			s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
			return
		}
		if pending.ExpiresAt.Before(s.clock().UTC()) {
			s.store.DeletePendingAuthorization(pendingID)
			s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
			return
		}
		if pending.UserID == "" {
			s.renderError(w, "invalid_request", "user not authenticated", http.StatusBadRequest)
			return
		}
		s.renderConsentView(w, r, pending)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	pendingID := strings.TrimSpace(r.FormValue("pending_id"))
	decision := strings.TrimSpace(r.FormValue("decision"))

	pending, err := s.store.GetPendingAuthorization(pendingID)
	if err != nil || pending == nil {
		s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
		return
	}
	if pending.ExpiresAt.Before(s.clock().UTC()) {
		s.store.DeletePendingAuthorization(pendingID)
		s.renderError(w, "invalid_request", "authorization session expired", http.StatusBadRequest)
		return
	}
	if pending.UserID == "" {
		s.renderError(w, "invalid_request", "user not authenticated", http.StatusBadRequest)
		return
	}

	defer s.store.DeletePendingAuthorization(pendingID)

	if decision != "allow" {
		s.redirectError(w, r, pending.Request, "access_denied", "user denied the request")
		return
	}

	code, err := s.store.CreateAuthorizationCode(pending.Request, pending.UserID, s.config.AuthorizationCodeTTL)
	if err != nil {
		s.redirectError(w, r, pending.Request, "server_error", "failed to create authorization code")
		return
	}

	redirectURL, err := url.Parse(pending.Request.RedirectURI)
	if err != nil {
		s.renderError(w, "server_error", "invalid redirect uri", http.StatusInternalServerError)
		return
	}
	query := redirectURL.Query()
	query.Set("code", code.Code)
	if pending.Request.State != "" {
		query.Set("state", pending.Request.State)
	}
	redirectURL.RawQuery = query.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid form data")
		return
	}

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if grantType != "authorization_code" {
		writeJSONError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported")
		return
	}
	if code == "" || codeVerifier == "" || clientID == "" || redirectURI == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing required fields")
		return
	}

	client := s.clientForID(clientID)
	if client == nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid_client", "unknown client")
		return
	}
	if err := validateTokenClientAuth(client, clientSecret); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid_client", "invalid client authentication")
		return
	}

	authCode, err := s.store.GetAuthorizationCode(code)
	if err != nil || authCode == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "invalid authorization code")
		return
	}
	if s.clock().UTC().After(authCode.ExpiresAt) {
		s.store.DeleteAuthorizationCode(code)
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code expired")
		return
	}
	if authCode.Used {
		s.store.DeleteAuthorizationCode(code)
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code already used")
		return
	}
	if authCode.ClientID != clientID {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
		return
	}
	if authCode.RedirectURI != redirectURI {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}
	if !ValidatePKCE(codeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	used, err := s.store.MarkAuthorizationCodeUsed(code)
	if err != nil || !used {
		writeJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code already used")
		return
	}

	accessToken, err := s.store.CreateAccessToken(authCode.ClientID, authCode.UserID, authCode.Scope, s.config.TokenTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "failed to create access token")
		return
	}

	s.store.DeleteAuthorizationCode(code)

	response := tokenResponse{
		AccessToken: accessToken.Token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.config.TokenTTL.Seconds()),
		Scope:       authCode.Scope,
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.config.ResourceSecret == "" {
		http.Error(w, "missing shared secret", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("X-Resource-Secret") != s.config.ResourceSecret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusBadRequest)
		return
	}
	accessToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if accessToken == "" {
		http.Error(w, "missing bearer token", http.StatusBadRequest)
		return
	}
	entry, ok, err := s.store.ValidateAccessToken(accessToken)
	if err != nil || !ok || entry == nil {
		writeJSON(w, http.StatusOK, introspectResponse{Active: false})
		return
	}

	writeJSON(w, http.StatusOK, introspectResponse{
		Active:   true,
		Scope:    entry.Scope,
		ClientID: entry.ClientID,
		UserID:   entry.UserID,
		Exp:      entry.ExpiresAt.Unix(),
	})
}

func (s *Server) renderError(w http.ResponseWriter, code, description string, status int) {
	w.WriteHeader(status)
	_ = templates.ExecuteTemplate(w, "error.html", errorView{AppName: branding.AppName, Error: code, ErrorDescription: description})
}

func (s *Server) renderLoginError(w http.ResponseWriter, r *http.Request, pending *PendingAuthorization, message string) {
	client := s.clientForID(pending.Request.ClientID)
	loginUIURL := strings.TrimSpace(s.config.LoginUIURL)
	if loginUIURL != "" {
		redirectURL, err := url.Parse(loginUIURL)
		if err != nil {
			s.renderError(w, "server_error", "invalid login ui url", http.StatusInternalServerError)
			return
		}
		query := redirectURL.Query()
		query.Set("pending_id", pending.ID)
		query.Set("client_id", pending.Request.ClientID)
		query.Set("client_name", clientDisplayName(client))
		query.Set("error", message)
		redirectURL.RawQuery = query.Encode()
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}
	view := loginView{
		AppName:    branding.AppName,
		PendingID:  pending.ID,
		ClientID:   pending.Request.ClientID,
		ClientName: clientDisplayName(client),
		Error:      message,
	}
	_ = templates.ExecuteTemplate(w, "login.html", view)
}

func (s *Server) renderConsentView(w http.ResponseWriter, r *http.Request, pending *PendingAuthorization) {
	client := s.clientForID(pending.Request.ClientID)
	username := pending.UserID
	if s.userStore != nil {
		if user, err := s.userStore.GetUser(r.Context(), pending.UserID); err == nil {
			if strings.TrimSpace(user.DisplayName) != "" {
				username = user.DisplayName
			}
		}
	}
	view := consentView{
		AppName:    branding.AppName,
		PendingID:  pending.ID,
		ClientID:   pending.Request.ClientID,
		ClientName: clientDisplayName(client),
		Username:   username,
		Scopes:     formatScopes(pending.Request.Scope),
	}
	_ = templates.ExecuteTemplate(w, "consent.html", view)
}

func (s *Server) redirectError(w http.ResponseWriter, r *http.Request, request AuthorizationRequest, code, description string) {
	redirectURL, err := url.Parse(request.RedirectURI)
	if err != nil {
		s.renderError(w, "server_error", "invalid redirect uri", http.StatusInternalServerError)
		return
	}
	query := redirectURL.Query()
	query.Set("error", code)
	query.Set("error_description", description)
	if request.State != "" {
		query.Set("state", request.State)
	}
	redirectURL.RawQuery = query.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) clientForID(clientID string) *Client {
	if clientID == "" {
		return nil
	}
	for _, client := range s.config.Clients {
		if client.ID == clientID {
			return &client
		}
	}
	return nil
}

func clientDisplayName(client *Client) string {
	if client == nil {
		return "Unknown Client"
	}
	if client.Name != "" {
		return client.Name
	}
	return client.ID
}

func redirectURIAllowed(uri string, allowed []string) bool {
	for _, value := range allowed {
		if value == uri {
			return true
		}
	}
	return false
}

func validateTokenClientAuth(client *Client, clientSecret string) error {
	if client == nil {
		return errors.New("unknown client")
	}
	method := strings.TrimSpace(client.TokenEndpointAuthMethod)
	if method == "" {
		if client.Secret != "" {
			method = "client_secret_post"
		} else {
			method = "none"
		}
	}
	if method == "none" {
		return nil
	}
	if method != "client_secret_post" {
		return errors.New("unsupported token endpoint auth method")
	}
	if client.Secret == "" {
		return errors.New("client secret not configured")
	}
	if clientSecret == "" || clientSecret != client.Secret {
		return errors.New("invalid client authentication")
	}
	return nil
}

func formatScopes(scope string) []string {
	values := strings.Fields(scope)
	if len(values) == 0 {
		return []string{"basic profile"}
	}
	return values
}

func writeJSONError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, errorResponse{Error: code, ErrorDescription: description})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(payload)
}
