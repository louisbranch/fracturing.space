package adversarytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// DeleteAdversary validates the current projection, emits the delete command,
// and returns the removed adversary snapshot.
func (h *Handler) DeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	if in == nil {
		return nil, invalidArgument("delete adversary request is required")
	}
	if err := h.requireBaseDependencies(); err != nil {
		return nil, err
	}
	if h.deps.ExecuteDomainCommand == nil {
		return nil, internal("domain command executor is not configured")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart adversaries"); err != nil {
		return nil, err
	}

	current, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	sessionID := strings.TrimSpace(current.SessionID)
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sessionID != "" {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.Gate, campaignID, sessionID); err != nil {
			return nil, err
		}
	}
	payloadJSON, err := json.Marshal(daggerheart.AdversaryDeletePayload{
		AdversaryID: ids.AdversaryID(adversaryID),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryDelete,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary delete did not emit an event",
		ApplyErrMessage: "apply adversary deleted event",
	}); err != nil {
		return nil, err
	}
	return &pb.DaggerheartDeleteAdversaryResponse{Adversary: adversaryToProto(current)}, nil
}
