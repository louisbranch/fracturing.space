package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

func (s *Server) handleProviderRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/oauth/providers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	providerID := parts[0]
	action := parts[1]

	switch action {
	case "start":
		s.handleProviderStart(w, r, providerID)
	case "callback":
		s.handleProviderCallback(w, r, providerID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleProviderStart(w http.ResponseWriter, r *http.Request, providerID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider, ok := s.config.Providers[providerID]
	if !ok {
		http.NotFound(w, r)
		return
	}
	redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
	if redirectURI != "" && !isAllowedRedirect(redirectURI, s.config.LoginRedirectAllowlist) {
		http.Error(w, "redirect_uri is not allowed", http.StatusBadRequest)
		return
	}

	codeVerifier, err := newCodeVerifier()
	if err != nil {
		http.Error(w, "failed to generate code verifier", http.StatusInternalServerError)
		return
	}
	codeChallenge := ComputeS256Challenge(codeVerifier)

	state, err := s.store.CreateProviderState(providerID, redirectURI, codeVerifier, s.config.PendingAuthorizationTTL)
	if err != nil {
		http.Error(w, "failed to start provider flow", http.StatusInternalServerError)
		return
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", provider.ClientID)
	query.Set("redirect_uri", provider.RedirectURI)
	query.Set("scope", strings.Join(provider.Scopes, " "))
	query.Set("state", state.State)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")

	authURL, err := url.Parse(provider.AuthURL)
	if err != nil {
		http.Error(w, "invalid provider config", http.StatusInternalServerError)
		return
	}
	authURL.RawQuery = query.Encode()
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (s *Server) handleProviderCallback(w http.ResponseWriter, r *http.Request, providerID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider, ok := s.config.Providers[providerID]
	if !ok {
		http.NotFound(w, r)
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		s.renderError(w, errParam, r.URL.Query().Get("error_description"), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	stateValue := r.URL.Query().Get("state")
	if code == "" || stateValue == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	state, err := s.store.GetProviderState(stateValue)
	if err != nil || state == nil {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	if state.ExpiresAt.Before(s.clock().UTC()) {
		s.store.DeleteProviderState(stateValue)
		http.Error(w, "state expired", http.StatusBadRequest)
		return
	}
	defer s.store.DeleteProviderState(stateValue)

	token, err := s.exchangeProviderToken(r.Context(), provider, code, state.CodeVerifier)
	if err != nil {
		http.Error(w, "failed to exchange provider token", http.StatusBadRequest)
		return
	}

	profile, err := s.fetchProviderProfile(r.Context(), provider, token.AccessToken)
	if err != nil {
		http.Error(w, "failed to fetch provider profile", http.StatusBadRequest)
		return
	}

	userID, err := s.ensureUserForProfile(r.Context(), providerID, profile)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	identityID, err := id.NewID()
	if err != nil {
		http.Error(w, "failed to store identity", http.StatusInternalServerError)
		return
	}
	expiresAt := token.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = s.clock().UTC().Add(time.Hour)
	}

	err = s.store.UpsertExternalIdentity(ExternalIdentity{
		ID:             identityID,
		Provider:       providerID,
		ProviderUserID: profile.ProviderUserID,
		UserID:         userID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		Scope:          token.Scope,
		ExpiresAt:      expiresAt,
		IDToken:        token.IDToken,
		UpdatedAt:      s.clock().UTC(),
	})
	if err != nil {
		http.Error(w, "failed to store identity", http.StatusInternalServerError)
		return
	}

	if state.RedirectURI != "" {
		redirectURL, err := url.Parse(state.RedirectURI)
		if err != nil {
			http.Error(w, "invalid redirect uri", http.StatusBadRequest)
			return
		}
		query := redirectURL.Query()
		query.Set("user_id", userID)
		query.Set("provider", providerID)
		query.Set("provider_user_id", profile.ProviderUserID)
		redirectURL.RawQuery = query.Encode()
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":          userID,
		"provider":         providerID,
		"provider_user_id": profile.ProviderUserID,
	})
}

type providerToken struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	ExpiresAt    time.Time
	IDToken      string
}

func (s *Server) exchangeProviderToken(ctx context.Context, provider ProviderConfig, code, codeVerifier string) (providerToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", provider.RedirectURI)
	form.Set("client_id", provider.ClientID)
	form.Set("client_secret", provider.ClientSecret)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return providerToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return providerToken{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return providerToken{}, errors.New("token exchange failed")
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		ExpiresIn    int64  `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return providerToken{}, err
	}
	if payload.AccessToken == "" {
		return providerToken{}, errors.New("missing access token")
	}

	expiresAt := time.Time{}
	if payload.ExpiresIn > 0 {
		expiresAt = s.clock().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return providerToken{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		Scope:        payload.Scope,
		ExpiresAt:    expiresAt,
		IDToken:      payload.IDToken,
	}, nil
}

type providerProfile struct {
	ProviderUserID string
	DisplayName    string
}

func (s *Server) fetchProviderProfile(ctx context.Context, provider ProviderConfig, accessToken string) (providerProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserInfoURL, nil)
	if err != nil {
		return providerProfile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return providerProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return providerProfile{}, errors.New("profile request failed")
	}

	if strings.EqualFold(provider.Name, "Google") {
		var payload struct {
			Sub   string `json:"sub"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return providerProfile{}, err
		}
		return providerProfile{ProviderUserID: payload.Sub, DisplayName: firstNonEmpty(payload.Name, payload.Email, payload.Sub)}, nil
	}

	var payload struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return providerProfile{}, err
	}
	return providerProfile{ProviderUserID: formatGitHubID(payload.ID), DisplayName: firstNonEmpty(payload.Name, payload.Login, payload.Email)}, nil
}

func (s *Server) ensureUserForProfile(ctx context.Context, providerID string, profile providerProfile) (string, error) {
	if profile.ProviderUserID == "" {
		return "", errors.New("missing provider user id")
	}
	identity, err := s.store.GetExternalIdentity(providerID, profile.ProviderUserID)
	if err != nil {
		return "", err
	}
	if identity != nil {
		return identity.UserID, nil
	}
	if s.userStore == nil {
		return "", errors.New("user store not configured")
	}
	created, err := user.CreateUser(user.CreateUserInput{DisplayName: profile.DisplayName}, s.clock, id.NewID)
	if err != nil {
		return "", err
	}
	if err := s.userStore.PutUser(ctx, created); err != nil {
		return "", err
	}
	return created.ID, nil
}

func newCodeVerifier() (string, error) {
	verifier, err := generateToken(48)
	if err != nil {
		return "", err
	}
	return verifier, nil
}

func isAllowedRedirect(uri string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		if strings.TrimSpace(allowed) == uri {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Unknown User"
}

func formatGitHubID(value int64) string {
	if value == 0 {
		return ""
	}
	return "github-" + strconv.FormatInt(value, 10)
}
