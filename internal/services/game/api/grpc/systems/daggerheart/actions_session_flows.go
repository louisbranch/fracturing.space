package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rolls")
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

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	modifierTotal, modifierList := normalizeActionModifiers(in.GetModifiers())
	rollKind := normalizeRollKind(in.GetRollKind())
	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	if advantage < 0 {
		advantage = 0
	}
	if disadvantage < 0 {
		disadvantage = 0
	}
	if in.GetUnderwater() && rollKind == pb.RollKind_ROLL_KIND_ACTION {
		disadvantage++
	}
	hopeSpends := hopeSpendsFromModifiers(in.GetModifiers())
	spendEventCount := 0
	totalSpend := 0
	for _, spend := range hopeSpends {
		if spend.Amount > 0 {
			spendEventCount++
			totalSpend += spend.Amount
		}
	}
	if rollKind == pb.RollKind_ROLL_KIND_REACTION && spendEventCount > 0 {
		return nil, status.Error(codes.InvalidArgument, "reaction rolls cannot spend hope")
	}

	latestSeq, err := s.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	preEvents := spendEventCount
	rollSeq := latestSeq + uint64(preEvents) + 1

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

	if rollKind == pb.RollKind_ROLL_KIND_ACTION && spendEventCount > 0 {
		hopeBefore := state.Hope
		hopeAfter := hopeBefore
		if hopeBefore < totalSpend {
			return nil, status.Error(codes.FailedPrecondition, "insufficient hope")
		}

		for _, spend := range hopeSpends {
			if spend.Amount <= 0 {
				continue
			}
			before := hopeAfter
			after := before - spend.Amount
			payload := daggerheart.HopeSpendPayload{
				CharacterID: characterID,
				Amount:      spend.Amount,
				Before:      before,
				After:       after,
				RollSeq:     &rollSeq,
				Source:      spend.Source,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "encode hope spend payload: %v", err)
			}
			requestID := grpcmeta.RequestIDFromContext(ctx)
			invocationID := grpcmeta.InvocationIDFromContext(ctx)
			if s.stores.Domain == nil {
				return nil, status.Error(codes.Internal, "domain engine is not configured")
			}
			adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
			_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
				CampaignID:    campaignID,
				Type:          commandTypeDaggerheartHopeSpend,
				ActorType:     command.ActorTypeSystem,
				SessionID:     sessionID,
				RequestID:     requestID,
				InvocationID:  invocationID,
				EntityType:    "character",
				EntityID:      characterID,
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}, adapter, domainCommandApplyOptions{
				requireEvents:   true,
				missingEventMsg: "hope spend did not emit an event",
				executeErrMsg:   "execute domain command",
			})
			if err != nil {
				return nil, err
			}
			hopeAfter = after
		}
	}

	difficulty := int(in.GetDifficulty())
	result, generateHopeFear, triggerGMMove, critNegatesEffects, err := resolveRoll(
		rollKind,
		daggerheartdomain.ActionRequest{
			Modifier:     modifierTotal,
			Difficulty:   &difficulty,
			Seed:         seed,
			Advantage:    advantage,
			Disadvantage: disadvantage,
		},
	)
	if err != nil {
		if errors.Is(err, daggerheartdomain.ErrInvalidDifficulty) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to roll action: %v", err)
	}

	rollModeLabel := rollMode.String()
	outcomeCode := outcomeToProto(result.Outcome).String()
	flavor := outcomeFlavorFromCode(outcomeCode)
	if !generateHopeFear {
		flavor = ""
	}

	results := map[string]any{
		"rng": map[string]any{
			"seed_used":   uint64(seed),
			"rng_algo":    random.RngAlgoMathRandV1,
			"seed_source": seedSource,
			"roll_mode":   rollModeLabel,
		},
		"dice": map[string]any{
			"hope_die":      result.Hope,
			"fear_die":      result.Fear,
			"advantage_die": result.AdvantageDie,
		},
		"modifier":           result.Modifier,
		"advantage_modifier": result.AdvantageModifier,
		"total":              result.Total,
		"difficulty":         difficulty,
		"success":            result.MeetsDifficulty,
		"crit":               result.IsCrit,
	}
	if len(modifierList) > 0 {
		results["modifiers"] = modifierList
	}

	systemData := map[string]any{
		"character_id": characterID,
		"trait":        trait,
		"roll_kind":    rollKind.String(),
		"outcome":      outcomeCode,
		"flavor":       flavor,
		"crit":         result.IsCrit,
		"hope_fear":    generateHopeFear,
		"gm_move":      triggerGMMove,
		"crit_negates": critNegatesEffects,
		"advantage":    advantage,
		"disadvantage": disadvantage,
		"underwater":   in.GetUnderwater(),
	}
	if len(modifierList) > 0 {
		systemData["modifiers"] = modifierList
	}
	if countdownID := strings.TrimSpace(in.GetBreathCountdownId()); countdownID != "" {
		systemData["breath_countdown_id"] = countdownID
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payload := action.RollResolvePayload{
		RequestID:  requestID,
		RollSeq:    rollSeq,
		Results:    results,
		Outcome:    outcomeCode,
		SystemData: systemData,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}
	var rollSeqValue uint64
	domainResult, err := s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeActionRollResolve,
		ActorType:    command.ActorTypeSystem,
		SessionID:    sessionID,
		RequestID:    requestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "roll",
		EntityID:     requestID,
		PayloadJSON:  payloadJSON,
	}, s.stores.Applier(), domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "action roll did not emit an event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	rollSeqValue = domainResult.Decision.Events[0].Seq

	failed := result.Difficulty != nil && !result.MeetsDifficulty
	if err := s.advanceBreathCountdown(ctx, campaignID, sessionID, strings.TrimSpace(in.GetBreathCountdownId()), failed); err != nil {
		return nil, err
	}

	return &pb.SessionActionRollResponse{
		RollSeq:    rollSeqValue,
		HopeDie:    int32(result.Hope),
		FearDie:    int32(result.Fear),
		Total:      int32(result.Total),
		Difficulty: int32(difficulty),
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Crit:       result.IsCrit,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func (s *DaggerheartService) SessionDamageRoll(ctx context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session damage roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
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
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rolls")
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
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to resolve seed: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to roll damage: %v", err)
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payload := action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":          result.Rolls,
			"base_total":     result.BaseTotal,
			"modifier":       result.Modifier,
			"critical":       in.GetCritical(),
			"critical_bonus": result.CriticalBonus,
			"total":          result.Total,
		},
		SystemData: map[string]any{
			"character_id":   characterID,
			"roll_kind":      "damage_roll",
			"roll":           result.Total,
			"base_total":     result.BaseTotal,
			"modifier":       result.Modifier,
			"critical":       in.GetCritical(),
			"critical_bonus": result.CriticalBonus,
			"total":          result.Total,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}
	var rollSeqValue uint64
	domainResult, err := s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeActionRollResolve,
		ActorType:    command.ActorTypeSystem,
		SessionID:    sessionID,
		RequestID:    requestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "roll",
		EntityID:     requestID,
		PayloadJSON:  payloadJSON,
	}, s.stores.Applier(), domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "damage roll did not emit an event",
		executeErrMsg:   "execute domain command",
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

