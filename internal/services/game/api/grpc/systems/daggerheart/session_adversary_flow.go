package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runSessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary rolls")
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

	if _, err := s.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
	}

	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	if advantage > 0 && disadvantage > 0 {
		advantage = 0
		disadvantage = 0
	}
	rollCount := 1
	if advantage > 0 || disadvantage > 0 {
		rollCount = 2
	}

	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 20, Count: rollCount}},
		Seed: seed,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to roll adversary die: %v", err)
	}
	rolls := result.Rolls[0].Results
	selected := rolls[0]
	if rollCount == 2 {
		if advantage > 0 {
			if rolls[1] > selected {
				selected = rolls[1]
			}
		} else if disadvantage > 0 {
			if rolls[1] < selected {
				selected = rolls[1]
			}
		}
	}
	modifier := int(in.GetAttackModifier())
	total := selected + modifier

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	rollSeq := latestSeq + 1

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payload := action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":        rolls,
			sdKeyRoll:      selected,
			sdKeyModifier:  modifier,
			sdKeyTotal:     total,
			"advantage":    advantage,
			"disadvantage": disadvantage,
		},
		SystemData: map[string]any{
			sdKeyCharacterID: adversaryID,
			sdKeyAdversaryID: adversaryID,
			sdKeyRollKind:    "adversary_roll",
			sdKeyRoll:        selected,
			sdKeyModifier:    modifier,
			sdKeyTotal:       total,
			"advantage":      advantage,
			"disadvantage":   disadvantage,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode adversary roll payload: %v", err)
	}

	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	domainResult, err := s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeActionRollResolve,
		ActorType:    command.ActorTypeSystem,
		SessionID:    sessionID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "adversary",
		EntityID:     adversaryID,
		PayloadJSON:  payloadJSON,
	}, s.stores.Applier(), domainwrite.Options{
		RequireEvents:     true,
		MissingEventMsg:   "adversary roll did not emit an event",
		ExecuteErrMessage: "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	rollSeqValue := domainResult.Decision.Events[0].Seq

	rollValues := make([]int32, 0, len(rolls))
	for _, roll := range rolls {
		rollValues = append(rollValues, int32(roll))
	}

	return &pb.SessionAdversaryAttackRollResponse{
		RollSeq: rollSeqValue,
		Roll:    int32(selected),
		Total:   int32(total),
		Rolls:   rollValues,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func (s *DaggerheartService) runSessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary action check request is required")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary checks")
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

	if _, err := s.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	rollSeq := latestSeq + 1

	difficulty := int(in.GetDifficulty())
	modifier := int(in.GetModifier())
	autoSuccess := !in.GetDramatic()
	success := true
	roll := 0
	total := 0
	var rngResp *commonv1.RngResponse

	if !autoSuccess {
		seed, seedSource, rollMode, err := random.ResolveSeed(
			in.GetRng(),
			s.seedFunc,
			func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
		)
		if err != nil {
			if errors.Is(err, random.ErrSeedOutOfRange()) {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
		}
		result, err := dice.RollDice(dice.Request{
			Dice: []dice.Spec{{Sides: 20, Count: 1}},
			Seed: seed,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to roll adversary action die: %v", err)
		}
		roll = result.Rolls[0].Results[0]
		total = roll + modifier
		success = total >= difficulty
		rngResp = &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		}
	} else {
		total = modifier
	}

	return &pb.SessionAdversaryActionCheckResponse{
		RollSeq:     rollSeq,
		AutoSuccess: autoSuccess,
		Success:     success,
		Roll:        int32(roll),
		Modifier:    int32(modifier),
		Total:       int32(total),
		Rng:         rngResp,
	}, nil
}

func (s *DaggerheartService) runSessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}
	targetID := strings.TrimSpace(in.GetTargetId())
	if targetID == "" {
		return nil, status.Error(codes.InvalidArgument, "target id is required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := s.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:     campaignID,
		SessionId:      sessionID,
		AdversaryId:    adversaryID,
		AttackModifier: in.GetAttackModifier(),
		Advantage:      in.GetAdvantage(),
		Disadvantage:   in.GetDisadvantage(),
		Rng:            in.GetAttackRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	attackOutcome, err := s.ApplyAdversaryAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  sessionID,
		RollSeq:    rollResp.GetRollSeq(),
		Targets:    []string{targetID},
		Difficulty: in.GetDifficulty(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAdversaryAttackFlowResponse{
		AttackRoll:    rollResp,
		AttackOutcome: attackOutcome,
	}

	if attackOutcome.GetResult() == nil || !attackOutcome.GetResult().GetSuccess() {
		return response, nil
	}

	if len(in.GetDamageDice()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "damage_dice are required")
	}

	critical := attackOutcome.GetResult().GetCrit() || in.GetDamageCritical()
	damageRoll, err := s.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: adversaryID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

	sourceCharacterIDs := normalizeTargets(in.GetDamage().GetSourceCharacterIds())
	sourceCharacterIDs = append(sourceCharacterIDs, adversaryID)
	sourceCharacterIDs = normalizeTargets(sourceCharacterIDs)

	damageReq := &pb.DaggerheartDamageRequest{
		Amount:             damageRoll.GetTotal(),
		DamageType:         in.GetDamage().GetDamageType(),
		ResistPhysical:     in.GetDamage().GetResistPhysical(),
		ResistMagic:        in.GetDamage().GetResistMagic(),
		ImmunePhysical:     in.GetDamage().GetImmunePhysical(),
		ImmuneMagic:        in.GetDamage().GetImmuneMagic(),
		Direct:             in.GetDamage().GetDirect(),
		MassiveDamage:      in.GetDamage().GetMassiveDamage(),
		Source:             in.GetDamage().GetSource(),
		SourceCharacterIds: sourceCharacterIDs,
	}

	applyDamage, err := s.ApplyDamage(ctxWithMeta, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        campaignID,
		CharacterId:       targetID,
		Damage:            damageReq,
		RollSeq:           &damageRoll.RollSeq,
		RequireDamageRoll: in.GetRequireDamageRoll(),
	})
	if err != nil {
		return nil, err
	}

	response.DamageRoll = damageRoll
	response.DamageApplied = applyDamage
	return response, nil
}
