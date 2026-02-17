// Package accessrequest models owner-gated approval of AI agent invocations.
//
// It provides a narrow approval workflow so runtime invocation can stay logged and
// controlled rather than automatically inheriting user ownership.
package accessrequest

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// Scope represents a requested usage permission.
type Scope string

const (
	// ScopeInvoke allows runtime invocation of a target AI agent.
	ScopeInvoke Scope = "invoke"
)

// Status represents access-request lifecycle state.
type Status string

const (
	// StatusPending indicates request is awaiting owner review.
	StatusPending Status = "pending"
	// StatusApproved indicates owner accepted the request.
	StatusApproved Status = "approved"
	// StatusDenied indicates owner denied the request.
	StatusDenied Status = "denied"
	// StatusRevoked indicates owner removed previously approved access.
	StatusRevoked Status = "revoked"
)

// Decision represents a review action taken by an owner.
type Decision string

const (
	// DecisionApprove accepts a pending request.
	DecisionApprove Decision = "approve"
	// DecisionDeny rejects a pending request.
	DecisionDeny Decision = "deny"
)

var (
	// ErrEmptyID indicates an ID is required.
	ErrEmptyID = errors.New("id is required")
	// ErrEmptyRequesterUserID indicates requester user ID is required.
	ErrEmptyRequesterUserID = errors.New("requester user id is required")
	// ErrEmptyOwnerUserID indicates owner user ID is required.
	ErrEmptyOwnerUserID = errors.New("owner user id is required")
	// ErrEmptyAgentID indicates target agent ID is required.
	ErrEmptyAgentID = errors.New("agent id is required")
	// ErrInvalidScope indicates the requested scope is unsupported.
	ErrInvalidScope = errors.New("scope is invalid")
	// ErrRequesterIsOwner indicates owner and requester must differ.
	ErrRequesterIsOwner = errors.New("requester cannot request access to own agent")
	// ErrEmptyReviewerUserID indicates reviewer user ID is required.
	ErrEmptyReviewerUserID = errors.New("reviewer user id is required")
	// ErrInvalidDecision indicates review decision must be approve/deny.
	ErrInvalidDecision = errors.New("decision is invalid")
	// ErrNotPending indicates reviewed requests are immutable.
	ErrNotPending = errors.New("access request is not pending")
	// ErrReviewerNotOwner indicates only owners may review requests.
	ErrReviewerNotOwner = errors.New("reviewer must match owner")
	// ErrNotApproved indicates only approved requests may be revoked.
	ErrNotApproved = errors.New("access request is not approved")
)

// AccessRequest stores owner-mediated access intent for one target agent.
type AccessRequest struct {
	ID string

	RequesterUserID string
	OwnerUserID     string
	AgentID         string
	Scope           Scope

	RequestNote string

	Status Status

	ReviewerUserID string
	ReviewNote     string

	CreatedAt  time.Time
	UpdatedAt  time.Time
	ReviewedAt *time.Time
}

// CreateInput contains requester-provided fields for request creation.
type CreateInput struct {
	RequesterUserID string
	OwnerUserID     string
	AgentID         string
	Scope           Scope
	RequestNote     string
}

// ReviewInput contains fields required to review one access request.
type ReviewInput struct {
	ID             string
	ReviewerUserID string
	Decision       Decision
	ReviewNote     string
}

// RevokeInput contains fields required to revoke one approved request.
type RevokeInput struct {
	ID            string
	RevokerUserID string
	RevokeNote    string
}

// NormalizeCreateInput canonicalizes and validates request creation input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.RequesterUserID = strings.TrimSpace(input.RequesterUserID)
	if input.RequesterUserID == "" {
		return CreateInput{}, ErrEmptyRequesterUserID
	}

	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}
	if input.OwnerUserID == input.RequesterUserID {
		return CreateInput{}, ErrRequesterIsOwner
	}

	input.AgentID = strings.TrimSpace(input.AgentID)
	if input.AgentID == "" {
		return CreateInput{}, ErrEmptyAgentID
	}

	normalizedScope, err := normalizeScope(input.Scope)
	if err != nil {
		return CreateInput{}, err
	}
	input.Scope = normalizedScope
	input.RequestNote = strings.TrimSpace(input.RequestNote)

	return input, nil
}