func (s *DaggerheartService) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session attack flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	attackerID := strings.TrimSpace(in.GetCharacterId())
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}
	targetID := strings.TrimSpace(in.GetTargetId())
	if targetID == "" {
		return nil, status.Error(codes.InvalidArgument, "target id is required")
	}
	if in.GetDamage() == nil {
		return nil, status.Error(codes.InvalidArgument, "damage is required")
	}
	if in.GetDamage().GetDamageType() == pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "damage_type is required")
	}

	rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		CharacterId:       attackerID,
		Trait:             trait,
		RollKind:          pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:        in.GetDifficulty(),
		Modifiers:         in.GetModifiers(),
		Underwater:        in.GetUnderwater(),
		BreathCountdownId: in.GetBreathCountdownId(),
		Rng:               in.GetActionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	attackOutcome, err := s.ApplyAttackOutcome(ctxWithMeta, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionAttackFlowResponse{
		ActionRoll:    rollResp,
		RollOutcome:   rollOutcome,
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
		CharacterId: attackerID,
		Dice:        in.GetDamageDice(),
		Modifier:    in.GetDamageModifier(),
		Critical:    critical,
		Rng:         in.GetDamageRng(),
	})
	if err != nil {
		return nil, err
	}

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
		SourceCharacterIds: normalizeTargets(in.GetDamage().GetSourceCharacterIds()),
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

func (s *DaggerheartService) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	actorID := strings.TrimSpace(in.GetCharacterId())
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		CharacterId:  actorID,
		Trait:        trait,
		RollKind:     pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:   in.GetDifficulty(),
		Modifiers:    in.GetModifiers(),
		Advantage:    in.GetAdvantage(),
		Disadvantage: in.GetDisadvantage(),
		Rng:          in.GetReactionRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	reactionOutcome, err := s.ApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	response := &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}

	return response, nil
}

func (s *DaggerheartService) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
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
			"roll":         selected,
			"modifier":     modifier,
			"total":        total,
			"advantage":    advantage,
			"disadvantage": disadvantage,
		},
		SystemData: map[string]any{
			"character_id": adversaryID,
			"adversary_id": adversaryID,
			"roll_kind":    "adversary_roll",
			"roll":         selected,
			"modifier":     modifier,
			"total":        total,
			"advantage":    advantage,
			"disadvantage": disadvantage,
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
	}, s.stores.Applier(), domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "adversary roll did not emit an event",
		executeErrMsg:   "execute domain command",
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

func (s *DaggerheartService) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary action check request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
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

