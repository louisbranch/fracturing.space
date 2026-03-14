package ai

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ensureCredentialNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the credential is still bound to a DRAFT or ACTIVE campaign.
func (s *Service) ensureCredentialNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, credentialID string) error {
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return status.Error(codes.InvalidArgument, "credential_id is required")
	}
	return s.ensureAuthReferenceNotBoundToActiveCampaigns(ctx, ownerUserID, func(record storage.AgentRecord) bool {
		return strings.TrimSpace(record.CredentialID) == credentialID
	}, "credential is in use by active campaigns")
}

// ensureProviderGrantNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the provider grant is still bound to a DRAFT or ACTIVE campaign.
func (s *Service) ensureProviderGrantNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, providerGrantID string) error {
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return status.Error(codes.InvalidArgument, "provider_grant_id is required")
	}
	return s.ensureAuthReferenceNotBoundToActiveCampaigns(ctx, ownerUserID, func(record storage.AgentRecord) bool {
		return strings.TrimSpace(record.ProviderGrantID) == providerGrantID
	}, "provider grant is in use by active campaigns")
}

func (s *Service) ensureAuthReferenceNotBoundToActiveCampaigns(
	ctx context.Context,
	ownerUserID string,
	match func(storage.AgentRecord) bool,
	message string,
) error {
	if s == nil {
		return status.Error(codes.Internal, "service is not configured")
	}
	if s.agentStore == nil {
		return status.Error(codes.Internal, "agent store is not configured")
	}
	if match == nil {
		return nil
	}

	pageToken := ""
	for {
		page, err := s.agentStore.ListAgentsByOwner(ctx, ownerUserID, maxPageSize, pageToken)
		if err != nil {
			return status.Errorf(codes.Internal, "list agents: %v", err)
		}
		for _, record := range page.Agents {
			if !match(record) {
				continue
			}
			if err := s.ensureAgentNotBoundToActiveCampaigns(ctx, record.ID); err != nil {
				return errWithMessage(err, message)
			}
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			return nil
		}
		pageToken = nextPageToken
	}
}

func errWithMessage(err error, message string) error {
	if err == nil || strings.TrimSpace(message) == "" {
		return err
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	return status.Error(st.Code(), message)
}
