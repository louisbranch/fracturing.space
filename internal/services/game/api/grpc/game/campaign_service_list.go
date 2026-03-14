package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListCampaigns returns a page of campaign metadata records.
// Admin override requests are allowed to enumerate campaigns without participant scope.
// Non-admin calls remain participant/user scoped and only return member campaigns.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignsPageSize,
		Max:     maxListCampaignsPageSize,
	})
	statusFilter := requestedCampaignStatuses(in.GetStatuses())

	participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	overrideReason, overrideRequested := adminOverrideFromContext(ctx)

	if overrideRequested {
		if userID == "" {
			err := status.Error(codes.PermissionDenied, "admin override requires authenticated principal")
			emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
				Store:      s.stores.Audit,
				Capability: domainauthz.CapabilityReadCampaign,
				Decision:   authzDecisionDeny,
				ReasonCode: authzReasonDenyMissingIdentity,
				Err:        err,
			})
			return nil, err
		}
		if overrideReason == "" {
			err := status.Error(codes.PermissionDenied, "admin override reason is required")
			emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
				Store:      s.stores.Audit,
				Capability: domainauthz.CapabilityReadCampaign,
				Decision:   authzDecisionDeny,
				ReasonCode: authzReasonDenyOverrideReasonRequired,
				Err:        err,
			})
			return nil, err
		}
		emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
			Store:      s.stores.Audit,
			Capability: domainauthz.CapabilityReadCampaign,
			Decision:   authzDecisionAllow,
			ReasonCode: authzReasonAllowAdminOverride,
		})
		campaignRecords, nextPageToken, err := s.listCampaignsFromStore(ctx, pageSize, in.GetPageToken(), statusFilter)
		if err != nil {
			return nil, err
		}
		response := &campaignv1.ListCampaignsResponse{NextPageToken: nextPageToken}
		if len(campaignRecords) == 0 {
			return response, nil
		}
		response.Campaigns = make([]*campaignv1.Campaign, 0, len(campaignRecords))
		for _, c := range campaignRecords {
			response.Campaigns = append(response.Campaigns, campaignToProto(c))
		}
		return response, nil
	}

	if participantID == "" && userID == "" {
		err := status.Error(codes.PermissionDenied, "missing participant identity")
		emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
			Store:      s.stores.Audit,
			Capability: domainauthz.CapabilityReadCampaign,
			Decision:   authzDecisionDeny,
			ReasonCode: authzReasonDenyMissingIdentity,
			Err:        err,
		})
		return nil, err
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	nextPageToken := ""
	var err error
	if participantID != "" {
		campaignRecords, nextPageToken, err = s.listCampaignsForParticipant(ctx, participantID, pageSize, in.GetPageToken(), statusFilter)
	} else {
		campaignRecords, nextPageToken, err = s.listCampaignsForUser(ctx, userID, pageSize, in.GetPageToken(), statusFilter)
	}
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListCampaignsResponse{
		NextPageToken: nextPageToken,
	}
	if len(campaignRecords) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignv1.Campaign, 0, len(campaignRecords))
	for _, c := range campaignRecords {
		response.Campaigns = append(response.Campaigns, campaignToProto(c))
	}

	return response, nil
}

func (s *CampaignService) listCampaignsFromStore(ctx context.Context, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	if len(statusFilter) == 0 {
		page, err := s.stores.Campaign.List(ctx, pageSize, pageToken)
		if err != nil {
			return nil, "", grpcerror.Internal("list campaigns", err)
		}
		return page.Campaigns, page.NextPageToken, nil
	}

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	scanToken := pageToken
	for len(campaignRecords) < pageSize {
		page, err := s.stores.Campaign.List(ctx, pageSize, scanToken)
		if err != nil {
			return nil, "", grpcerror.Internal("list campaigns", err)
		}
		if len(page.Campaigns) == 0 {
			return campaignRecords, "", nil
		}

		for idx, record := range page.Campaigns {
			if !campaignRecordMatchesStatuses(record, statusFilter) {
				continue
			}
			campaignRecords = append(campaignRecords, record)
			if len(campaignRecords) == pageSize {
				hasMoreInPage := idx+1 < len(page.Campaigns)
				if hasMoreInPage || page.NextPageToken != "" {
					return campaignRecords, record.ID, nil
				}
				return campaignRecords, "", nil
			}
		}

		if page.NextPageToken == "" {
			break
		}
		scanToken = page.NextPageToken
	}

	return campaignRecords, "", nil
}

func (s *CampaignService) listCampaignsForUser(ctx context.Context, userID string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	userID = strings.TrimSpace(userID)
	campaignIDs, err := s.stores.Participant.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return nil, "", grpcerror.Internal("list campaign IDs by user", err)
	}
	return s.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken, statusFilter)
}

func (s *CampaignService) listCampaignsForParticipant(ctx context.Context, participantID string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	participantID = strings.TrimSpace(participantID)
	campaignIDs, err := s.stores.Participant.ListCampaignIDsByParticipant(ctx, participantID)
	if err != nil {
		return nil, "", grpcerror.Internal("list campaign IDs by participant", err)
	}
	return s.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken, statusFilter)
}

// listCampaignsByIDs paginates a pre-resolved list of campaign IDs and fetches
// the corresponding records, skipping any that have been deleted.
func (s *CampaignService) listCampaignsByIDs(ctx context.Context, campaignIDs []string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	if len(campaignIDs) == 0 {
		return nil, "", nil
	}

	start := 0
	if pageToken != "" {
		for idx, campaignID := range campaignIDs {
			if strings.TrimSpace(campaignID) == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start < 0 || start >= len(campaignIDs) {
		start = 0
	}

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	nextPageToken := ""
	for idx := start; idx < len(campaignIDs); idx++ {
		campaignID := campaignIDs[idx]
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}
		record, err := s.stores.Campaign.Get(ctx, campaignID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, "", grpcerror.Internal("get campaign", err)
		}
		if !campaignRecordMatchesStatuses(record, statusFilter) {
			continue
		}
		campaignRecords = append(campaignRecords, record)
		if len(campaignRecords) == pageSize {
			if idx+1 < len(campaignIDs) {
				nextPageToken = campaignID
			}
			break
		}
	}

	return campaignRecords, nextPageToken, nil
}

func requestedCampaignStatuses(values []campaignv1.CampaignStatus) map[campaign.Status]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[campaign.Status]struct{}, len(values))
	for _, value := range values {
		status, ok := campaignStatusFromProto(value)
		if !ok {
			continue
		}
		result[status] = struct{}{}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func campaignStatusMatchesFilter(status campaign.Status, allowed map[campaign.Status]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[status]
	return ok
}

func campaignRecordMatchesStatuses(record storage.CampaignRecord, allowed map[campaign.Status]struct{}) bool {
	return campaignStatusMatchesFilter(record.Status, allowed)
}

func campaignStatusFromProto(value campaignv1.CampaignStatus) (campaign.Status, bool) {
	switch value {
	case campaignv1.CampaignStatus_DRAFT:
		return campaign.StatusDraft, true
	case campaignv1.CampaignStatus_ACTIVE:
		return campaign.StatusActive, true
	case campaignv1.CampaignStatus_COMPLETED:
		return campaign.StatusCompleted, true
	case campaignv1.CampaignStatus_ARCHIVED:
		return campaign.StatusArchived, true
	default:
		return campaign.StatusUnspecified, false
	}
}
