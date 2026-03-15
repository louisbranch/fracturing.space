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

// ApplyAdversaryConditions applies adversary condition mutations while keeping
// the existing read/load and command-emission contract intact.
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
		return AdversaryConditionsResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return AdversaryConditionsResult{}, grpcerror.HandleDomainError(err)
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

	addConditions, err := ConditionsFromProto(in.GetAdd())
	if err != nil {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := ConditionsFromProto(in.GetRemove())
	if err != nil {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 {
		return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, "conditions to add or remove are required")
	}

	normalizedAdd := []string{}
	if len(addConditions) > 0 {
		normalizedAdd, err = daggerheart.NormalizeConditions(addConditions)
		if err != nil {
			return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalizedRemove := []string{}
	if len(removeConditions) > 0 {
		normalizedRemove, err = daggerheart.NormalizeConditions(removeConditions)
		if err != nil {
			return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	removeSet := make(map[string]struct{}, len(normalizedRemove))
	for _, value := range normalizedRemove {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value]; ok {
			return AdversaryConditionsResult{}, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	adversary, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return AdversaryConditionsResult{}, err
	}
	before, err := daggerheart.NormalizeConditions(adversary.Conditions)
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("invalid stored conditions", err)
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
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return AdversaryConditionsResult{}, grpcerror.Internal("invalid condition set", err)
	}

	added, removed := daggerheart.DiffConditions(before, after)
	if len(added) == 0 && len(removed) == 0 {
		return AdversaryConditionsResult{}, status.Error(codes.FailedPrecondition, "no condition changes to apply")
	}

	if err := h.validateRollSeq(ctx, campaignID, sessionID, in.RollSeq); err != nil {
		return AdversaryConditionsResult{}, err
	}

	payloadJSON, err := json.Marshal(daggerheart.AdversaryConditionChangePayload{
		AdversaryID:      ids.AdversaryID(adversaryID),
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
		Added:       added,
		Removed:     removed,
	}, nil
}
