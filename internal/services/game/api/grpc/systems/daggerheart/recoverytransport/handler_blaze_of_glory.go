package recoverytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (BlazeResult, error) {
	if in == nil {
		return BlazeResult{}, status.Error(codes.InvalidArgument, "resolve blaze of glory request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return BlazeResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return BlazeResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return BlazeResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return BlazeResult{}, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return BlazeResult{}, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart blaze of glory"); err != nil {
		return BlazeResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return BlazeResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return BlazeResult{}, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return BlazeResult{}, handleDomainError(ctx, err)
	}
	if state.LifeState == "" {
		state.LifeState = daggerheartstate.LifeStateAlive
	}
	if state.LifeState == mechanics.LifeStateDead {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is already dead")
	}
	if state.LifeState != mechanics.LifeStateBlazeOfGlory {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is not in blaze of glory")
	}
	lifeStateBefore := state.LifeState
	lifeStateAfter := mechanics.LifeStateDead
	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:     ids.CharacterID(characterID),
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	})
	if err != nil {
		return BlazeResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "blaze of glory did not emit an event",
		ApplyErrMessage: "apply character state event",
	}); err != nil {
		return BlazeResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return BlazeResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Reason:      mechanics.LifeStateBlazeOfGlory,
	}); err != nil {
		return BlazeResult{}, err
	}
	return BlazeResult{CharacterID: characterID, State: updated, LifeState: lifeStateAfter}, nil
}

func applyDeathOutcomeState(current projectionstore.DaggerheartCharacterState, outcome daggerheart.DeathMoveOutcome) projectionstore.DaggerheartCharacterState {
	current.Hp = outcome.HPAfter
	current.Hope = outcome.HopeAfter
	current.HopeMax = outcome.HopeMaxAfter
	current.Stress = outcome.StressAfter
	current.LifeState = outcome.LifeState
	return current
}
