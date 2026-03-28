package ai

import (
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProtoHelpersStatusAndAuthReferenceConversions(t *testing.T) {
	t.Parallel()

	if got := credentialStatusToProto(credential.StatusActive); got != aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE {
		t.Fatalf("credentialStatusToProto(active) = %v", got)
	}
	if got := credentialStatusToProto(credential.StatusRevoked); got != aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED {
		t.Fatalf("credentialStatusToProto(revoked) = %v", got)
	}
	if got := credentialStatusToProto(""); got != aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED {
		t.Fatalf("credentialStatusToProto(unspecified) = %v", got)
	}

	if got := providerGrantStatusToProto(providergrant.StatusActive); got != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE {
		t.Fatalf("providerGrantStatusToProto(active) = %v", got)
	}
	if got := providerGrantStatusToProto(providergrant.StatusRefreshFailed); got != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED {
		t.Fatalf("providerGrantStatusToProto(refresh_failed) = %v", got)
	}
	if got := providerGrantStatusToProto(""); got != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED {
		t.Fatalf("providerGrantStatusToProto(unspecified) = %v", got)
	}

	if got := accessRequestStatusToProto(accessrequest.StatusApproved); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED {
		t.Fatalf("accessRequestStatusToProto(approved) = %v", got)
	}
	if got := accessRequestStatusToProto(accessrequest.StatusRevoked); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED {
		t.Fatalf("accessRequestStatusToProto(revoked) = %v", got)
	}
	if got := accessRequestStatusToProto(""); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_UNSPECIFIED {
		t.Fatalf("accessRequestStatusToProto(unspecified) = %v", got)
	}

	kind, err := agentAuthReferenceTypeFromProto(aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_CREDENTIAL)
	if err != nil || kind != agent.AuthReferenceKindCredential {
		t.Fatalf("agentAuthReferenceTypeFromProto(credential) = (%q, %v)", kind, err)
	}
	_, err = agentAuthReferenceTypeFromProto(aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_UNSPECIFIED)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("agentAuthReferenceTypeFromProto(unspecified) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	ref, err := agentAuthReferenceFromProto(&aiv1.AgentAuthReference{
		Type: aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_PROVIDER_GRANT,
		Id:   "grant-1",
	}, true)
	if err != nil {
		t.Fatalf("agentAuthReferenceFromProto() error = %v", err)
	}
	if ref.ProviderGrantID() != "grant-1" {
		t.Fatalf("ProviderGrantID() = %q, want %q", ref.ProviderGrantID(), "grant-1")
	}
	if _, err := agentAuthReferenceFromProto(nil, true); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("agentAuthReferenceFromProto(nil, true) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if _, err := agentAuthReferenceFromProto(&aiv1.AgentAuthReference{}, true); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("agentAuthReferenceFromProto(empty, true) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestProtoHelpersRecordMappings(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC)
	revokedAt := now.Add(5 * time.Minute)
	refreshedAt := now.Add(10 * time.Minute)
	expiresAt := now.Add(time.Hour)
	reviewedAt := now.Add(15 * time.Minute)

	credentialProto := credentialToProto(credential.Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.Anthropic,
		Label:       "Claude",
		Status:      credential.StatusRevoked,
		CreatedAt:   now,
		UpdatedAt:   now,
		RevokedAt:   &revokedAt,
	})
	if credentialProto.GetProvider() != aiv1.Provider_PROVIDER_ANTHROPIC || credentialProto.GetRevokedAt() == nil {
		t.Fatalf("unexpected credential proto: %+v", credentialProto)
	}

	providerGrantProto := providerGrantToProto(providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		RefreshSupported: true,
		Status:           providergrant.StatusRefreshFailed,
		LastRefreshError: "expired",
		CreatedAt:        now,
		UpdatedAt:        now,
		RevokedAt:        &revokedAt,
		RefreshedAt:      &refreshedAt,
		ExpiresAt:        &expiresAt,
	})
	if providerGrantProto.GetStatus() != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED {
		t.Fatalf("providerGrant status = %v", providerGrantProto.GetStatus())
	}
	if providerGrantProto.GetRevokedAt() == nil || providerGrantProto.GetLastRefreshedAt() == nil || providerGrantProto.GetExpiresAt() == nil {
		t.Fatalf("unexpected provider grant proto timestamps: %+v", providerGrantProto)
	}

	accessRequestProto := accessRequestToProto(accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		RequestNote:     "please allow",
		Status:          accessrequest.StatusRevoked,
		ReviewerUserID:  "owner-1",
		ReviewNote:      "approved first",
		RevokerUserID:   "owner-1",
		RevokeNote:      "no longer needed",
		CreatedAt:       now,
		UpdatedAt:       now,
		ReviewedAt:      &reviewedAt,
		RevokedAt:       &revokedAt,
	})
	if accessRequestProto.GetStatus() != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED {
		t.Fatalf("access request status = %v", accessRequestProto.GetStatus())
	}
	if accessRequestProto.GetReviewedAt() == nil || accessRequestProto.GetRevokedAt() == nil {
		t.Fatalf("unexpected access request timestamps: %+v", accessRequestProto)
	}
}
