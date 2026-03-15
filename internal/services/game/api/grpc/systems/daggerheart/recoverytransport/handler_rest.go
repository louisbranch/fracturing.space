package recoverytransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplyRest(ctx context.Context, in *pb.DaggerheartApplyRestRequest) (RestResult, error) {
	if in == nil {
		return RestResult{}, status.Error(codes.InvalidArgument, "apply rest request is required")
	}
	if err := h.requireDependencies(true); err != nil {
		return RestResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return RestResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart rest"); err != nil {
		return RestResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return RestResult{}, err
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return RestResult{}, err
	}
	if in.Rest == nil {
		return RestResult{}, status.Error(codes.InvalidArgument, "rest is required")
	}
	if in.Rest.RestType == pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED {
		return RestResult{}, status.Error(codes.InvalidArgument, "rest_type is required")
	}

	seed, err := resolveSeed(in.Rest.GetRng(), h.deps.SeedGenerator, h.deps.ResolveSeed)
	if err != nil {
		return RestResult{}, grpcerror.Internal("failed to resolve rest seed", err)
	}
	currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return RestResult{}, grpcerror.Internal("get daggerheart snapshot", err)
	}
	state := daggerheart.RestState{ConsecutiveShortRests: currentSnap.ConsecutiveShortRests}
	restType, err := restTypeFromProto(in.Rest.RestType)
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	outcome, err := daggerheart.ResolveRestOutcome(state, restType, in.Rest.Interrupted, seed, int(in.Rest.PartySize))
	if err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}

	longTermCountdownID := strings.TrimSpace(in.Rest.GetLongTermCountdownId())
	var longTermCountdown *daggerheart.Countdown
	if outcome.AdvanceCountdown && longTermCountdownID != "" {
		storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, longTermCountdownID)
		if err != nil {
			return RestResult{}, grpcerror.HandleDomainError(err)
		}
		countdown := countdownFromStorage(storedCountdown)
		longTermCountdown = &countdown
	}

	characterIDs := append([]string(nil), in.GetCharacterIds()...)
	payload, err := daggerheart.ResolveRestApplication(daggerheart.RestApplicationInput{
		RestType:               restType,
		Interrupted:            in.Rest.Interrupted,
		Outcome:                outcome,
		CurrentGMFear:          currentSnap.GMFear,
		ConsecutiveShortRests:  currentSnap.ConsecutiveShortRests,
		CharacterIDs:           characterIDs,
		LongTermCountdownState: longTermCountdown,
	})
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return RestResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartRestTake,
		SessionID:       sessionID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "session",
		EntityID:        campaignID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "rest did not emit an event",
		ApplyErrMessage: "apply rest event",
	}); err != nil {
		return RestResult{}, err
	}

	updatedSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return RestResult{}, grpcerror.Internal("load daggerheart snapshot", err)
	}
	entries := make([]CharacterStateEntry, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		if strings.TrimSpace(characterID) == "" {
			continue
		}
		currentState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return RestResult{}, grpcerror.Internal("get daggerheart character state", err)
		}
		entries = append(entries, CharacterStateEntry{CharacterID: characterID, State: currentState})
	}
	return RestResult{Snapshot: updatedSnap, CharacterStates: entries}, nil
}
