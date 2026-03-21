package ai

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(userIDHeader)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func providerFromProto(value aiv1.Provider) (provider.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return provider.OpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func providerFromString(value string) provider.Provider {
	normalized, err := provider.Normalize(value)
	if err != nil {
		return ""
	}
	return normalized
}

func providerToProto(value string) aiv1.Provider {
	if providerFromString(value) == provider.OpenAI {
		return aiv1.Provider_PROVIDER_OPENAI
	}
	return aiv1.Provider_PROVIDER_UNSPECIFIED
}

func usageToProto(value provider.Usage) *aiv1.Usage {
	if value.IsZero() {
		return nil
	}
	return &aiv1.Usage{
		InputTokens:     value.InputTokens,
		OutputTokens:    value.OutputTokens,
		ReasoningTokens: value.ReasoningTokens,
		TotalTokens:     value.TotalTokens,
	}
}

func credentialStatusToProto(value credential.Status) aiv1.CredentialStatus {
	switch value {
	case credential.StatusActive:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
	case credential.StatusRevoked:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED
	default:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED
	}
}

func agentStatusToProto(value agent.Status) aiv1.AgentStatus {
	if value.IsActive() {
		return aiv1.AgentStatus_AGENT_STATUS_ACTIVE
	}
	return aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED
}

func providerGrantStatusToProto(value providergrant.Status) aiv1.ProviderGrantStatus {
	switch value {
	case providergrant.StatusActive:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE
	case providergrant.StatusRevoked:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED
	case providergrant.StatusExpired:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED
	case providergrant.StatusRefreshFailed:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED
	default:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED
	}
}

func accessRequestStatusToProto(value accessrequest.Status) aiv1.AccessRequestStatus {
	switch value {
	case accessrequest.StatusPending:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_PENDING
	case accessrequest.StatusApproved:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED
	case accessrequest.StatusDenied:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_DENIED
	case accessrequest.StatusRevoked:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED
	default:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_UNSPECIFIED
	}
}

func accessRequestDecisionFromProto(value aiv1.AccessRequestDecision) (accessrequest.Decision, error) {
	switch value {
	case aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE:
		return accessrequest.DecisionApprove, nil
	case aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_DENY:
		return accessrequest.DecisionDeny, nil
	default:
		return "", status.Error(codes.InvalidArgument, "decision is required")
	}
}

func credentialToProto(c credential.Credential) *aiv1.Credential {
	// Intentionally omits SecretCiphertext to avoid exposing encrypted credential
	// material over read APIs.
	proto := &aiv1.Credential{
		Id:          c.ID,
		OwnerUserId: c.OwnerUserID,
		Provider:    providerToProto(string(c.Provider)),
		Label:       c.Label,
		Status:      credentialStatusToProto(c.Status),
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
	if c.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*c.RevokedAt)
	}
	return proto
}

func agentToProto(a agent.Agent) *aiv1.Agent {
	return &aiv1.Agent{
		Id:              a.ID,
		OwnerUserId:     a.OwnerUserID,
		Label:           a.Label,
		Instructions:    a.Instructions,
		Provider:        providerToProto(string(a.Provider)),
		Model:           a.Model,
		CredentialId:    a.AuthReference.CredentialID(),
		ProviderGrantId: a.AuthReference.ProviderGrantID(),
		Status:          agentStatusToProto(a.Status),
		CreatedAt:       timestamppb.New(a.CreatedAt),
		UpdatedAt:       timestamppb.New(a.UpdatedAt),
	}
}

func providerGrantToProto(grant providergrant.ProviderGrant) *aiv1.ProviderGrant {
	proto := &aiv1.ProviderGrant{
		Id:               grant.ID,
		OwnerUserId:      grant.OwnerUserID,
		Provider:         providerToProto(string(grant.Provider)),
		GrantedScopes:    append([]string(nil), grant.GrantedScopes...),
		RefreshSupported: grant.RefreshSupported,
		Status:           providerGrantStatusToProto(grant.Status),
		LastRefreshError: grant.LastRefreshError,
		CreatedAt:        timestamppb.New(grant.CreatedAt),
		UpdatedAt:        timestamppb.New(grant.UpdatedAt),
	}
	if grant.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*grant.RevokedAt)
	}
	if grant.ExpiresAt != nil {
		proto.ExpiresAt = timestamppb.New(*grant.ExpiresAt)
	}
	if grant.RefreshedAt != nil {
		proto.LastRefreshedAt = timestamppb.New(*grant.RefreshedAt)
	}
	return proto
}

func campaignArtifactToProto(record storage.CampaignArtifactRecord) *aiv1.CampaignArtifact {
	return &aiv1.CampaignArtifact{
		CampaignId: record.CampaignID,
		Path:       record.Path,
		Content:    record.Content,
		ReadOnly:   record.ReadOnly,
		CreatedAt:  timestamppb.New(record.CreatedAt),
		UpdatedAt:  timestamppb.New(record.UpdatedAt),
	}
}

func referenceDocumentSummaryToProto(result referencecorpus.SearchResult) *aiv1.SystemReferenceDocumentSummary {
	return &aiv1.SystemReferenceDocumentSummary{
		System:     result.System,
		DocumentId: result.DocumentID,
		Title:      result.Title,
		Kind:       result.Kind,
		Path:       result.Path,
		Aliases:    append([]string(nil), result.Aliases...),
		Snippet:    result.Snippet,
	}
}

func referenceDocumentToProto(document referencecorpus.Document) *aiv1.SystemReferenceDocument {
	return &aiv1.SystemReferenceDocument{
		System:     document.System,
		DocumentId: document.DocumentID,
		Title:      document.Title,
		Kind:       document.Kind,
		Path:       document.Path,
		Aliases:    append([]string(nil), document.Aliases...),
		Content:    document.Content,
	}
}

func accessRequestToProto(ar accessrequest.AccessRequest) *aiv1.AccessRequest {
	proto := &aiv1.AccessRequest{
		Id:              ar.ID,
		RequesterUserId: ar.RequesterUserID,
		OwnerUserId:     ar.OwnerUserID,
		AgentId:         ar.AgentID,
		Scope:           string(ar.Scope),
		RequestNote:     ar.RequestNote,
		Status:          accessRequestStatusToProto(ar.Status),
		ReviewerUserId:  ar.ReviewerUserID,
		ReviewNote:      ar.ReviewNote,
		CreatedAt:       timestamppb.New(ar.CreatedAt),
		UpdatedAt:       timestamppb.New(ar.UpdatedAt),
	}
	if ar.ReviewedAt != nil {
		proto.ReviewedAt = timestamppb.New(*ar.ReviewedAt)
	}
	return proto
}

func auditEventToProto(record storage.AuditEventRecord) *aiv1.AuditEvent {
	return &aiv1.AuditEvent{
		Id:              record.ID,
		EventName:       record.EventName,
		ActorUserId:     record.ActorUserID,
		OwnerUserId:     record.OwnerUserID,
		RequesterUserId: record.RequesterUserID,
		AgentId:         record.AgentID,
		AccessRequestId: record.AccessRequestID,
		Outcome:         record.Outcome,
		CreatedAt:       timestamppb.New(record.CreatedAt),
	}
}

func clampPageSize(requested int32) int {
	if requested <= 0 {
		return defaultPageSize
	}
	if requested > maxPageSize {
		return maxPageSize
	}
	return int(requested)
}
