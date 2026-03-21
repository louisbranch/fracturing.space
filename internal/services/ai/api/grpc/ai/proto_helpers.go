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

func credentialStatusToProto(value string) aiv1.CredentialStatus {
	switch credential.ParseStatus(value) {
	case credential.StatusActive:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
	case credential.StatusRevoked:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED
	default:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED
	}
}

func agentStatusToProto(value string) aiv1.AgentStatus {
	switch agent.ParseStatus(value) {
	case agent.StatusActive:
		return aiv1.AgentStatus_AGENT_STATUS_ACTIVE
	default:
		return aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED
	}
}

func providerGrantStatusToProto(value string) aiv1.ProviderGrantStatus {
	switch providergrant.ParseStatus(value) {
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

func accessRequestStatusToProto(value string) aiv1.AccessRequestStatus {
	switch accessrequest.ParseStatus(value) {
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

func credentialToProto(record storage.CredentialRecord) *aiv1.Credential {
	// Intentionally omits SecretCiphertext to avoid exposing encrypted credential
	// material over read APIs.
	proto := &aiv1.Credential{
		Id:          record.ID,
		OwnerUserId: record.OwnerUserID,
		Provider:    providerToProto(record.Provider),
		Label:       record.Label,
		Status:      credentialStatusToProto(record.Status),
		CreatedAt:   timestamppb.New(record.CreatedAt),
		UpdatedAt:   timestamppb.New(record.UpdatedAt),
	}
	if record.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*record.RevokedAt)
	}
	return proto
}

func agentToProto(record storage.AgentRecord) *aiv1.Agent {
	return &aiv1.Agent{
		Id:              record.ID,
		OwnerUserId:     record.OwnerUserID,
		Label:           record.Label,
		Instructions:    record.Instructions,
		Provider:        providerToProto(record.Provider),
		Model:           record.Model,
		CredentialId:    record.CredentialID,
		ProviderGrantId: record.ProviderGrantID,
		Status:          agentStatusToProto(record.Status),
		CreatedAt:       timestamppb.New(record.CreatedAt),
		UpdatedAt:       timestamppb.New(record.UpdatedAt),
	}
}

func providerGrantToProto(record storage.ProviderGrantRecord) *aiv1.ProviderGrant {
	proto := &aiv1.ProviderGrant{
		Id:               record.ID,
		OwnerUserId:      record.OwnerUserID,
		Provider:         providerToProto(record.Provider),
		GrantedScopes:    append([]string(nil), record.GrantedScopes...),
		RefreshSupported: record.RefreshSupported,
		Status:           providerGrantStatusToProto(record.Status),
		LastRefreshError: record.LastRefreshError,
		CreatedAt:        timestamppb.New(record.CreatedAt),
		UpdatedAt:        timestamppb.New(record.UpdatedAt),
	}
	if record.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*record.RevokedAt)
	}
	if record.ExpiresAt != nil {
		proto.ExpiresAt = timestamppb.New(*record.ExpiresAt)
	}
	if record.LastRefreshedAt != nil {
		proto.LastRefreshedAt = timestamppb.New(*record.LastRefreshedAt)
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

func accessRequestToProto(record storage.AccessRequestRecord) *aiv1.AccessRequest {
	proto := &aiv1.AccessRequest{
		Id:              record.ID,
		RequesterUserId: record.RequesterUserID,
		OwnerUserId:     record.OwnerUserID,
		AgentId:         record.AgentID,
		Scope:           record.Scope,
		RequestNote:     record.RequestNote,
		Status:          accessRequestStatusToProto(record.Status),
		ReviewerUserId:  record.ReviewerUserID,
		ReviewNote:      record.ReviewNote,
		CreatedAt:       timestamppb.New(record.CreatedAt),
		UpdatedAt:       timestamppb.New(record.UpdatedAt),
	}
	if record.ReviewedAt != nil {
		proto.ReviewedAt = timestamppb.New(*record.ReviewedAt)
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