// NormalizeReviewInput canonicalizes and validates review input.
func NormalizeReviewInput(input ReviewInput) (ReviewInput, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return ReviewInput{}, ErrEmptyID
	}

	input.ReviewerUserID = strings.TrimSpace(input.ReviewerUserID)
	if input.ReviewerUserID == "" {
		return ReviewInput{}, ErrEmptyReviewerUserID
	}

	input.Decision = Decision(strings.ToLower(strings.TrimSpace(string(input.Decision))))
	if input.Decision != DecisionApprove && input.Decision != DecisionDeny {
		return ReviewInput{}, ErrInvalidDecision
	}

	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	return input, nil
}

// Create constructs a normalized pending access request.
func Create(input CreateInput, now func() time.Time, idGenerator func() (string, error)) (AccessRequest, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateInput(input)
	if err != nil {
		return AccessRequest{}, err
	}

	requestID, err := idGenerator()
	if err != nil {
		return AccessRequest{}, fmt.Errorf("generate access request id: %w", err)
	}

	createdAt := now().UTC()
	return AccessRequest{
		ID:              requestID,
		RequesterUserID: normalized.RequesterUserID,
		OwnerUserID:     normalized.OwnerUserID,
		AgentID:         normalized.AgentID,
		Scope:           normalized.Scope,
		RequestNote:     normalized.RequestNote,
		Status:          StatusPending,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}, nil
}

// Review applies one owner review decision to a pending access request.
func Review(request AccessRequest, input ReviewInput, now func() time.Time) (AccessRequest, error) {
	if now == nil {
		now = time.Now
	}

	normalized, err := NormalizeReviewInput(input)
	if err != nil {
		return AccessRequest{}, err
	}

	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		return AccessRequest{}, ErrEmptyID
	}
	if request.Status != StatusPending {
		return AccessRequest{}, ErrNotPending
	}
	if strings.TrimSpace(request.OwnerUserID) != normalized.ReviewerUserID {
		return AccessRequest{}, ErrReviewerNotOwner
	}

	reviewedAt := now().UTC()
	request.ReviewerUserID = normalized.ReviewerUserID
	request.ReviewNote = normalized.ReviewNote
	request.ReviewedAt = &reviewedAt
	request.UpdatedAt = reviewedAt
	if normalized.Decision == DecisionApprove {
		request.Status = StatusApproved
	} else {
		request.Status = StatusDenied
	}
	return request, nil
}

// NormalizeRevokeInput canonicalizes and validates revoke input.
func NormalizeRevokeInput(input RevokeInput) (RevokeInput, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return RevokeInput{}, ErrEmptyID
	}

	input.RevokerUserID = strings.TrimSpace(input.RevokerUserID)
	if input.RevokerUserID == "" {
		return RevokeInput{}, ErrEmptyReviewerUserID
	}
	input.RevokeNote = strings.TrimSpace(input.RevokeNote)
	return input, nil
}

// Revoke applies one owner revocation to an approved request.
func Revoke(request AccessRequest, input RevokeInput, now func() time.Time) (AccessRequest, error) {
	if now == nil {
		now = time.Now
	}

	normalized, err := NormalizeRevokeInput(input)
	if err != nil {
		return AccessRequest{}, err
	}

	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		return AccessRequest{}, ErrEmptyID
	}
	if request.Status != StatusApproved {
		return AccessRequest{}, ErrNotApproved
	}
	if strings.TrimSpace(request.OwnerUserID) != normalized.RevokerUserID {
		return AccessRequest{}, ErrReviewerNotOwner
	}

	revokedAt := now().UTC()
	request.Status = StatusRevoked
	request.ReviewerUserID = normalized.RevokerUserID
	request.ReviewNote = normalized.RevokeNote
	request.UpdatedAt = revokedAt
	return request, nil
}

func normalizeScope(scope Scope) (Scope, error) {
	normalized := Scope(strings.ToLower(strings.TrimSpace(string(scope))))
	if normalized == "" {
		return ScopeInvoke, nil
	}
	if normalized != ScopeInvoke {
		return "", ErrInvalidScope
	}
	return normalized, nil
}
