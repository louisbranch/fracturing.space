package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New("record not found")

// ErrConflict indicates a requested state transition is invalid.
var ErrConflict = errors.New("record conflict")

// CredentialRecord stores a persisted provider credential.
type CredentialRecord struct {
	ID          string
	OwnerUserID string
	Provider    string
	Label       string
	// SecretCiphertext stores encrypted credential material only; plaintext
	// secrets must never cross into storage records.
	SecretCiphertext string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RevokedAt        *time.Time
}

// CredentialPage is a paged set of credentials.
type CredentialPage struct {
	Credentials   []CredentialRecord
	NextPageToken string
}

// AgentRecord stores a persisted AI agent profile.
type AgentRecord struct {
	ID              string
	OwnerUserID     string
	Name            string
	Provider        string
	Model           string
	CredentialID    string
	ProviderGrantID string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AgentPage is a paged set of agents.
type AgentPage struct {
	Agents        []AgentRecord
	NextPageToken string
}

// ProviderGrantRecord stores a persisted provider OAuth grant.
type ProviderGrantRecord struct {
	ID          string
	OwnerUserID string
	Provider    string

	GrantedScopes []string

	// TokenCiphertext stores encrypted grant token material only.
	TokenCiphertext string

	RefreshSupported bool
	Status           string
	LastRefreshError string

	CreatedAt       time.Time
	UpdatedAt       time.Time
	RevokedAt       *time.Time
	ExpiresAt       *time.Time
	LastRefreshedAt *time.Time
}

// ProviderGrantPage is a paged set of provider grants.
type ProviderGrantPage struct {
	ProviderGrants []ProviderGrantRecord
	NextPageToken  string
}

// ProviderGrantFilter narrows owner-scoped provider-grant listing.
//
// Security note: owner scope is mandatory and enforced separately; optional
// filter fields can only reduce result visibility.
type ProviderGrantFilter struct {
	Provider string
	Status   string
}

// AccessRequestRecord stores one owner-reviewed access request for an agent.
type AccessRequestRecord struct {
	ID string

	RequesterUserID string
	OwnerUserID     string
	AgentID         string
	Scope           string

	RequestNote string

	Status string

	ReviewerUserID string
	ReviewNote     string

	CreatedAt  time.Time
	UpdatedAt  time.Time
	ReviewedAt *time.Time
}

// AccessRequestPage is a paged set of access requests.
type AccessRequestPage struct {
	AccessRequests []AccessRequestRecord
	NextPageToken  string
}

// AuditEventRecord stores one append-only AI audit event.
type AuditEventRecord struct {
	ID string

	EventName string

	ActorUserID string

	OwnerUserID     string
	RequesterUserID string
	AgentID         string
	AccessRequestID string

	Outcome string

	CreatedAt time.Time
}

// AuditEventPage is a paged set of audit events.
type AuditEventPage struct {
	AuditEvents   []AuditEventRecord
	NextPageToken string
}

// AuditEventFilter narrows owner-scoped audit event listing.
//
// Security note: owner scope is mandatory and enforced separately; these fields
// are optional in-memory constraints that must never broaden tenant visibility.
type AuditEventFilter struct {
	EventName string
	AgentID   string

	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// ProviderConnectSessionRecord stores one provider OAuth connect handshake.
type ProviderConnectSessionRecord struct {
	ID              string
	OwnerUserID     string
	Provider        string
	Status          string
	RequestedScopes []string

	// StateHash stores a non-reversible hash of the outbound OAuth state token.
	StateHash string
	// CodeVerifierCiphertext stores encrypted PKCE verifier material.
	CodeVerifierCiphertext string

	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	CompletedAt *time.Time
}

// CredentialStore persists credential records.
type CredentialStore interface {
	PutCredential(ctx context.Context, record CredentialRecord) error
	GetCredential(ctx context.Context, credentialID string) (CredentialRecord, error)
	ListCredentialsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (CredentialPage, error)
	RevokeCredential(ctx context.Context, ownerUserID string, credentialID string, revokedAt time.Time) error
}

// AgentStore persists AI agent records.
type AgentStore interface {
	PutAgent(ctx context.Context, record AgentRecord) error
	GetAgent(ctx context.Context, agentID string) (AgentRecord, error)
	ListAgentsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (AgentPage, error)
	DeleteAgent(ctx context.Context, ownerUserID string, agentID string) error
}

// ProviderGrantStore persists provider grant records.
type ProviderGrantStore interface {
	PutProviderGrant(ctx context.Context, record ProviderGrantRecord) error
	GetProviderGrant(ctx context.Context, providerGrantID string) (ProviderGrantRecord, error)
	ListProviderGrantsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter ProviderGrantFilter) (ProviderGrantPage, error)
	RevokeProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string, revokedAt time.Time) error
	UpdateProviderGrantToken(ctx context.Context, ownerUserID string, providerGrantID string, tokenCiphertext string, refreshedAt time.Time, expiresAt *time.Time, status string, lastRefreshError string) error
}

// ProviderConnectSessionStore persists connect-session records.
type ProviderConnectSessionStore interface {
	PutProviderConnectSession(ctx context.Context, record ProviderConnectSessionRecord) error
	GetProviderConnectSession(ctx context.Context, connectSessionID string) (ProviderConnectSessionRecord, error)
	CompleteProviderConnectSession(ctx context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error
}

// AccessRequestStore persists agent access-request records.
type AccessRequestStore interface {
	PutAccessRequest(ctx context.Context, record AccessRequestRecord) error
	GetAccessRequest(ctx context.Context, accessRequestID string) (AccessRequestRecord, error)
	ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (AccessRequestPage, error)
	ListAccessRequestsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (AccessRequestPage, error)
	// GetApprovedInvokeAccessByRequesterForAgent returns one approved invoke grant
	// for a requester/owner/agent tuple. Callers use this to authorize a single
	// invoke decision without scanning unrelated access requests.
	GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (AccessRequestRecord, error)
	// ListApprovedInvokeAccessRequestsByRequester returns only approved invoke
	// access rows for one requester. This narrows list-accessible authorization
	// scans to relevant records.
	ListApprovedInvokeAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (AccessRequestPage, error)
	ReviewAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, reviewedAt time.Time) error
	RevokeAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, revokedAt time.Time) error
}

// AuditEventStore persists append-only AI audit events.
type AuditEventStore interface {
	PutAuditEvent(ctx context.Context, record AuditEventRecord) error
	ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter AuditEventFilter) (AuditEventPage, error)
}
