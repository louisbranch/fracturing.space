package oauth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

const oauthTimeFormat = time.RFC3339Nano

// Store provides SQLite-backed storage for OAuth data.
type Store struct {
	db *sql.DB
}

// NewStore creates a new OAuth store using the provided database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ensureDB() error {
	if s == nil || s.db == nil {
		return errors.New("oauth store is not configured")
	}
	return nil
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// UpsertOAuthUserCredentials stores credentials for a user.
func (s *Store) UpsertOAuthUserCredentials(userID, username, passwordHash string, now time.Time) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO oauth_user_credentials (user_id, username, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			username = excluded.username,
			password_hash = excluded.password_hash,
			updated_at = excluded.updated_at`,
		userID, username, passwordHash, now.Format(oauthTimeFormat), now.Format(oauthTimeFormat),
	)
	return err
}

// GetOAuthUserByUsername returns the oauth user credentials by username.
func (s *Store) GetOAuthUserByUsername(username string) (*OAuthUser, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}

	var user OAuthUser
	err := s.db.QueryRow(
		`SELECT c.user_id, c.username, c.password_hash, u.display_name
		FROM oauth_user_credentials c
		JOIN users u ON u.id = c.user_id
		WHERE c.username = ?`,
		username,
	).Scan(&user.UserID, &user.Username, &user.PasswordHash, &user.DisplayName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateAuthorizationCode stores a new authorization code.
func (s *Store) CreateAuthorizationCode(request AuthorizationRequest, userID string, ttl time.Duration) (*AuthorizationCode, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	code, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.db.Exec(
		`INSERT INTO oauth_authorization_codes
		(code, client_id, user_id, redirect_uri, code_challenge, code_challenge_method, scope, state, expires_at, used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		code, request.ClientID, userID, request.RedirectURI, request.CodeChallenge,
		request.CodeChallengeMethod, request.Scope, request.State, expiresAt.Format(oauthTimeFormat),
	)
	if err != nil {
		return nil, err
	}
	return &AuthorizationCode{
		Code:                code,
		ClientID:            request.ClientID,
		UserID:              userID,
		RedirectURI:         request.RedirectURI,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
		Scope:               request.Scope,
		State:               request.State,
		ExpiresAt:           expiresAt,
		Used:                false,
	}, nil
}

