package conditiontransport

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyConditions applies character condition and life-state mutations while
// preserving the package's existing transport contract.
func (h *Handler) ApplyConditions(ctx context.Context, in *pb.DaggerheartApplyConditionsRequest) (CharacterConditionsResult, error) {
	if in == nil {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, "apply conditions request is required")
	}
	if err := h.requireCharacterDependencies(); err != nil {
		return CharacterConditionsResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterConditionsResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterConditionsResult{}, err
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterConditionsResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterConditionsResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart conditions"); err != nil {
		return CharacterConditionsResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterConditionsResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterConditionsResult{}, err
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterConditionsResult{}, grpcerror.HandleDomainError(err)
	}

	lifeStateProvided := in.GetLifeState() != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	var lifeStateAfter string
	if lifeStateProvided {
		lifeStateAfter, err = lifeStateFromProto(in.GetLifeState())
		if err != nil {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	lifeStateChanged := false
	if lifeStateProvided {
		beforeValue, err := daggerheart.NormalizeLifeState(lifeStateBefore)
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("invalid stored life_state", err)
		}
		afterValue, err := daggerheart.NormalizeLifeState(lifeStateAfter)
		if err != nil {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
		lifeStateChanged = beforeValue != afterValue
	}

	addConditions, err := ConditionsFromProto(in.GetAdd())
	if err != nil {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := ConditionsFromProto(in.GetRemove())
	if err != nil {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 && !lifeStateProvided {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, "conditions or life_state are required")
	}

	normalizedAdd := []string{}
	if len(addConditions) > 0 {
		normalizedAdd, err = daggerheart.NormalizeConditions(addConditions)
		if err != nil {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalizedRemove := []string{}
	if len(removeConditions) > 0 {
		normalizedRemove, err = daggerheart.NormalizeConditions(removeConditions)
		if err != nil {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	removeSet := make(map[string]struct{}, len(normalizedRemove))
	for _, value := range normalizedRemove {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value]; ok {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	var before []string
	var after []string
	var added []string
	var removed []string
	conditionChanged := false
	if len(normalizedAdd) > 0 || len(normalizedRemove) > 0 {
		before, err = daggerheart.NormalizeConditions(state.Conditions)
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("invalid stored conditions", err)
		}

		afterSet := make(map[string]struct{}, len(before)+len(normalizedAdd))
		for _, value := range before {
			afterSet[value] = struct{}{}
		}
		for _, value := range normalizedRemove {
			delete(afterSet, value)
		}
		for _, value := range normalizedAdd {
			afterSet[value] = struct{}{}
		}

		afterList := make([]string, 0, len(afterSet))
		for value := range afterSet {
			afterList = append(afterList, value)
		}
		after, err = daggerheart.NormalizeConditions(afterList)
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("invalid condition set", err)
		}

		added, removed = daggerheart.DiffConditions(before, after)
		conditionChanged = len(added) > 0 || len(removed) > 0
		if !conditionChanged && !lifeStateChanged {
			return CharacterConditionsResult{}, status.Error(codes.FailedPrecondition, "no condition or life_state changes to apply")
		}
	} else if !lifeStateChanged {
		return CharacterConditionsResult{}, status.Error(codes.FailedPrecondition, "no condition or life_state changes to apply")
	}

	if err := h.validateRollSeq(ctx, campaignID, sessionID, in.RollSeq); err != nil {
		return CharacterConditionsResult{}, err
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	source := strings.TrimSpace(in.GetSource())

	if conditionChanged {
		payloadJSON, err := json.Marshal(daggerheart.ConditionChangePayload{
			CharacterID:      ids.CharacterID(characterID),
			ConditionsBefore: before,
			ConditionsAfter:  after,
			Added:            added,
			Removed:          removed,
			Source:           source,
			RollSeq:          in.RollSeq,
		})
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("encode condition payload", err)
		}
		if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartConditionChange,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       requestID,
			InvocationID:    invocationID,
			EntityType:      "character",
			EntityID:        characterID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "condition change did not emit an event",
			ApplyErrMessage: "apply condition event",
		}); err != nil {
			return CharacterConditionsResult{}, err
		}
	}

	if lifeStateChanged {
		payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchPayload{
			CharacterID:     ids.CharacterID(characterID),
			LifeStateBefore: &lifeStateBefore,
			LifeStateAfter:  &lifeStateAfter,
		})
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("encode character state payload", err)
		}
		if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartCharacterStatePatch,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       requestID,
			InvocationID:    invocationID,
			EntityType:      "character",
			EntityID:        characterID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "character state patch did not emit an event",
			ApplyErrMessage: "apply character state event",
		}); err != nil {
			return CharacterConditionsResult{}, err
		}
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterConditionsResult{}, grpcerror.Internal("load daggerheart state", err)
	}

	return CharacterConditionsResult{
		CharacterID: characterID,
		State:       updated,
		Added:       added,
		Removed:     removed,
	}, nil
}
