package ai

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
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
func credentialProviderFromProto(value aiv1.Provider) (credential.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return credential.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func agentProviderFromProto(value aiv1.Provider) (agent.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return agent.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func providerGrantProviderFromProto(value aiv1.Provider) (providergrant.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return providergrant.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}
func providerToProto(value string) aiv1.Provider {
	if strings.EqualFold(strings.TrimSpace(value), "openai") {
		return aiv1.Provider_PROVIDER_OPENAI
	}
	return aiv1.Provider_PROVIDER_UNSPECIFIED
}

func credentialStatusToProto(value string) aiv1.CredentialStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
	case "revoked":
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED
	default:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED
	}
}

func agentStatusToProto(value string) aiv1.AgentStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.AgentStatus_AGENT_STATUS_ACTIVE
	default:
		return aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED
	}
}

func providerGrantStatusToProto(value string) aiv1.ProviderGrantStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE
	case "revoked":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED
	case "expired":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED
	case "refresh_failed":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED
	default:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED
	}
}

func accessRequestStatusToProto(value string) aiv1.AccessRequestStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_PENDING
	case "approved":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED
	case "denied":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_DENIED
	case "revoked":
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
		Name:            record.Name,
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
