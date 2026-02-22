package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sessionFlowApplication struct {
	service *DaggerheartService
}

func newSessionFlowApplication(service *DaggerheartService) sessionFlowApplication {
	return sessionFlowApplication{service: service}
}

func (a sessionFlowApplication) runSessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if err := a.service.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
		dependencyDomainEngine,
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
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart rolls")
	}

	sess, err := a.service.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	state, err := a.service.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
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

	latestSeq, err := a.service.stores.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load latest event seq: %v", err)
	}
	preEvents := spendEventCount
	rollSeq := latestSeq + uint64(preEvents) + 1

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		a.service.seedFunc,
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
			adapter := daggerheart.NewAdapter(a.service.stores.Daggerheart)
			cmd := commandbuild.DaggerheartSystemCommand(commandbuild.DaggerheartSystemCommandInput{
				CampaignID:   campaignID,
				Type:         commandTypeDaggerheartHopeSpend,
				SessionID:    sessionID,
				RequestID:    requestID,
				InvocationID: invocationID,
				EntityType:   "character",
				EntityID:     characterID,
				PayloadJSON:  payloadJSON,
			})
			_, err = a.service.executeAndApplyDomainCommand(ctx, cmd, adapter, domainCommandApplyOptions{
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

	var rollSeqValue uint64
	cmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
		CampaignID:   campaignID,
		Type:         commandTypeActionRollResolve,
		SessionID:    sessionID,
		RequestID:    requestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "roll",
		EntityID:     requestID,
		PayloadJSON:  payloadJSON,
	})
	domainResult, err := a.service.executeAndApplyDomainCommand(ctx, cmd, a.service.stores.Applier(), domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "action roll did not emit an event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}
	rollSeqValue = domainResult.Decision.Events[0].Seq

	failed := result.Difficulty != nil && !result.MeetsDifficulty
	if err := a.service.advanceBreathCountdown(ctx, campaignID, sessionID, strings.TrimSpace(in.GetBreathCountdownId()), failed); err != nil {
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
