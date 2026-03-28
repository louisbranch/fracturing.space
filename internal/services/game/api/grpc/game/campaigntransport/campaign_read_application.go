package campaigntransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignListPage struct {
	campaigns     []storage.CampaignRecord
	nextPageToken string
}

func (c campaignApplication) GetCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	record, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequireReadPolicy(ctx, c.auth, record); err != nil {
		return storage.CampaignRecord{}, err
	}
	return record, nil
}

func (c campaignApplication) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (campaignListPage, error) {
	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignsPageSize,
		Max:     maxListCampaignsPageSize,
	})
	statusFilter := requestedCampaignStatuses(in.GetStatuses())

	participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	overrideReason, overrideRequested := authz.AdminOverrideFromContext(ctx)

	if overrideRequested {
		if userID == "" {
			err := status.Error(codes.PermissionDenied, "admin override requires authenticated principal")
			authz.EmitDecisionTelemetry(ctx, authz.DecisionEvent{
				Store:      c.auth.Audit,
				Capability: domainauthz.CapabilityReadCampaign(),
				Decision:   authz.DecisionDeny,
				ReasonCode: authz.ReasonDenyMissingIdentity,
				Err:        err,
			})
			return campaignListPage{}, err
		}
		if overrideReason == "" {
			err := status.Error(codes.PermissionDenied, "admin override reason is required")
			authz.EmitDecisionTelemetry(ctx, authz.DecisionEvent{
				Store:      c.auth.Audit,
				Capability: domainauthz.CapabilityReadCampaign(),
				Decision:   authz.DecisionDeny,
				ReasonCode: authz.ReasonDenyOverrideReasonRequired,
				Err:        err,
			})
			return campaignListPage{}, err
		}
		authz.EmitDecisionTelemetry(ctx, authz.DecisionEvent{
			Store:      c.auth.Audit,
			Capability: domainauthz.CapabilityReadCampaign(),
			Decision:   authz.DecisionAllow,
			ReasonCode: authz.ReasonAllowAdminOverride,
		})
		campaignRecords, nextPageToken, err := c.listCampaignsFromStore(ctx, pageSize, in.GetPageToken(), statusFilter)
		if err != nil {
			return campaignListPage{}, err
		}
		return campaignListPage{campaigns: campaignRecords, nextPageToken: nextPageToken}, nil
	}

	if participantID == "" && userID == "" {
		err := status.Error(codes.PermissionDenied, "missing participant identity")
		authz.EmitDecisionTelemetry(ctx, authz.DecisionEvent{
			Store:      c.auth.Audit,
			Capability: domainauthz.CapabilityReadCampaign(),
			Decision:   authz.DecisionDeny,
			ReasonCode: authz.ReasonDenyMissingIdentity,
			Err:        err,
		})
		return campaignListPage{}, err
	}
	if c.stores.Participant == nil {
		return campaignListPage{}, status.Error(codes.Internal, "participant store is not configured")
	}

	var (
		campaignRecords []storage.CampaignRecord
		nextPageToken   string
		err             error
	)
	if participantID != "" {
		campaignRecords, nextPageToken, err = c.listCampaignsForParticipant(ctx, participantID, pageSize, in.GetPageToken(), statusFilter)
	} else {
		campaignRecords, nextPageToken, err = c.listCampaignsForUser(ctx, userID, pageSize, in.GetPageToken(), statusFilter)
	}
	if err != nil {
		return campaignListPage{}, err
	}
	return campaignListPage{campaigns: campaignRecords, nextPageToken: nextPageToken}, nil
}

func (c campaignApplication) listCampaignsFromStore(ctx context.Context, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	if len(statusFilter) == 0 {
		page, err := c.stores.Campaign.List(ctx, pageSize, pageToken)
		if err != nil {
			return nil, "", grpcerror.Internal("list campaigns", err)
		}
		return page.Campaigns, page.NextPageToken, nil
	}

	campaignRecords := make([]storage.CampaignRecord, 0, pageSize)
	scanToken := pageToken
	for len(campaignRecords) < pageSize {
		page, err := c.stores.Campaign.List(ctx, pageSize, scanToken)
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

func (c campaignApplication) listCampaignsForUser(ctx context.Context, userID string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	userID = strings.TrimSpace(userID)
	campaignIDs, err := c.stores.Participant.ListCampaignIDsByUser(ctx, userID)
	if err != nil {
		return nil, "", grpcerror.Internal("list campaign IDs by user", err)
	}
	return c.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken, statusFilter)
}

func (c campaignApplication) listCampaignsForParticipant(ctx context.Context, participantID string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
	participantID = strings.TrimSpace(participantID)
	campaignIDs, err := c.stores.Participant.ListCampaignIDsByParticipant(ctx, participantID)
	if err != nil {
		return nil, "", grpcerror.Internal("list campaign IDs by participant", err)
	}
	return c.listCampaignsByIDs(ctx, campaignIDs, pageSize, pageToken, statusFilter)
}

func (c campaignApplication) listCampaignsByIDs(ctx context.Context, campaignIDs []string, pageSize int, pageToken string, statusFilter map[campaign.Status]struct{}) ([]storage.CampaignRecord, string, error) {
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
		campaignID := strings.TrimSpace(campaignIDs[idx])
		if campaignID == "" {
			continue
		}
		record, err := c.stores.Campaign.Get(ctx, campaignID)
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get campaign"); lookupErr != nil {
			return nil, "", lookupErr
		}
		if err != nil {
			continue
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
