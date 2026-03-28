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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart condition and life-state mutation transport
// endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart condition transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

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
		return CharacterConditionsResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterConditionsResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
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
		return CharacterConditionsResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
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
		lifeStateBefore = daggerheartstate.LifeStateAlive
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

	addConditions, err := ConditionStatesFromProto(in.GetAddConditions())
	if err != nil {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := normalizeConditionRemovalIDs(in.GetRemoveConditionIds())
	if err != nil {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 && !lifeStateProvided {
		return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, "conditions or life_state are required")
	}

	normalizedAdd := []rules.ConditionState{}
	if len(addConditions) > 0 {
		normalizedAdd, err = ConditionStateViewsToDomain(addConditions)
		if err != nil {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	removeSet := make(map[string]struct{}, len(removeConditions))
	for _, value := range removeConditions {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value.ID]; ok {
			return CharacterConditionsResult{}, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	var before []rules.ConditionState
	var after []rules.ConditionState
	var added []rules.ConditionState
	var removed []rules.ConditionState
	conditionChanged := false
	if len(normalizedAdd) > 0 || len(removeConditions) > 0 {
		before, err = rules.NormalizeConditionStates(ProjectionConditionStatesToDomain(state.Conditions))
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("invalid stored conditions", err)
		}

		afterSet := make(map[string]rules.ConditionState, len(before)+len(normalizedAdd))
		for _, value := range before {
			afterSet[value.ID] = value
		}
		for _, value := range removeConditions {
			delete(afterSet, value)
		}
		for _, value := range normalizedAdd {
			afterSet[value.ID] = value
		}

		afterList := make([]rules.ConditionState, 0, len(afterSet))
		for _, value := range afterSet {
			afterList = append(afterList, value)
		}
		after, err = rules.NormalizeConditionStates(afterList)
		if err != nil {
			return CharacterConditionsResult{}, grpcerror.Internal("invalid condition set", err)
		}

		added, removed = rules.DiffConditionStates(before, after)
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
		payloadJSON, err := json.Marshal(daggerheartpayload.ConditionChangePayload{
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
		payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchPayload{
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
		Added:       DomainConditionStatesToViews(added),
		Removed:     DomainConditionStatesToViews(removed),
	}, nil
}

func (h *Handler) ApplyAdversaryConditions(ctx context.Context, in *pb.DaggerheartApplyAdversaryConditionsRequest) (AdversaryConditionsResult, error) {
	if in == nil {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, "apply adversary conditions request is required")
	}
	if err := h.requireAdversaryDependencies(); err != nil {
		return AdversaryConditionsResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return AdversaryConditionsResult{}, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return AdversaryConditionsResult{}, err
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return AdversaryConditionsResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart conditions"); err != nil {
		return AdversaryConditionsResult{}, err
	}

	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return AdversaryConditionsResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return AdversaryConditionsResult{}, err
	}

	addConditions, err := ConditionStatesFromProto(in.GetAddConditions())
	if err != nil {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := normalizeConditionRemovalIDs(in.GetRemoveConditionIds())
	if err != nil {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, "conditions to add or remove are required")
	}

	normalizedAdd := []rules.ConditionState{}
	if len(addConditions) > 0 {
		normalizedAdd, err = ConditionStateViewsToDomain(addConditions)
		if err != nil {
			return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	removeSet := make(map[string]struct{}, len(removeConditions))
	for _, value := range removeConditions {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value.ID]; ok {
			return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	adversary, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return AdversaryConditionsResult{}, err
	}
	before, err := rules.NormalizeConditionStates(ProjectionConditionStatesToDomain(adversary.Conditions))
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("invalid stored conditions", err)
	}

	afterSet := make(map[string]rules.ConditionState, len(before)+len(normalizedAdd))
	for _, value := range before {
		afterSet[value.ID] = value
	}
	for _, value := range removeConditions {
		delete(afterSet, value)
	}
	for _, value := range normalizedAdd {
		afterSet[value.ID] = value
	}

	afterList := make([]rules.ConditionState, 0, len(afterSet))
	for _, value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := rules.NormalizeConditionStates(afterList)
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("invalid condition set", err)
	}

	added, removed := rules.DiffConditionStates(before, after)
	if len(added) == 0 && len(removed) == 0 {
		return AdversaryConditionsResult{}, status.Error(codes.FailedPrecondition, "no condition changes to apply")
	}

	if err := h.validateRollSeq(ctx, campaignID, sessionID, in.RollSeq); err != nil {
		return AdversaryConditionsResult{}, err
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.AdversaryConditionChangePayload{
		AdversaryID:      dhids.AdversaryID(adversaryID),
		ConditionsBefore: before,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
		Source:           strings.TrimSpace(in.GetSource()),
		RollSeq:          in.RollSeq,
	})
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("encode condition payload", err)
	}

	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartAdversaryConditionChange,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary condition change did not emit an event",
		ApplyErrMessage: "apply adversary condition event",
	}); err != nil {
		return AdversaryConditionsResult{}, err
	}

	updated, err := h.deps.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("load daggerheart adversary", err)
	}

	return AdversaryConditionsResult{
		AdversaryID: adversaryID,
		Adversary:   updated,
		Added:       DomainConditionStatesToViews(added),
		Removed:     DomainConditionStatesToViews(removed),
	}, nil
}

func normalizeConditionRemovalIDs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, status.Error(codes.InvalidArgument, "remove_condition_ids cannot include empty values")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func (h *Handler) validateRollSeq(ctx context.Context, campaignID, sessionID string, rollSeq *uint64) error {
	if rollSeq == nil {
		return nil
	}
	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, *rollSeq)
	if err != nil {
		return grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if sessionID != "" && rollEvent.SessionID.String() != sessionID {
		return status.Error(codes.InvalidArgument, "roll seq does not match session")
	}
	return nil
}

func (h *Handler) requireCharacterDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}
