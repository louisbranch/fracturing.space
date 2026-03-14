package game

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	aiAuthRotateReasonCampaignAIBound   = "campaign_ai_bound"
	aiAuthRotateReasonCampaignAIUnbound = "campaign_ai_unbound"
)

func rotateCampaignAIAuthEpoch(
	ctx context.Context,
	stores Stores,
	campaignID string,
	reason string,
	actorID string,
	actorType command.ActorType,
) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return status.Error(codes.InvalidArgument, "campaign id is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return status.Error(codes.InvalidArgument, "ai auth rotate reason is required")
	}

	payloadJSON, err := json.Marshal(campaign.AIAuthRotatePayload{Reason: reason})
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores.Write,
		stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignAIAuthRotate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return err
	}
	return nil
}
