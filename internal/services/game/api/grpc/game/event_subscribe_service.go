package game

import (
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultCampaignUpdatePollInterval = time.Second
	minCampaignUpdatePollInterval     = 50 * time.Millisecond
)

type normalizedSubscribeCampaignUpdatesRequest struct {
	campaignID            string
	afterSeq              uint64
	includeEventCommitted bool
	includeProjection     bool
	projectionScopes      map[string]struct{}
	pollInterval          time.Duration
}

// SubscribeCampaignUpdates streams campaign update envelopes for realtime clients.
func (s *EventService) SubscribeCampaignUpdates(in *campaignv1.SubscribeCampaignUpdatesRequest, stream grpc.ServerStreamingServer[campaignv1.CampaignUpdate]) error {
	if in == nil {
		return status.Error(codes.InvalidArgument, "subscribe campaign updates request is required")
	}
	return s.app.SubscribeCampaignUpdates(in, stream)
}

func normalizeSubscribeCampaignUpdatesRequest(in *campaignv1.SubscribeCampaignUpdatesRequest) (normalizedSubscribeCampaignUpdatesRequest, error) {
	if in == nil {
		return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	includeEventCommitted := true
	includeProjection := true
	if len(in.GetKinds()) > 0 {
		includeEventCommitted = false
		includeProjection = false
		for _, kind := range in.GetKinds() {
			switch kind {
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_EVENT_COMMITTED:
				includeEventCommitted = true
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED:
				includeProjection = true
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_UNSPECIFIED:
				// ignore
			default:
				return normalizedSubscribeCampaignUpdatesRequest{}, status.Errorf(codes.InvalidArgument, "invalid update kind: %d", kind)
			}
		}
		if !includeEventCommitted && !includeProjection {
			return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "at least one update kind is required")
		}
	}

	pollInterval := defaultCampaignUpdatePollInterval
	if in.GetPollIntervalMs() > 0 {
		pollInterval = time.Duration(in.GetPollIntervalMs()) * time.Millisecond
	}
	if pollInterval < minCampaignUpdatePollInterval {
		pollInterval = minCampaignUpdatePollInterval
	}

	var projectionScopes map[string]struct{}
	if len(in.GetProjectionScopes()) > 0 {
		projectionScopes = make(map[string]struct{}, len(in.GetProjectionScopes()))
		for _, scope := range in.GetProjectionScopes() {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			projectionScopes[scope] = struct{}{}
		}
	}

	return normalizedSubscribeCampaignUpdatesRequest{
		campaignID:            campaignID,
		afterSeq:              in.GetAfterSeq(),
		includeEventCommitted: includeEventCommitted,
		includeProjection:     includeProjection,
		projectionScopes:      projectionScopes,
		pollInterval:          pollInterval,
	}, nil
}
