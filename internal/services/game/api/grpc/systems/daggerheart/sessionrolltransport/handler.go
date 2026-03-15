package sessionrolltransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the low-level Daggerheart session roll endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a low-level Daggerheart session roll handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) SessionActionRoll(ctx context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}
	if err := h.requireActionRollDependencies(); err != nil {
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

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}

	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
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

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + uint64(spendEventCount) + 1

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

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
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
			if err := h.deps.ExecuteHopeSpend(ctx, HopeSpendInput{
				CampaignID:   campaignID,
				SessionID:    sessionID,
				SceneID:      sceneID,
				RequestID:    requestID,
				InvocationID: invocationID,
				CharacterID:  characterID,
				Source:       spend.Source,
				Amount:       spend.Amount,
				HopeBefore:   before,
				HopeAfter:    after,
				RollSeq:      rollSeq,
			}); err != nil {
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

	outcomeCode := outcomeToProto(result.Outcome).String()
	flavor := workflowtransport.OutcomeFlavorFromCode(outcomeCode)
	if !generateHopeFear {
		flavor = ""
	}

	results := map[string]any{
		"rng": map[string]any{
			"seed_used":   uint64(seed),
			"rng_algo":    random.RngAlgoMathRandV1,
			"seed_source": seedSource,
			"roll_mode":   rollMode.String(),
		},
		"dice": map[string]any{
			"hope_die":      result.Hope,
			"fear_die":      result.Fear,
			"advantage_die": result.AdvantageDie,
		},
		workflowtransport.KeyModifier: result.Modifier,
		"advantage_modifier":          result.AdvantageModifier,
		workflowtransport.KeyTotal:    result.Total,
		"difficulty":                  difficulty,
		"success":                     result.MeetsDifficulty,
		workflowtransport.KeyCrit:     result.IsCrit,
	}
	if len(modifierList) > 0 {
		results["modifiers"] = modifierList
	}

	systemMetadata := workflowtransport.RollSystemMetadata{
		CharacterID:  characterID,
		Trait:        trait,
		RollKind:     rollKind.String(),
		Outcome:      outcomeCode,
		Flavor:       flavor,
		HopeFear:     workflowtransport.BoolPtr(generateHopeFear),
		Crit:         workflowtransport.BoolPtr(result.IsCrit),
		CritNegates:  workflowtransport.BoolPtr(critNegatesEffects),
		GMMove:       workflowtransport.BoolPtr(triggerGMMove),
		Advantage:    workflowtransport.IntPtr(advantage),
		Disadvantage: workflowtransport.IntPtr(disadvantage),
		Underwater:   workflowtransport.BoolPtr(in.GetUnderwater()),
		Modifiers:    modifierList,
	}
	if countdownID := strings.TrimSpace(in.GetBreathCountdownId()); countdownID != "" {
		systemMetadata.BreathCountdownID = countdownID
	}

	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID:  requestID,
		RollSeq:    rollSeq,
		Results:    results,
		Outcome:    outcomeCode,
		SystemData: systemMetadata.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteActionRollResolve(ctx, RollResolveInput{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       requestID,
		InvocationID:    invocationID,
		EntityType:      "roll",
		EntityID:        requestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "action roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	failed := result.Difficulty != nil && !result.MeetsDifficulty
	if err := h.deps.AdvanceBreathCountdown(ctx, campaignID, sessionID, strings.TrimSpace(in.GetBreathCountdownId()), failed); err != nil {
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
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart rolls"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
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

	result, err := bridge.RollDamage(bridge.DamageRollRequest{
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

func (h *Handler) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
	}
	if err := h.requireAdversaryRollDependencies(); err != nil {
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
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversary rolls"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}
	if _, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

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
		return nil, grpcerror.Internal("failed to roll adversary die", err)
	}
	rolls := result.Rolls[0].Results
	selected := rolls[0]
	if rollCount == 2 {
		if advantage > 0 && rolls[1] > selected {
			selected = rolls[1]
		}
		if disadvantage > 0 && rolls[1] < selected {
			selected = rolls[1]
		}
	}
	modifier := int(in.GetAttackModifier())
	total := selected + modifier

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + 1

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":                       rolls,
			workflowtransport.KeyRoll:     selected,
			workflowtransport.KeyModifier: modifier,
			workflowtransport.KeyTotal:    total,
			"advantage":                   advantage,
			"disadvantage":                disadvantage,
		},
		SystemData: workflowtransport.RollSystemMetadata{
			CharacterID:  adversaryID,
			AdversaryID:  adversaryID,
			RollKind:     "adversary_roll",
			Roll:         workflowtransport.IntPtr(selected),
			Modifier:     workflowtransport.IntPtr(modifier),
			Total:        workflowtransport.IntPtr(total),
			Advantage:    workflowtransport.IntPtr(advantage),
			Disadvantage: workflowtransport.IntPtr(disadvantage),
		}.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary roll payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteAdversaryRollResolve(ctx, RollResolveInput{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       requestID,
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

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

func (h *Handler) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary action check request is required")
	}
	if err := h.requireAdversaryActionCheckDependencies(); err != nil {
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
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart adversary checks"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}
	if _, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
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
			h.deps.SeedFunc,
			func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
		)
		if err != nil {
			if errors.Is(err, random.ErrSeedOutOfRange()) {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			return nil, grpcerror.Internal("failed to resolve seed", err)
		}
		result, err := dice.RollDice(dice.Request{
			Dice: []dice.Spec{{Sides: 20, Count: 1}},
			Seed: seed,
		})
		if err != nil {
			return nil, grpcerror.Internal("failed to roll adversary action die", err)
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

func (h *Handler) requireActionRollDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.ExecuteActionRollResolve == nil:
		return status.Error(codes.Internal, "action roll executor is not configured")
	case h.deps.ExecuteHopeSpend == nil:
		return status.Error(codes.Internal, "hope spend executor is not configured")
	case h.deps.AdvanceBreathCountdown == nil:
		return status.Error(codes.Internal, "breath countdown handler is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireDamageRollDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.ExecuteDamageRollResolve == nil:
		return status.Error(codes.Internal, "damage roll executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryRollDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.ExecuteAdversaryRollResolve == nil:
		return status.Error(codes.Internal, "adversary roll executor is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}

func (h *Handler) requireAdversaryActionCheckDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Event == nil:
		return status.Error(codes.Internal, "event store is not configured")
	case h.deps.SeedFunc == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	case h.deps.LoadAdversaryForSession == nil:
		return status.Error(codes.Internal, "adversary loader is not configured")
	default:
		return nil
	}
}
