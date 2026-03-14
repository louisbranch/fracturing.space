package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runSessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if err := s.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencySeedGenerator,
		dependencyDomainEngine,
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
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
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
		return nil, grpcerror.Internal("load latest event seq", err)
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
		return nil, grpcerror.Internal("failed to resolve seed", err)
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
				CharacterID: ids.CharacterID(characterID),
				Amount:      spend.Amount,
				Before:      before,
				After:       after,
				RollSeq:     &rollSeq,
				Source:      spend.Source,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, grpcerror.Internal("encode hope spend payload", err)
			}
			requestID := grpcmeta.RequestIDFromContext(ctx)
			invocationID := grpcmeta.InvocationIDFromContext(ctx)
			adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
			cmd := commandbuild.SystemCommand(commandbuild.SystemCommandInput{
				CampaignID:    campaignID,
				Type:          commandTypeDaggerheartHopeSpend,
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				SessionID:     sessionID,
				SceneID:       sceneID,
				RequestID:     requestID,
				InvocationID:  invocationID,
				EntityType:    "character",
				EntityID:      characterID,
				PayloadJSON:   payloadJSON,
			})
			_, err = s.executeAndApplyDomainCommand(ctx, cmd, adapter, domainwrite.Options{
				RequireEvents:     true,
				MissingEventMsg:   "hope spend did not emit an event",
				ExecuteErrMessage: "execute domain command",
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
		return nil, grpcerror.Internal("failed to roll action", err)
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
		sdKeyModifier:        result.Modifier,
		"advantage_modifier": result.AdvantageModifier,
		sdKeyTotal:           result.Total,
		"difficulty":         difficulty,
		"success":            result.MeetsDifficulty,
		sdKeyCrit:            result.IsCrit,
	}
	if len(modifierList) > 0 {
		results["modifiers"] = modifierList
	}

	systemMetadata := rollSystemMetadata{
		CharacterID:  characterID,
		Trait:        trait,
		RollKind:     rollKind.String(),
		Outcome:      outcomeCode,
		Flavor:       flavor,
		HopeFear:     boolPtr(generateHopeFear),
		Crit:         boolPtr(result.IsCrit),
		CritNegates:  boolPtr(critNegatesEffects),
		GMMove:       boolPtr(triggerGMMove),
		Advantage:    intPtrValue(advantage),
		Disadvantage: intPtrValue(disadvantage),
		Underwater:   boolPtr(in.GetUnderwater()),
		Modifiers:    modifierList,
	}
	if countdownID := strings.TrimSpace(in.GetBreathCountdownId()); countdownID != "" {
		systemMetadata.BreathCountdownID = countdownID
	}
	systemData := systemMetadata.mapValue()

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
		return nil, grpcerror.Internal("encode payload", err)
	}

	var rollSeqValue uint64
	cmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
		CampaignID:   campaignID,
		Type:         commandTypeActionRollResolve,
		SessionID:    sessionID,
		SceneID:      sceneID,
		RequestID:    requestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "roll",
		EntityID:     requestID,
		PayloadJSON:  payloadJSON,
	})
	domainResult, err := s.executeAndApplyDomainCommand(ctx, cmd, s.stores.Applier(), domainwrite.Options{
		RequireEvents:     true,
		MissingEventMsg:   "action roll did not emit an event",
		ExecuteErrMessage: "execute domain command",
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
