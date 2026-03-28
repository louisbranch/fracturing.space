package forktransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type forkSourceState struct {
	campaign         storage.CampaignRecord
	originCampaignID string
}

func (a forkApplication) loadForkSourceState(ctx context.Context, sourceCampaignID string, copyParticipants bool) (forkSourceState, error) {
	if copyParticipants && a.stores.Participant == nil {
		return forkSourceState{}, status.Error(codes.Internal, "participant store is not configured")
	}

	sourceCampaign, err := a.stores.Campaign.Get(ctx, sourceCampaignID)
	if err != nil {
		return forkSourceState{}, grpcerror.EnsureStatus(err)
	}
	if sourceCampaign.AccessPolicy == campaign.AccessPolicyPublic {
		if !copyParticipants {
			return forkSourceState{}, status.Error(codes.FailedPrecondition, "public campaign forks must copy participants")
		}
		if strings.TrimSpace(grpcmeta.UserIDFromContext(ctx)) == "" {
			return forkSourceState{}, status.Error(codes.Unauthenticated, "authenticated user is required to fork public campaigns")
		}
	} else {
		if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageCampaign(), sourceCampaign); err != nil {
			return forkSourceState{}, err
		}
	}
	// Root startup validation guarantees Session in production wiring; keep a nil
	// guard so focused unit tests with partial stores remain supported.
	if a.stores.Session != nil {
		activeSession, err := a.stores.Session.GetActiveSession(ctx, sourceCampaignID)
		if err == nil {
			return forkSourceState{}, status.Errorf(
				codes.FailedPrecondition,
				"campaign has an active session: active_session_id=%s",
				activeSession.ID,
			)
		}
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "check active session"); lookupErr != nil {
			return forkSourceState{}, lookupErr
		}
	}

	sourceMetadata, err := a.stores.CampaignFork.GetCampaignForkMetadata(ctx, sourceCampaignID)
	if err != nil {
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get source fork metadata"); lookupErr != nil {
			return forkSourceState{}, lookupErr
		}
	}

	originCampaignID := sourceMetadata.OriginCampaignID
	if originCampaignID == "" {
		originCampaignID = sourceCampaignID
	}
	return forkSourceState{
		campaign:         sourceCampaign,
		originCampaignID: originCampaignID,
	}, nil
}

func (a forkApplication) buildLineage(ctx context.Context, campaignID, parentCampaignID, originCampaignID string, forkEventSeq uint64) *campaignv1.Lineage {
	depth := calculateDepth(ctx, a.stores.CampaignFork, parentCampaignID) + 1
	return &campaignv1.Lineage{
		CampaignId:       campaignID,
		ParentCampaignId: parentCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignId: originCampaignID,
		Depth:            int32(depth),
	}
}
