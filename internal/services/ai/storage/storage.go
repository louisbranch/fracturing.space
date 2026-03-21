package storage

import (
	"context"
	"errors"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New("record not found")

// ErrConflict indicates a requested state transition is invalid.
var ErrConflict = errors.New("record conflict")

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

// CampaignArtifactRecord stores one campaign-scoped GM working artifact.
type CampaignArtifactRecord struct {
	CampaignID string
	Path       string
	Content    string
	ReadOnly   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
	PutCredential(ctx context.Context, c credential.Credential) error
	GetCredential(ctx context.Context, credentialID string) (credential.Credential, error)
	ListCredentialsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (credential.Page, error)
}

// AgentStore persists AI agent profiles.
type AgentStore interface {
	PutAgent(ctx context.Context, a agent.Agent) error
	GetAgent(ctx context.Context, agentID string) (agent.Agent, error)
	ListAgentsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (agent.Page, error)
	// ListAccessibleAgents returns agents the user can invoke: owned agents
	// plus agents with approved shared invoke access, in one paginated query.
	ListAccessibleAgents(ctx context.Context, userID string, pageSize int, pageToken string) (agent.Page, error)
	DeleteAgent(ctx context.Context, ownerUserID string, agentID string) error
}

// ProviderGrantStore persists provider grants.
type ProviderGrantStore interface {
	PutProviderGrant(ctx context.Context, grant providergrant.ProviderGrant) error
	GetProviderGrant(ctx context.Context, providerGrantID string) (providergrant.ProviderGrant, error)
	ListProviderGrantsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter providergrant.Filter) (providergrant.Page, error)
}

// ProviderConnectSessionStore persists connect-session records.
type ProviderConnectSessionStore interface {
	PutProviderConnectSession(ctx context.Context, record ProviderConnectSessionRecord) error
	GetProviderConnectSession(ctx context.Context, connectSessionID string) (ProviderConnectSessionRecord, error)
	CompleteProviderConnectSession(ctx context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error
}

// AccessRequestStore persists agent access-request records.
type AccessRequestStore interface {
	PutAccessRequest(ctx context.Context, request accessrequest.AccessRequest) error
	GetAccessRequest(ctx context.Context, accessRequestID string) (accessrequest.AccessRequest, error)
	ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (accessrequest.Page, error)
	ListAccessRequestsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (accessrequest.Page, error)
	// GetApprovedInvokeAccessByRequesterForAgent returns one approved invoke grant
	// for a requester/owner/agent tuple. Callers use this to authorize a single
	// invoke decision without scanning unrelated access requests.
	GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (accessrequest.AccessRequest, error)
	// ListApprovedInvokeAccessRequestsByRequester returns only approved invoke
	// access rows for one requester. This narrows list-accessible authorization
	// scans to relevant records.
	ListApprovedInvokeAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (accessrequest.Page, error)
	// ReviewAccessRequest applies an owner review decision for one pending
	// request. The storage layer extracts status, reviewer, and timestamp
	// fields from the domain object and performs a CAS update against the
	// current pending status.
	ReviewAccessRequest(ctx context.Context, reviewed accessrequest.AccessRequest) error
	// RevokeAccessRequest applies an owner revocation for one approved
	// request. Same CAS pattern as ReviewAccessRequest.
	RevokeAccessRequest(ctx context.Context, revoked accessrequest.AccessRequest) error
}

// AuditEventStore persists append-only AI audit events.
type AuditEventStore interface {
	PutAuditEvent(ctx context.Context, record AuditEventRecord) error
	ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter AuditEventFilter) (AuditEventPage, error)
}

// CampaignArtifactStore persists campaign-scoped GM working artifacts.
type CampaignArtifactStore interface {
	PutCampaignArtifact(ctx context.Context, record CampaignArtifactRecord) error
	GetCampaignArtifact(ctx context.Context, campaignID string, path string) (CampaignArtifactRecord, error)
	ListCampaignArtifacts(ctx context.Context, campaignID string) ([]CampaignArtifactRecord, error)
}

// Store embeds all AI storage interfaces so composition roots and integration
// tests can accept a single concrete store instead of enumerating every
// sub-interface. Handler-level code should continue accepting narrow interfaces.
type Store interface {
	AgentStore
	CredentialStore
	ProviderGrantStore
	ProviderConnectSessionStore
	AccessRequestStore
	AuditEventStore
	CampaignArtifactStore
}
