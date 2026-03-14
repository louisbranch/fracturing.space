package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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
	normalized, err := normalizeSubscribeCampaignUpdatesRequest(in)
	if err != nil {
		return err
	}
	if s == nil || s.stores.Event == nil {
		return status.Error(codes.Internal, "event store is not configured")
	}
	if stream == nil {
		return status.Error(codes.Internal, "stream is required")
	}

	ctx := stream.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := requireReadPolicy(ctx, s.stores, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return err
	}
	lastSeq := normalized.afterSeq

	sendAvailable := func() error {
		events, err := s.stores.Event.ListEvents(ctx, normalized.campaignID, lastSeq, maxListEventsPageSize)
		if err != nil {
			return grpcerror.Internal("list events", err)
		}

		for _, evt := range events {
			if normalized.includeEventCommitted {
				if err := stream.Send(campaignUpdateEventCommitted(evt)); err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return status.Errorf(codes.Unavailable, "send event_committed update: %v", err)
				}
			}

			if normalized.includeProjection {
				scopes := projectionScopesForEventType(string(evt.Type))
				if hasProjectionScopeIntersection(scopes, normalized.projectionScopes) {
					if err := stream.Send(campaignUpdateProjectionApplied(evt, scopes)); err != nil {
						if ctx.Err() != nil {
							return nil
						}
						return status.Errorf(codes.Unavailable, "send projection_applied update: %v", err)
					}
				}
			}

			if evt.Seq > lastSeq {
				lastSeq = evt.Seq
			}
		}
		return nil
	}

	if err := sendAvailable(); err != nil {
		return err
	}

	ticker := time.NewTicker(normalized.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sendAvailable(); err != nil {
				return err
			}
		}
	}
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
