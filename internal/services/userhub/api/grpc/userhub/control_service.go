package userhub

import (
	"context"
	"strings"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// dashboardInvalidator is the cache-control behavior surface consumed by transport.
type dashboardInvalidator interface {
	InvalidateDashboards(context.Context, domain.InvalidateDashboardsInput) (domain.InvalidateDashboardsResult, error)
}

// ControlService exposes trusted internal cache invalidation endpoints.
type ControlService struct {
	userhubv1.UnimplementedUserHubControlServiceServer
	domain dashboardInvalidator
}

// NewControlService constructs a userhub cache control service.
func NewControlService(domainSvc dashboardInvalidator) *ControlService {
	return &ControlService{domain: domainSvc}
}

// InvalidateDashboards removes cached dashboard entries for users and/or campaigns.
func (s *ControlService) InvalidateDashboards(ctx context.Context, in *userhubv1.InvalidateDashboardsRequest) (*userhubv1.InvalidateDashboardsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "invalidate dashboards request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "userhub invalidation service is not configured")
	}

	hasUser := false
	for _, userID := range in.GetUserIds() {
		if strings.TrimSpace(userID) != "" {
			hasUser = true
			break
		}
	}
	hasCampaign := false
	for _, campaignID := range in.GetCampaignIds() {
		if strings.TrimSpace(campaignID) != "" {
			hasCampaign = true
			break
		}
	}
	if !hasUser && !hasCampaign {
		return nil, status.Error(codes.InvalidArgument, "at least one user_id or campaign_id is required")
	}

	result, err := s.domain.InvalidateDashboards(ctx, domain.InvalidateDashboardsInput{
		UserIDs:     append([]string{}, in.GetUserIds()...),
		CampaignIDs: append([]string{}, in.GetCampaignIds()...),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		if err == domain.ErrServiceNotConfigured {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "invalidate dashboards: %v", err)
	}
	return &userhubv1.InvalidateDashboardsResponse{
		InvalidatedEntries: int32(result.InvalidatedEntries),
	}, nil
}
