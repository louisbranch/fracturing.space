package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplySubclassFeature(ctx context.Context, in *pb.DaggerheartApplySubclassFeatureRequest) (*pb.DaggerheartApplySubclassFeatureResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply subclass feature request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}

	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "subclass feature")
	if err != nil {
		return nil, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}

	classState := classStateFromProjection(state.ClassState)
	subclassState := subclassStateFromProjection(state.SubclassState)
	payload, err := h.resolveSubclassFeaturePayload(ctx, campaignID, profile, state, classState, subclassState, in)
	if err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode subclass feature payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartSubclassFeatureApply,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "subclass feature did not emit an event",
		ApplyErrMessage: "apply subclass feature event",
	}); err != nil {
		return nil, err
	}

	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartApplySubclassFeatureResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}
