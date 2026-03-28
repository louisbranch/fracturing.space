package sessionrolltransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session damage roll request is required")
	}
	if err := h.requireDamageRollDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	if len(in.GetDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "dice are required")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}

	diceSpecs, err := damageDiceFromProto(in.GetDice())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		h.deps.SeedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to resolve seed", err)
	}

	result, err := rules.RollDamage(rules.DamageRollRequest{
		Dice:     diceSpecs,
		Modifier: int(in.GetModifier()),
		Seed:     seed,
		Critical: in.GetCritical(),
	})
	if err != nil {
		if errors.Is(err, dice.ErrMissingDice) || errors.Is(err, dice.ErrInvalidDiceSpec) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to roll damage", err)
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":                       result.Rolls,
			"base_total":                  result.BaseTotal,
			workflowtransport.KeyModifier: result.Modifier,
			"critical":                    in.GetCritical(),
			"critical_bonus":              result.CriticalBonus,
			workflowtransport.KeyTotal:    result.Total,
		},
		SystemData: workflowtransport.RollSystemMetadata{
			CharacterID:   characterID,
			RollKind:      "damage_roll",
			Roll:          workflowtransport.IntPtr(result.Total),
			BaseTotal:     workflowtransport.IntPtr(result.BaseTotal),
			Modifier:      workflowtransport.IntPtr(result.Modifier),
			Critical:      workflowtransport.BoolPtr(in.GetCritical()),
			CriticalBonus: workflowtransport.IntPtr(result.CriticalBonus),
			Total:         workflowtransport.IntPtr(result.Total),
		}.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteDamageRollResolve(ctx, RollResolveInput{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       requestID,
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "roll",
		EntityID:        requestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "damage roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionDamageRollResponse{
		RollSeq:       rollSeqValue,
		Rolls:         diceRollsToProto(result.Rolls),
		BaseTotal:     int32(result.BaseTotal),
		Modifier:      int32(result.Modifier),
		CriticalBonus: int32(result.CriticalBonus),
		Total:         int32(result.Total),
		Critical:      in.GetCritical(),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}