// GetAuthorizationCode retrieves an authorization code.
func (s *Store) GetAuthorizationCode(code string) (*AuthorizationCode, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	var authCode AuthorizationCode
	var expiresAt string
	var used int
	err := s.db.QueryRow(
		`SELECT code, client_id, user_id, redirect_uri, code_challenge, code_challenge_method, scope, state, expires_at, used
		FROM oauth_authorization_codes WHERE code = ?`,
		code,
	).Scan(
		&authCode.Code, &authCode.ClientID, &authCode.UserID, &authCode.RedirectURI,
		&authCode.CodeChallenge, &authCode.CodeChallengeMethod, &authCode.Scope, &authCode.State,
		&expiresAt, &used,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiry, err := time.Parse(oauthTimeFormat, expiresAt)
	if err != nil {
		return nil, err
	}
	authCode.ExpiresAt = expiry
	authCode.Used = used != 0
	return &authCode, nil
}

// MarkAuthorizationCodeUsed marks a code as used.
func (s *Store) MarkAuthorizationCodeUsed(code string) (bool, error) {
	if err := s.ensureDB(); err != nil {
		return false, err
	}
	result, err := s.db.Exec(
		`UPDATE oauth_authorization_codes SET used = 1 WHERE code = ? AND used = 0`,
		code,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// DeleteAuthorizationCode deletes a code.
func (s *Store) DeleteAuthorizationCode(code string) {
	if s == nil || s.db == nil {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM oauth_authorization_codes WHERE code = ?`, code)
}

// CreateAccessToken creates and stores a new access token.
func (s *Store) CreateAccessToken(clientID, userID, scope string, ttl time.Duration) (*AccessToken, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.db.Exec(
		`INSERT INTO oauth_access_tokens (token, client_id, user_id, scope, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		token, clientID, userID, scope, expiresAt.Format(oauthTimeFormat),
	)
	if err != nil {
		return nil, err
	}
	return &AccessToken{Token: token, ClientID: clientID, UserID: userID, Scope: scope, ExpiresAt: expiresAt}, nil
}

// GetAccessToken retrieves an access token.
func (s *Store) GetAccessToken(token string) (*AccessToken, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	var access AccessToken
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT token, client_id, user_id, scope, expires_at FROM oauth_access_tokens WHERE token = ?`,
		token,
	).Scan(&access.Token, &access.ClientID, &access.UserID, &access.Scope, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiry, err := time.Parse(oauthTimeFormat, expiresAt)
	if err != nil {
		return nil, err
	}
	access.ExpiresAt = expiry
	return &access, nil
}

// ValidateAccessToken checks if a token is valid.
func (s *Store) ValidateAccessToken(token string) (*AccessToken, bool, error) {
	if err := s.ensureDB(); err != nil {
		return nil, false, err
	}
	access, err := s.GetAccessToken(token)
	if err != nil || access == nil {
		return nil, false, err
	}
	if time.Now().After(access.ExpiresAt) {
		return nil, false, nil
	}
	return access, true, nil
}

// DeleteAccessToken removes an access token.
func (s *Store) DeleteAccessToken(token string) {
	if s == nil || s.db == nil {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM oauth_access_tokens WHERE token = ?`, token)
}

// CreatePendingAuthorization stores a pending authorization request.
func (s *Store) CreatePendingAuthorization(req AuthorizationRequest, ttl time.Duration) (string, error) {
	if err := s.ensureDB(); err != nil {
		return "", err
	}
	id, err := generateToken(16)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.db.Exec(
		`INSERT INTO oauth_pending_authorizations
		(id, response_type, client_id, redirect_uri, scope, state, code_challenge, code_challenge_method, user_id, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', ?)`,
		id, req.ResponseType, req.ClientID, req.RedirectURI, req.Scope, req.State,
		req.CodeChallenge, req.CodeChallengeMethod, expiresAt.Format(oauthTimeFormat),
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetPendingAuthorization retrieves a pending authorization.
func (s *Store) GetPendingAuthorization(id string) (*PendingAuthorization, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	var pending PendingAuthorization
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT id, response_type, client_id, redirect_uri, scope, state, code_challenge, code_challenge_method, user_id, expires_at
		FROM oauth_pending_authorizations WHERE id = ?`,
		id,
	).Scan(
		&pending.ID, &pending.Request.ResponseType, &pending.Request.ClientID, &pending.Request.RedirectURI,
		&pending.Request.Scope, &pending.Request.State, &pending.Request.CodeChallenge, &pending.Request.CodeChallengeMethod,
		&pending.UserID, &expiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiry, err := time.Parse(oauthTimeFormat, expiresAt)
	if err != nil {
		return nil, err
	}
	pending.ExpiresAt = expiry
	return &pending, nil
}

// UpdatePendingAuthorizationUserID sets the user ID for a pending authorization.
func (s *Store) UpdatePendingAuthorizationUserID(id, userID string) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE oauth_pending_authorizations SET user_id = ? WHERE id = ?`,
		userID, id,
	)
	return err
}

// DeletePendingAuthorization deletes a pending authorization.
func (s *Store) DeletePendingAuthorization(id string) {
	if s == nil || s.db == nil {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM oauth_pending_authorizations WHERE id = ?`, id)
}

// CreateProviderState stores a provider state for external OAuth flows.
func (s *Store) CreateProviderState(provider, redirectURI, codeVerifier string, ttl time.Duration) (*ProviderState, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	state, err := generateToken(16)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.db.Exec(
		`INSERT INTO oauth_provider_states (state, provider, redirect_uri, code_verifier, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		state, provider, redirectURI, codeVerifier, expiresAt.Format(oauthTimeFormat),
	)
	if err != nil {
		return nil, err
	}
	return &ProviderState{State: state, Provider: provider, RedirectURI: redirectURI, CodeVerifier: codeVerifier, ExpiresAt: expiresAt}, nil
}

// GetProviderState retrieves a provider state.
func (s *Store) GetProviderState(state string) (*ProviderState, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	var stored ProviderState
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT state, provider, redirect_uri, code_verifier, expires_at FROM oauth_provider_states WHERE state = ?`,
		state,
	).Scan(&stored.State, &stored.Provider, &stored.RedirectURI, &stored.CodeVerifier, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiry, err := time.Parse(oauthTimeFormat, expiresAt)
	if err != nil {
		return nil, err
	}
	stored.ExpiresAt = expiry
	return &stored, nil
}

// DeleteProviderState deletes a provider state.
func (s *Store) DeleteProviderState(state string) {
	if s == nil || s.db == nil {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM oauth_provider_states WHERE state = ?`, state)
}

// UpsertExternalIdentity stores an external identity.
func (s *Store) UpsertExternalIdentity(identity ExternalIdentity) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO oauth_external_identities
		(id, provider, provider_user_id, user_id, access_token, refresh_token, scope, expires_at, id_token, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, provider_user_id) DO UPDATE SET
			user_id = excluded.user_id,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			scope = excluded.scope,
			expires_at = excluded.expires_at,
			id_token = excluded.id_token,
			updated_at = excluded.updated_at`,
		identity.ID, identity.Provider, identity.ProviderUserID, identity.UserID,
		identity.AccessToken, identity.RefreshToken, identity.Scope,
		identity.ExpiresAt.Format(oauthTimeFormat), identity.IDToken,
		identity.UpdatedAt.Format(oauthTimeFormat),
	)
	return err
}

// GetExternalIdentity retrieves an external identity by provider + provider user ID.
func (s *Store) GetExternalIdentity(provider, providerUserID string) (*ExternalIdentity, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	var identity ExternalIdentity
	var expiresAt string
	var updatedAt string
	err := s.db.QueryRow(
		`SELECT id, provider, provider_user_id, user_id, access_token, refresh_token, scope, expires_at, id_token, updated_at
		FROM oauth_external_identities WHERE provider = ? AND provider_user_id = ?`,
		provider, providerUserID,
	).Scan(
		&identity.ID, &identity.Provider, &identity.ProviderUserID, &identity.UserID,
		&identity.AccessToken, &identity.RefreshToken, &identity.Scope, &expiresAt, &identity.IDToken, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiry, err := time.Parse(oauthTimeFormat, expiresAt)
	if err != nil {
		return nil, err
	}
	updated, err := time.Parse(oauthTimeFormat, updatedAt)
	if err != nil {
		return nil, err
	}
	identity.ExpiresAt = expiry
	identity.UpdatedAt = updated
	return &identity, nil
}

// CleanupExpired deletes expired rows.
func (s *Store) CleanupExpired(now time.Time) {
	if s == nil || s.db == nil {
		return
	}
	now = now.UTC()
	_, _ = s.db.Exec(`DELETE FROM oauth_authorization_codes WHERE expires_at <= ?`, now.Format(oauthTimeFormat))
	_, _ = s.db.Exec(`DELETE FROM oauth_access_tokens WHERE expires_at <= ?`, now.Format(oauthTimeFormat))
	_, _ = s.db.Exec(`DELETE FROM oauth_pending_authorizations WHERE expires_at <= ?`, now.Format(oauthTimeFormat))
	_, _ = s.db.Exec(`DELETE FROM oauth_provider_states WHERE expires_at <= ?`, now.Format(oauthTimeFormat))
}
