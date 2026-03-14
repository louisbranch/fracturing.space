package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runSessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session damage roll request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
	); err != nil {
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

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return nil, err
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	diceSpecs, err := damageDiceFromProto(in.GetDice())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcerror.Internal("failed to resolve seed", err)
	}

	result, err := daggerheart.RollDamage(daggerheart.DamageRollRequest{
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
	payload := action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":          result.Rolls,
			"base_total":     result.BaseTotal,
			sdKeyModifier:    result.Modifier,
			"critical":       in.GetCritical(),
			"critical_bonus": result.CriticalBonus,
			sdKeyTotal:       result.Total,
		},
		SystemData: rollSystemMetadata{
			CharacterID:   characterID,
			RollKind:      "damage_roll",
			Roll:          intPtrValue(result.Total),
			BaseTotal:     intPtrValue(result.BaseTotal),
			Modifier:      intPtrValue(result.Modifier),
			Critical:      boolPtr(in.GetCritical()),
			CriticalBonus: intPtrValue(result.CriticalBonus),
			Total:         intPtrValue(result.Total),
		}.mapValue(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}

	var rollSeqValue uint64
	domainResult, err := s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   ids.CampaignID(campaignID),
		Type:         commandTypeActionRollResolve,
		ActorType:    command.ActorTypeSystem,
		SessionID:    ids.SessionID(sessionID),
		SceneID:      ids.SceneID(sceneID),
		RequestID:    requestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "roll",
		EntityID:     requestID,
		PayloadJSON:  payloadJSON,
	}, s.stores.Applier(), domainwrite.Options{
		RequireEvents:     true,
		MissingEventMsg:   "damage roll did not emit an event",
		ExecuteErrMessage: "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	rollSeqValue = domainResult.Decision.Events[0].Seq

	response := &pb.SessionDamageRollResponse{
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
	}

	return response, nil
}