func (s *DaggerheartService) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest) (*pb.SessionAdversaryAttackFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
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

func (s *DaggerheartService) SessionGroupActionFlow(ctx context.Context, in *pb.SessionGroupActionFlowRequest) (*pb.SessionGroupActionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session group action flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	leaderID := strings.TrimSpace(in.GetLeaderCharacterId())
	if leaderID == "" {
		return nil, status.Error(codes.InvalidArgument, "leader character id is required")
	}
	leaderTrait := strings.TrimSpace(in.GetLeaderTrait())
	if leaderTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "leader trait is required")
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	supporters := in.GetSupporters()
	if len(supporters) == 0 {
		return nil, status.Error(codes.InvalidArgument, "supporters are required")
	}

	supportRolls := make([]*pb.GroupActionSupporterRoll, 0, len(supporters))
	supportSuccesses := 0
	supportFailures := 0
	for _, supporter := range supporters {
		if supporter == nil {
			return nil, status.Error(codes.InvalidArgument, "supporter is required")
		}
		supporterID := strings.TrimSpace(supporter.GetCharacterId())
		if supporterID == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter character id is required")
		}
		supporterTrait := strings.TrimSpace(supporter.GetTrait())
		if supporterTrait == "" {
			return nil, status.Error(codes.InvalidArgument, "supporter trait is required")
		}

		rollResp, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CharacterId: supporterID,
			Trait:       supporterTrait,
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			Difficulty:  in.GetDifficulty(),
			Modifiers:   supporter.GetModifiers(),
			Rng:         supporter.GetRng(),
		})
		if err != nil {
			return nil, err
		}
		if rollResp.GetSuccess() {
			supportSuccesses++
		} else {
			supportFailures++
		}

		supportRolls = append(supportRolls, &pb.GroupActionSupporterRoll{
			CharacterId: supporterID,
			ActionRoll:  rollResp,
			Success:     rollResp.GetSuccess(),
		})
	}

	supportModifier := supportSuccesses - supportFailures
	leaderModifiers := append([]*pb.ActionRollModifier{}, in.GetLeaderModifiers()...)
	if supportModifier != 0 {
		leaderModifiers = append(leaderModifiers, &pb.ActionRollModifier{
			Value:  int32(supportModifier),
			Source: "group_action_support",
		})
	}

	leaderRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: leaderID,
		Trait:       leaderTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   leaderModifiers,
		Rng:         in.GetLeaderRng(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	leaderOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   leaderRoll.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionGroupActionFlowResponse{
		LeaderRoll:       leaderRoll,
		LeaderOutcome:    leaderOutcome,
		SupporterRolls:   supportRolls,
		SupportModifier:  int32(supportModifier),
		SupportSuccesses: int32(supportSuccesses),
		SupportFailures:  int32(supportFailures),
	}, nil
}

func (s *DaggerheartService) SessionTagTeamFlow(ctx context.Context, in *pb.SessionTagTeamFlowRequest) (*pb.SessionTagTeamFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session tag team flow request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}
	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetDifficulty() == 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty is required")
	}
	first := in.GetFirst()
	if first == nil {
		return nil, status.Error(codes.InvalidArgument, "first participant is required")
	}
	second := in.GetSecond()
	if second == nil {
		return nil, status.Error(codes.InvalidArgument, "second participant is required")
	}
	firstID := strings.TrimSpace(first.GetCharacterId())
	if firstID == "" {
		return nil, status.Error(codes.InvalidArgument, "first character id is required")
	}
	secondID := strings.TrimSpace(second.GetCharacterId())
	if secondID == "" {
		return nil, status.Error(codes.InvalidArgument, "second character id is required")
	}
	if firstID == secondID {
		return nil, status.Error(codes.InvalidArgument, "tag team participants must be distinct")
	}
	firstTrait := strings.TrimSpace(first.GetTrait())
	if firstTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "first trait is required")
	}
	secondTrait := strings.TrimSpace(second.GetTrait())
	if secondTrait == "" {
		return nil, status.Error(codes.InvalidArgument, "second trait is required")
	}
	selectedID := strings.TrimSpace(in.GetSelectedCharacterId())
	if selectedID == "" {
		return nil, status.Error(codes.InvalidArgument, "selected character id is required")
	}
	if selectedID != firstID && selectedID != secondID {
		return nil, status.Error(codes.InvalidArgument, "selected character id must match a participant")
	}

	firstRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: firstID,
		Trait:       firstTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   first.GetModifiers(),
		Rng:         first.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	secondRoll, err := s.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: secondID,
		Trait:       secondTrait,
		RollKind:    pb.RollKind_ROLL_KIND_ACTION,
		Difficulty:  in.GetDifficulty(),
		Modifiers:   second.GetModifiers(),
		Rng:         second.GetRng(),
	})
	if err != nil {
		return nil, err
	}

	selectedRoll := firstRoll
	if selectedID == secondID {
		selectedRoll = secondRoll
	}

	ctxWithMeta := withCampaignSessionMetadata(ctx, campaignID, sessionID)
	applyTargets := []string{firstID, secondID}
	selectedOutcome, err := s.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   selectedRoll.GetRollSeq(),
		Targets:   applyTargets,
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionTagTeamFlowResponse{
		FirstRoll:           firstRoll,
		SecondRoll:          secondRoll,
		SelectedOutcome:     selectedOutcome,
		SelectedCharacterId: selectedID,
		SelectedRollSeq:     selectedRoll.GetRollSeq(),
	}, nil
}
